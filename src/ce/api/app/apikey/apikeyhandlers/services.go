package apikeyhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	// Endpoints with environment id.
	// Try to move previous endpoints to this endpoint
	s.NewEndpoint("/api-keys").
		Handler(shttp.MethodPost, "", user.WithAuth(handlerAPIKeyAdd)).
		Handler(shttp.MethodGet, "", user.WithAuth(handlerAPIKeyGet)).
		Handler(shttp.MethodDelete, "", user.WithAuth(handlerAPIKeyRemove))

	return s
}
