package status

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services installs the api-status handlers.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()
	e := s.NewEndpoint("/")

	e.Handler(shttp.MethodGet, "", handlerAPIStatus)
	e.Handler(shttp.MethodHead, "", handlerAPIStatus)
	e.Handler(shttp.MethodGet, "health", handlerAPIHealth)

	return s
}

func handlerAPIHealth(req *shttp.RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
	}
}
