package shttp

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// Router represents an api router.
type Router struct {
	mux     *mux.Router
	handler http.Handler
}

// NewRouter creates a new service instance.
func NewRouter() *Router {
	return &Router{
		mux: mux.NewRouter(),
	}
}

// NewService returns a router for the given endpoint.
func (r *Router) NewService() *Service {
	return &Service{router: r}
}

// RegisterService registers the given service handler.
func (r *Router) RegisterService(s ServiceFunc) *Service {
	return s(r)
}

// RegisterMiddleware adds support for the third-party packages to register
// their own middlewares.
func (r *Router) RegisterMiddleware(handler func(h http.Handler) http.Handler) {
	if r.handler != nil {
		r.handler = handler(r.handler)
	} else {
		r.handler = handler(r.mux)
	}
}

// WithJSONErrors makes unmatched routes answer with the same structured JSON
// body as the handlers do. Without it gorilla/mux falls back to net/http, which
// writes "404 page not found" as text/plain — unparseable for an API client.
func (r *Router) WithJSONErrors() *Router {
	r.mux.NotFoundHandler = jsonErrorHandler(APIErrorParams{
		Status:  http.StatusNotFound,
		Code:    "unknown-endpoint",
		Message: "This endpoint does not exist. See the API documentation for the available endpoints.",
		Docs:    DocsAuthenticationURL,
	})

	r.mux.MethodNotAllowedHandler = jsonErrorHandler(APIErrorParams{
		Status:  http.StatusMethodNotAllowed,
		Code:    "method-not-allowed",
		Message: "This endpoint does not accept the used HTTP method. See the API documentation for the accepted methods.",
		Docs:    DocsAuthenticationURL,
	})

	return r
}

func jsonErrorHandler(p APIErrorParams) http.Handler {
	body, _ := json.Marshal(APIErrorBody{Error: p.Message, Code: p.Code, Docs: p.Docs})

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(p.Status)
		_, _ = w.Write(body)
	})
}

// WithContext enables the context handler.
func (r *Router) WithContext() *Router {
	r.RegisterMiddleware(contextHandler)
	return r
}

// WithGzip enables gzipped responses.
func (r *Router) WithGzip() *Router {
	r.RegisterMiddleware(gzipHandler)
	return r
}

// Handler returns the handler.
func (r *Router) Handler() http.Handler {
	if r.handler == nil {
		r.handler = func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
			})
		}(r.mux)
	}

	return r.handler
}
