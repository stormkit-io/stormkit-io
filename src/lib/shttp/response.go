package shttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

type ServeContent struct {
	Content io.ReadSeeker
	Name    string
	ModTime time.Time
}

// Response is the http response.
type Response struct {
	// Status is the status code.
	Status int

	// Data is the payload to return.
	// It can be either a string or a map.
	Data any

	// Headers are the response headers.
	Headers http.Header

	// Cookies are used to set cookies.
	Cookies []http.Cookie

	// Error is the error that will be logged.
	Error error

	// ServeContent specifies whether the content should be served with ServeContent
	// or regular write. If provided, http.ServeContent will be used to serve the content.
	ServeContent *ServeContent

	Redirect *string

	BeforeClose func()
}

// SetError is a helper function to be used in chaining.
// It sets the error.
func (r *Response) SetError(err error) *Response {
	r.Error = err
	return r
}

// String returns the string representation of a response.
func (r *Response) String() string {
	data, _ := json.Marshal(r.Data)
	return string(data)
}

// NotFound returns a not found response.
func NotFound() *Response {
	return &Response{
		Status: http.StatusNotFound,
	}
}

// BadRequest returns a bad request response.
func BadRequest(data ...map[string]any) *Response {
	var responseData map[string]any

	if len(data) > 0 {
		responseData = data[0]
	}

	return &Response{
		Status: http.StatusBadRequest,
		Data:   responseData,
	}
}

// OK returns an ok response.
func OK() *Response {
	return &Response{
		Status: http.StatusOK,
		Data: map[string]bool{
			"ok": true,
		},
	}
}

// Created returns a status created response
func Created() *Response {
	return &Response{
		Status: http.StatusCreated,
	}
}

// NotOK return a ok: false response.
func NotOK() *Response {
	return &Response{
		Status: http.StatusOK,
		Data: map[string]bool{
			"ok": false,
		},
	}
}

// Backoff tells the client to retry in given seconds as
// there are too many requests right now.
func Backoff(seconds time.Duration) *Response {
	return &Response{
		Status: http.StatusTooManyRequests,
		Data: map[string]time.Duration{
			"retry": seconds,
		},
	}
}

// NoContent returns a no-content status.
func NoContent() *Response {
	return &Response{
		Status: http.StatusNoContent,
	}
}

// NotAllowed returns a 401 response with user data set to false.
func NotAllowed() *Response {
	return &Response{
		Status: http.StatusUnauthorized,
		Data: map[string]bool{
			"ok":   false,
			"user": false,
		},
	}
}

// Forbidden returns a 403 response with empty data.
func Forbidden() *Response {
	return &Response{
		Status: http.StatusForbidden,
	}
}

// Gone returns a 410 response with empty data.
func Gone() *Response {
	return &Response{
		Status: http.StatusGone,
	}
}

func Error(err error, slogMessage ...string) *Response {
	if len(slogMessage) > 0 {
		slog.Error(slogMessage[0])
	}

	if serr, ok := err.(*shttperr.Error); ok {
		status := serr.Status()

		if status/500 >= 1 {
			slog.Error(serr.OriginalError)
		}

		return &Response{
			Status: status,
			Error:  serr.OriginalError,
			Data: struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}{serr.Error(), serr.Code()},
		}
	}

	if serr, ok := err.(*shttperr.ValidationError); ok {
		return &Response{
			Status: http.StatusBadRequest,
			Data: struct {
				Errors map[string]string `json:"errors"`
			}{serr.Errors},
		}
	}

	_, file, no, _ := runtime.Caller(1)

	return UnexpectedError(err, fmt.Sprintf("%s:%d", file, no))
}

// ErrorMap is a map which contains all errors from validating a struct.
type ErrorMap map[string]ErrorArray

// ErrorMap implements the Error interface so we can check error against nil.
// The returned error is all existing errors with the map.
func (err ErrorMap) Error() string {
	var b bytes.Buffer

	for k, errs := range err {
		if len(errs) > 0 {
			b.WriteString(fmt.Sprintf("%s: %s, ", k, errs.Error()))
		}
	}

	return strings.TrimSuffix(b.String(), ", ")
}

// ErrorArray is a slice of errors returned by the Validate function.
type ErrorArray []error

// ErrorArray implements the Error interface and returns all the errors comma seprated
// if errors exist.
func (err ErrorArray) Error() string {
	var b bytes.Buffer

	for _, errs := range err {
		b.WriteString(fmt.Sprintf("%s, ", errs.Error()))
	}

	errs := b.String()
	return strings.TrimSuffix(errs, ", ")
}

// ValidationError prepares the validation errors in a user-friendly way
// and returns the response object with populated data.
func ValidationError(err error) *Response {
	res := &Response{}
	errors := make(map[string]string)
	res.Status = http.StatusBadRequest
	res.Data = map[string]interface{}{
		"ok": false,
	}

	// Gather errors into a map. If the errors is not an instance
	// of ErrorMap, then something else is wrong and we need debug it.
	if errMap, ok := err.(ErrorMap); ok {
		for field, err := range errMap {
			errors[strings.ToLower(field[:1])+field[1:]] = err.Error()
		}

		mp, _ := res.Data.(map[string]interface{})
		mp["errors"] = errors
	} else if valErr, ok := err.(*shttperr.ValidationError); ok {
		mp, _ := res.Data.(map[string]interface{})
		mp["errors"] = valErr.Errors
	} else {
		res.Error = err
	}

	return res
}

// UnexpectedError prints a user-friendly error to the end-user
// and it logs the error.
func UnexpectedError(err error, caller ...string) *Response {
	return &Response{
		Status: http.StatusInternalServerError,
		Error:  err,
		Data: map[string]interface{}{
			"ok":    false,
			"error": "unexpected-error",
		},
	}
}

// DuplicateKey returns a duplicate-error error to the client.
func DuplicateKey() *Response {
	return &Response{
		Status: http.StatusBadRequest,
		Data: map[string]interface{}{
			"ok":    false,
			"error": "duplicate-key",
		},
	}
}
