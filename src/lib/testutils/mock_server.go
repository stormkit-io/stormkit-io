package testutils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// MockServerInstance represents a mock server instance.
type MockServerInstance struct {
	ts *httptest.Server

	// Responses holds the array of mock responses given the url.
	responses map[string]*MockResponse

	// OnClose is a list of listeners that will be trigger by the close function.
	onClose []func()

	// The close function that needs to be deferred after
	// creating a new server instance.
	Close func()
}

// MockResponse represents a mock response instance.
type MockResponse struct {
	Status        int
	Headers       http.Header
	Data          interface{}
	DataText      string
	Method        string
	Expect        func(r *http.Request)
	OnCall        func()
	NumberOfCalls int
	isCalled      bool
}

// MockServer creates a new mock server instance.
func MockServer() *MockServerInstance {
	ms := &MockServerInstance{}

	ms.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.String()

		for u, res := range ms.responses {
			pieces := strings.Split(u, ":")
			method, uri := pieces[0], pieces[1]

			if url != uri {
				continue
			}

			if strings.ToLower(method) != strings.ToLower(r.Method) {
				continue
			}

			res.isCalled = true
			res.NumberOfCalls = res.NumberOfCalls + 1

			if res.Expect != nil {
				res.Expect(r)
			}

			if res.OnCall != nil {
				res.OnCall()
			}

			if res.Headers != nil {
				for k, v := range res.Headers {
					for _, h := range v {
						w.Header().Add(k, h)
					}
				}
			}

			w.WriteHeader(res.Status)

			if res.Data != nil {
				json.NewEncoder(w).Encode(res.Data)
			} else if res.DataText != "" {
				w.Write([]byte(res.DataText))
			}

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	ms.Close = func() {
		for url, res := range ms.responses {
			if res.isCalled == false {
				panic(fmt.Sprintf("Response for %s is not called", url))
			}
		}

		ms.ts.Close()

		for _, c := range ms.onClose {
			c()
		}
	}

	return ms
}

// OnClose adds a new onClose listener.
func (ms *MockServerInstance) OnClose(fn func()) {
	if ms.onClose == nil {
		ms.onClose = []func(){}
	}

	ms.onClose = append(ms.onClose, fn)
}

// Client returns the client instance.
func (ms *MockServerInstance) Client() *http.Client {
	return ms.ts.Client()
}

// URL returns the mock server url.
func (ms *MockServerInstance) URL() string {
	return ms.ts.URL
}

// NewResponse adds a new mock response.
func (ms *MockServerInstance) NewResponse(url string, mr *MockResponse) {
	if ms.responses == nil {
		ms.responses = map[string]*MockResponse{}
	}

	if mr.Method == "" {
		mr.Method = http.MethodGet
	}

	ms.responses[mr.Method+":"+url] = mr
}
