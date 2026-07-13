package dedicatedhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/dedicated").
		Handler(shttp.MethodPost, "/resource", user.WithAuth(handlerCreateResource)).
		Handler(shttp.MethodGet, "/resources", user.WithAuth(handlerListResources))

	return s
}
