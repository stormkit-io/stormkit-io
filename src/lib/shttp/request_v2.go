package shttp

import (
	"bytes"
	"context"
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

// dialAddrKey carries a dialOverride on a request's context.
type dialAddrKey struct{}

// dialOverride opens new connections for Target (host:port) at Addr instead.
// The URL, and with it the TLS server name and Host header, stay the target's.
// Other hosts the request reaches, such as a redirect, dial normally, though
// they may reuse a pooled loopback connection an earlier override opened for
// that same host.
type dialOverride struct {
	Target string
	Addr   string
}

// transportSet holds the connection pools requests go through. Requests with a
// dial override get their own pools, so a loopback connection is never handed
// to a request that should resolve its host through DNS.
type transportSet struct {
	dialer *net.Dialer
	dial   *http.Transport

	streamingMu sync.Mutex
	streaming   map[bool]*http.Transport
}

var transports = newTransportSet()

func newTransportSet() *transportSet {
	t := &transportSet{
		dialer:    &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second},
		streaming: map[bool]*http.Transport{},
	}

	t.dial = http.DefaultTransport.(*http.Transport).Clone()
	t.dial.DialContext = t.dialContext

	return t
}

// dialContext opens the connection at the request's override address when addr
// is the override's target, and at addr otherwise.
func (t *transportSet) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if o, ok := ctx.Value(dialAddrKey{}).(dialOverride); ok && o.Target == addr {
		addr = o.Addr
	}

	return t.dialer.DialContext(ctx, network, addr)
}

// base returns the pool for requests with or without a dial override.
func (t *transportSet) base(override bool) *http.Transport {
	if override {
		return t.dial
	}

	return http.DefaultTransport.(*http.Transport)
}

// streamingFor returns the streaming variant of the base pool. Streaming requests
// avoid an overall client timeout that would cut off large uploads and use
// ResponseHeaderTimeout instead, so the upstream must start responding within
// the deadline after the body is sent. Every caller uses the configured proxy
// timeout, so one variant per pool is safe to share.
func (t *transportSet) streamingFor(override bool, timeout time.Duration) *http.Transport {
	t.streamingMu.Lock()
	defer t.streamingMu.Unlock()

	if st, ok := t.streaming[override]; ok {
		return st
	}

	st := t.base(override).Clone()
	st.ResponseHeaderTimeout = timeout
	t.streaming[override] = st

	return st
}

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
	DialAddr(addr string) RequestInterface
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
	dialAddr              string
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
	client.dialAddr = ""
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

// DialAddr opens the connection to addr instead of the URL's host. The TLS
// server name and the Host header still come from the URL.
func (r *RequestV2) DialAddr(addr string) RequestInterface {
	r.dialAddr = addr
	return r
}

// withDialOverride returns req carrying a dialOverride for its own host and
// port, pointing at r.dialAddr.
func (r *RequestV2) withDialOverride(req *http.Request) *http.Request {
	port := req.URL.Port()

	if port == "" {
		port = "80"

		if req.URL.Scheme == "https" {
			port = "443"
		}
	}

	o := dialOverride{Target: net.JoinHostPort(req.URL.Hostname(), port), Addr: r.dialAddr}

	return req.WithContext(context.WithValue(req.Context(), dialAddrKey{}, o))
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

	if r.dialAddr != "" {
		req = r.withDialOverride(req)
	}

	override := r.dialAddr != ""
	client := &http.Client{Timeout: r.timeout}

	if r.bodyStream != nil && r.timeout > 0 {
		client = &http.Client{Transport: transports.streamingFor(override, r.timeout)}
	} else if override {
		client.Transport = transports.dial
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

	// DialAddr, when set, is where the connection to Target is opened instead
	// of Target's host. Used to reach a domain this process serves itself
	// without leaving the machine.
	DialAddr string
}

func Proxy(req *RequestContext, args ProxyArgs) *Response {
	headers := req.Header.Clone()

	// When Stormkit is the public edge (default), overwrite X-Forwarded-For with
	// the real socket address and overwrite X-Real-IP if the client supplied one,
	// so clients cannot spoof their IP for rate-limiting or access-control.
	// When STORMKIT_TRUST_PROXY_HEADERS=true a trusted upstream proxy (e.g. a
	// load balancer) has already set these headers correctly; pass them through
	// unchanged.
	if !config.Get().TrustProxyHeaders {
		if remoteAddr := req.RemoteAddr(); remoteAddr != "" {
			addr, port, err := net.SplitHostPort(remoteAddr)

			if err != nil {
				slog.Errorf("error splitting remote address: %s", remoteAddr)
			}

			if addr != "" {
				headers.Set("X-Forwarded-For", addr)

				if headers.Get("X-Real-IP") != "" {
					headers.Set("X-Real-IP", addr)
				}
			}

			if port != "" {
				headers.Set("X-Forwarded-Port", port)
			}
		}
	}

	// A loopback hop reaches the target from 127.0.0.1, so it carries the
	// visitor's address for the target to restore. A chain already set by a
	// trusted upstream proxy is passed on unchanged.
	if args.DialAddr != "" {
		if headers.Get("X-Forwarded-For") == "" && req.RemoteIP() != "" {
			headers.Set("X-Forwarded-For", req.RemoteIP())
		}

		if headers.Get("X-Forwarded-Port") == "" && req.RemotePort() != "" {
			headers.Set("X-Forwarded-Port", req.RemotePort())
		}
	}

	client := NewRequestV2(req.Method, args.Target).Headers(headers)

	if args.FollowRedirects != nil && !*args.FollowRedirects {
		client.FollowRedirects(false)
	}

	if args.DialAddr != "" {
		client.DialAddr(args.DialAddr)
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
