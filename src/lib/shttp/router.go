package shttp

import (
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
