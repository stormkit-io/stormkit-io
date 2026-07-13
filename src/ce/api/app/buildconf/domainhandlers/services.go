package domainhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services registers the internal domain endpoints. Domain CRUD and custom
// certificates live in the public API (publicapiv1), but domain lookup /
// verification is an internal, cloud-only concern and stays here.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/domains").
		Handler(shttp.MethodGet, "/lookup", app.WithApp(handlerDomainLookup, &app.Opts{Env: true}))

	return s
}
