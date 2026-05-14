package userhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services installs the user services.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()
	ep := s.NewEndpoint("/user")

	ep.
		Handler(shttp.MethodGet, "", user.WithAuth(handlerUserSession)).
		Handler(shttp.MethodGet, "/emails", user.WithAuth(handlerUserEmails)).
		Handler(shttp.MethodPut, "/access-token", user.WithAuth(handlerUpdatePersonalAccessToken)).
		Handler(shttp.MethodPost, "/shared-session", user.WithAuth(handlerUserSharedSession)).
		Handler(shttp.MethodDelete, "", user.WithAuth(handlerUserDelete))

	return s
}
