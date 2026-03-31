package shttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
)

// File is a struct that represent an uploaded file.
type File struct {
	Name     string
	Contents []byte
	Length   int64
	Error    error
}

// RequestContext is the context for the current request.
type RequestContext struct {
	*http.Request
	writer http.ResponseWriter

	// StartTime is the time when the request was first received.
	StartTime time.Time

	parsedURL url.Values
}

// NewRequestContext returns a new context object.
func NewRequestContext(req *http.Request) *RequestContext {
	if req == nil {
		req = &http.Request{}
	}

	return &RequestContext{
		Request: req,
	}
}

// Http methods
const (
	MethodPost    = http.MethodPost
	MethodGet     = http.MethodGet
	MethodPut     = http.MethodPut
	MethodDelete  = http.MethodDelete
	MethodOptions = http.MethodOptions
	MethodHead    = http.MethodHead
	MethodPatch   = http.MethodPatch
)

// SetWriter allows setting a different writer than http.ResponseWriter.
// It is mostly used for test purposes.
func (r *RequestContext) SetWriter(w http.ResponseWriter) {
	r.writer = w
}

// Writer returns the ResponseWriter object.
func (r *RequestContext) Writer() http.ResponseWriter {
	return r.writer
}

// Vars returns the route parameters.
func (r *RequestContext) Vars() map[string]string {
	if r.Request == nil {
		return map[string]string{}
	}

	return mux.Vars(r.Request)
}

// URL returns the current request's url.
func (r *RequestContext) URL() *url.URL {
	if r.Request == nil {
		return &url.URL{}
	}

	u := r.Request.URL

	// In case it is localhost, the scheme will be empty.
	if r.Request.TLS != nil {
		u.Scheme = "https"
	} else if u.Scheme == "" {
		u.Scheme = "http"
	}

	if u.Host == "" {
		u.Host = r.Request.Host
	}

	return u
}

// RemoteAddr returns the remote address. It first checks for X-Fowarded-*
// headers and if either the ip or the port is missing returns the request.RemoteAddr.
func (r *RequestContext) RemoteAddr() string {
	return RemoteAddr(r.Request)
}

// Query returns the query parameters.
func (r *RequestContext) Query() url.Values {
	if r.Request == nil {
		return url.Values{}
	}

	if r.parsedURL == nil && r.Request.URL != nil {
		r.parsedURL = r.Request.URL.Query()
	}

	return r.parsedURL
}

// ResetQuery clears the cached parsed query so that the next call to Query()
// re-parses Request.URL.RawQuery. Call this after mutating RawQuery directly.
func (r *RequestContext) ResetQuery() {
	r.parsedURL = nil
}

// Headers returns the request headers in a map[string]string.
func (r *RequestContext) Headers() http.Header {
	if r.Request == nil {
		return http.Header{}
	}

	return r.Request.Header
}

// HostName returns the host name from the request.
func (r *RequestContext) HostName() string {
	if r.Request == nil {
		return ""
	}

	// Check the X-Forwarded-Host header first (commonly used by proxies)
	host := r.Request.Header.Get("X-Forwarded-Host")

	// If X-Forwarded-Host is empty, use the Host header
	if host == "" {
		host = r.Request.Host
	}

	if host == "" {
		return r.Request.URL.Host
	}

	return host
}

// Post parses the request body and returns the post data.
func (r *RequestContext) Post(out any) error {
	if r.Request == nil || r.Request.Body == nil {
		return nil
	}

	contents, err := io.ReadAll(r.Request.Body)

	if err != nil {
		return err
	}

	defer func() {
		r.Request.Body = io.NopCloser(bytes.NewBuffer(contents))
	}()

	if err = json.Unmarshal(contents, out); err != nil {
		verr := &shttperr.ValidationError{}
		verr.SetError("error", fmt.Sprintf("Cannot unmarshal request: %s", err.Error()))
		return verr
	}

	if m, ok := out.(model.Model); ok {
		if errs := m.Validate(); errs != nil {
			return errs
		}
	}

	return nil
}

// UploadedFile returns
func (r *RequestContext) UploadedFile(key string) (*File, error) {
	var Buf bytes.Buffer
	file, header, err := r.Request.FormFile(key)

	if err != nil {
		return nil, err
	}

	defer file.Close()
	name := strings.Split(header.Filename, ".")

	f := &File{
		Name:     name[0],
		Contents: Buf.Bytes(),
	}

	f.Length, f.Error = io.Copy(&Buf, file)
	Buf.Reset()
	return f, nil
}

// Redirect redirects the url.
func (r *RequestContext) Redirect(url string, status int) {
	if status == 0 {
		status = http.StatusFound
	}

	http.Redirect(r.writer, r.Request, url, status)
}

// RemoteAddr returns the remote address. It first checks for X-Fowarded-*
// headers and if either the ip or the port is missing returns the request.RemoteAddr.
func RemoteAddr(r *http.Request) string {
	// Check the X-Forwarded-For header first (commonly used by proxies)
	addr := r.Header.Get("X-Forwarded-For")
	port := r.Header.Get("X-Forwarded-Port")

	// If X-Forwarded-For is empty, check the X-Real-IP header
	if addr == "" {
		addr = r.Header.Get("X-Real-IP")
	}

	if port == "" {
		port = r.Header.Get("X-Real-Port")
	}

	if addr == "" && port == "" {
		return r.RemoteAddr
	}

	if port == "" {
		return addr
	}

	return fmt.Sprintf("%s:%s", addr, port)
}
