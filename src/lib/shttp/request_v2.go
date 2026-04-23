package shttp

import (
	"bytes"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var DefaultRequest RequestInterface

func HeadersFromMap(m map[string]string) http.Header {
	headers := make(http.Header)

	for k, v := range m {
		headers.Add(k, v)
	}

	return headers
}

var (
	streamingTransport     *http.Transport
	streamingTransportOnce sync.Once
)

var clientPool = sync.Pool{
	New: func() any {
		return &RequestV2{}
	},
}

type RequestInterface interface {
	URL(url string) RequestInterface
	Method(method string) RequestInterface
	Payload(payload any) RequestInterface
	Stream(body io.Reader, contentLength int64) RequestInterface
	Headers(headers http.Header) RequestInterface
	WithExponentialBackoff(maxDelay time.Duration, maxRetries int) RequestInterface
	WithTimeout(duration time.Duration) RequestInterface
	FollowRedirects(bool) RequestInterface
	Do() (*HTTPResponse, error)
}

// Request represents an http request to be sent.
type RequestV2 struct {
	timeout               time.Duration
	method                string
	payload               []byte
	bodyStream            io.Reader
	contentLength         int64
	headers               http.Header
	url                   string
	followRedirects       bool
	backoffCurrentDelay   time.Duration
	backoffMaxDelay       time.Duration
	backoffMaxRetries     int
	backoffCurrentAttempt int
}

// NewRequest returns a new request object.
func NewRequestV2(method, url string) RequestInterface {
	if DefaultRequest != nil {
		return DefaultRequest.URL(url).Method(method)
	}

	client := clientPool.Get().(*RequestV2)
	client.method = method
	client.url = url
	client.payload = nil
	client.bodyStream = nil
	client.contentLength = 0
	client.headers = nil
	client.followRedirects = true
	client.timeout = config.Get().HTTPTimeouts.ProxyTimeout

	return client
}

func (r *RequestV2) WithTimeout(duration time.Duration) RequestInterface {
	r.timeout = duration
	return r
}

func (r *RequestV2) WithExponentialBackoff(maxDelay time.Duration, maxRetries int) RequestInterface {
	r.backoffCurrentDelay = time.Second * 1
	r.backoffMaxDelay = maxDelay
	r.backoffMaxRetries = maxRetries
	return r
}

func (r *RequestV2) Headers(headers http.Header) RequestInterface {
	r.headers = headers
	return r
}

// Payload sets the payload for the request.
func (r *RequestV2) Payload(payload any) RequestInterface {
	r.payload, _ = toByteArray(payload)
	return r
}

// Stream sets a streaming body for the request, bypassing buffering.
func (r *RequestV2) Stream(body io.Reader, contentLength int64) RequestInterface {
	r.bodyStream = body
	r.contentLength = contentLength
	return r
}

// Method sets the request method.
func (r *RequestV2) Method(method string) RequestInterface {
	r.method = method
	return r
}

// URL sets the request URL.
func (r *RequestV2) URL(url string) RequestInterface {
	r.url = url
	return r
}

func (r *RequestV2) FollowRedirects(v bool) RequestInterface {
	r.followRedirects = v
	return r
}

// Do triggers a request.
func (r *RequestV2) Do() (*HTTPResponse, error) {
	body := io.Reader(bytes.NewBuffer(r.payload))

	if r.bodyStream != nil {
		body = r.bodyStream
	}

	req, err := http.NewRequest(r.method, r.url, body)

	if err != nil {
		return nil, err
	}

	if r.bodyStream != nil && r.contentLength > 0 {
		req.ContentLength = r.contentLength
	}

	req.Header = r.headers

	var client *http.Client

	if r.bodyStream != nil && r.timeout > 0 {
		// For streaming requests, avoid an overall client timeout that would
		// cut off large uploads. Use ResponseHeaderTimeout instead so the
		// upstream must start sending response headers within the deadline
		// after the request body is fully sent. The transport is intentionally
		// shared across all streaming requests: all callers use the same
		// configured proxy timeout, so a single transport is safe to reuse.
		streamingTransportOnce.Do(func() {
			t := http.DefaultTransport.(*http.Transport).Clone()
			t.ResponseHeaderTimeout = r.timeout
			streamingTransport = t
		})

		client = &http.Client{Transport: streamingTransport}
	} else {
		client = &http.Client{Timeout: r.timeout}
	}

	if !r.followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	res, err := client.Do(req)

	if err == nil && res != nil {
		clientPool.Put(r)
		return &HTTPResponse{res}, nil
	}

	if r.backoffCurrentDelay > 0 && r.backoffMaxRetries > r.backoffCurrentAttempt {
		r.backoffCurrentAttempt = r.backoffCurrentAttempt + 1
		r.backoffCurrentDelay = time.Duration(
			math.Min(
				float64(r.backoffMaxDelay),
				float64(r.backoffCurrentDelay)*math.Pow(2, float64(r.backoffCurrentAttempt)),
			),
		)

		slog.Infof("request failed, retrying in: %v, current retry: %d, endpoint: %s", r.backoffCurrentDelay, r.backoffCurrentAttempt, r.url)
		time.Sleep(r.backoffCurrentDelay)
		return r.Do()
	}

	if err != nil {
		return nil, err
	}

	return nil, errors.New("response object is empty")
}

type ProxyArgs struct {
	Target          string
	FollowRedirects *bool
}

func Proxy(req *RequestContext, args ProxyArgs) *Response {
	headers := req.Header.Clone()

	// Make sure X-Forwarded-For headers are set
	if headers.Get("X-Forwarded-For") == "" && headers.Get("X-Real-IP") == "" {
		if remoteAddr := req.RemoteAddr(); remoteAddr != "" {
			addr, port, err := net.SplitHostPort(remoteAddr)

			if err != nil {
				slog.Infof("error splitting remote address: %s", remoteAddr)
			}

			if addr != "" {
				headers.Set("X-Forwarded-For", addr)
			}

			if port != "" {
				headers.Set("X-Forwarded-Port", port)
			}
		}
	}

	client := NewRequestV2(req.Method, args.Target).Headers(headers)

	if args.FollowRedirects != nil && !*args.FollowRedirects {
		client.FollowRedirects(false)
	}

	if req.Body != nil {
		client.Stream(req.Body, req.ContentLength)
	}

	response, err := client.Do()

	if err == nil {
		var data []byte

		if response.Body != nil {
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)

			if err != nil {
				return &Response{
					Status: http.StatusInternalServerError,
					Data:   err.Error(),
					Error:  err,
				}
			}

			data = body
		}

		return &Response{
			Status:  response.StatusCode,
			Data:    data,
			Headers: response.Header,
		}
	}

	return &Response{
		Status: http.StatusInternalServerError,
		Data:   err.Error(),
		Error:  err,
	}
}
