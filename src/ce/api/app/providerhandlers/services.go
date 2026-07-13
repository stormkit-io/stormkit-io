package providerhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/provider/{provider:github|gitlab|bitbucket}").
		Handler(shttp.MethodGet, "/repos", user.WithAuth(handlerRepoList)).
		Handler(shttp.MethodGet, "/accounts", user.WithAuth(handlerAccountList))

	return s
}
