package shttptest

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
)

// Response wraps httptest.ResponseRecorder
type Response struct {
	*httptest.ResponseRecorder
}

// String returns the response as a string.
func (r *Response) String() string {
	b := r.Byte()
	return strings.TrimSpace(string(b))
}

// Map returns the response as a map of string to any.
// This is useful when the response is expected to be a JSON object.
func (r *Response) Map() map[string]any {
	var m map[string]any

	if err := json.Unmarshal(r.Byte(), &m); err != nil {
		panic("Was expecting to unmarshal response data but could not")
	}

	return m
}

// String returns the response as an array of bytes.
func (r *Response) Byte() []byte {
	b, err := io.ReadAll(r.Body)

	if err != nil {
		panic(err)
	}

	return b
}

type UploadFile struct {
	Name string
	Data string
}

// MultipartForm prepares form data from the given fields. This function returns the request body,
// content-type header and an error.
func MultipartForm(fields map[string][]byte, files map[string][]UploadFile) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, uploadFiles := range files {
		for _, uploadFile := range uploadFiles {
			part, err := writer.CreateFormFile(k, uploadFile.Name)

			if err != nil {
				return nil, "", err
			}

			if _, err = part.Write([]byte(uploadFile.Data)); err != nil {
				return nil, "", err
			}
		}
	}

	for k, v := range fields {
		part, err := writer.CreateFormField(k)

		if err != nil {
			return nil, "", err
		}

		if _, err = part.Write(v); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return body, writer.FormDataContentType(), nil
}

// Request is used to test a generic endpoint.
func Request(h http.Handler, method, target string, body any) Response {
	return RequestWithHeaders(h, method, target, body, nil)
}

// RequestWithHeaders is used to test a generic endpoint.
func RequestWithHeaders(h http.Handler, method, target string, body any, headers map[string]string) Response {
	var httpBody io.Reader
	httpHeaders := make(http.Header)

	for k, v := range headers {
		httpHeaders.Add(k, v)
	}

	if httpHeaders.Get("Content-Type") == "" {
		httpHeaders.Set("Content-Type", "application/json")
	}

	if httpHeaders.Get("Content-Type") == "application/json" {
		data, err := json.Marshal(body)

		if err != nil {
			panic("Was expecting to marshal request data but could not")
		}

		httpBody = bytes.NewReader(data)
	}

	if strings.HasPrefix(httpHeaders.Get("Content-Type"), "multipart/form-data") {
		httpBody = body.(*bytes.Buffer)
	}

	r := httptest.NewRequest(method, target, httpBody)
	w := httptest.NewRecorder()

	r.Header = httpHeaders

	h.ServeHTTP(w, r)
	return Response{w}
}
