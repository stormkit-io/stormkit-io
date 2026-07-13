package shttp

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// Request represents an http request to be sent.
type Request struct {
	method        string
	headers       map[string]string
	payload       []byte
	query         map[string]string
	files         bytes.Buffer
	url           string
	insecure      bool
	checkRedirect func(*http.Request, []*http.Request) error
}

// HTTPResponse is a wrapper around http.Response struct.
type HTTPResponse struct {
	*http.Response
}

// String returns the string representation of the response.
func (hr *HTTPResponse) String() string {
	b, _ := io.ReadAll(hr.Body)
	return string(b)
}

func newRequest(method, url string) *Request {
	return &Request{
		method: method,
		url:    url,
		query:  nil,
		files:  *bytes.NewBuffer(nil),
		headers: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "api.stormkit.io",
		},
	}
}

// NewRequest returns a new request object.
func NewRequest(method, url string) *Request {
	return newRequest(method, url)
}

// Post creates a new Request instance. To send the prepared request
// use Request.Do method.
func Post(url string) *Request {
	return newRequest(MethodPost, url)
}

// Put creates a new Request instance. To send the prepared request
// use Request.Do method.
func Put(url string) *Request {
	return newRequest(MethodPut, url)
}

// Get creates a new Request instance. To send the prepared request
// use Request.Do method.
func Get(url string) *Request {
	return newRequest(MethodGet, url)
}

// Head creates a new Request instance. To send the prepared request
// use Request.Do method.
func Head(url string) *Request {
	return newRequest(MethodHead, url)
}

// Delete creates a new Request instance. To send the prepared request
// use Request.Do method.
func Delete(url string) *Request {
	return newRequest(MethodDelete, url)
}

// SetCheckRedirect sets the redirect function. In case the response returns a redirect,
// this function will be triggered.
func (r *Request) SetCheckRedirect(red func(*http.Request, []*http.Request) error) *Request {
	r.checkRedirect = red
	return r
}

// SetInsecure sets the connection either insecure or secure. Default is secure.
func (r *Request) SetInsecure(val bool) *Request {
	r.insecure = val
	return r
}

// Payload sets the payload for the request.
func (r *Request) Payload(payload interface{}) *Request {
	r.payload, _ = toByteArray(payload)
	return r
}

// Query sets the query parameters.
func (r *Request) Query(q map[string]string) *Request {
	r.query = q
	return r
}

// Headers sets the request headers.
func (r *Request) Headers(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers[k] = v
	}

	return r
}

// Attach attaches files to the request.
func (r *Request) Attach(values map[string]io.Reader) *Request {
	var err error
	w := multipart.NewWriter(&r.files)

	for key, reader := range values {
		var fw io.Writer

		if x, ok := reader.(io.Closer); ok {
			defer x.Close()
		}

		if x, ok := reader.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				slog.Error("Cannot create form file: ", err.Error())
				return r
			}
		} else if fw, err = w.CreateFormField(key); err != nil {
			slog.Error("Cannot create form field: ", err.Error())
			return r
		}

		if _, err = io.Copy(fw, reader); err != nil {
			slog.Error("Cannot copy file to writer: ", err.Error())
			return r
		}
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()
	r.headers["Content-Type"] = w.FormDataContentType()
	return r
}

// Do triggers a request.
func (r *Request) Do() (*HTTPResponse, error) {
	// Append payload to files buffer
	var buf *bytes.Buffer

	if r.files.Len() > 0 {
		buf = &r.files
	} else {
		buf = bytes.NewBuffer(r.payload)
	}

	// Create a new request
	req, err := http.NewRequest(r.method, r.url, buf)

	if err != nil {
		return nil, err
	}

	// Add query parameters, if any
	if r.query != nil {
		q := req.URL.Query()

		for _, k := range r.query {
			q.Add(k, r.query[k])
		}

		req.URL.RawQuery = q.Encode()
	}

	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: r.checkRedirect,
	}

	if r.insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	res, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	return &HTTPResponse{res}, nil
}

// toByteArray casts the given interface value into an array of bytes.
func toByteArray(v interface{}) ([]byte, error) {
	switch data := v.(type) {
	case io.ReadCloser:
		return io.ReadAll(data)
	case string:
		return []byte(data), nil

	case []byte:
		return data, nil

	default:
		return json.Marshal(v)
	}
}

// TimeAuth returns the string for time authentication.
func TimeAuth() string {
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().Unix()))
	encTime, _ := utils.Encrypt(timeBytes)
	return "Stormkit " + utils.EncodeToString(encTime)
}
