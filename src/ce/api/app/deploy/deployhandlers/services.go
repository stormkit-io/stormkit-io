package deployhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/my").
		Handler(shttp.MethodGet, "/deployments", user.WithAuth(handlerMyDeployments))

	s.NewEndpoint("/app/deploy").
		Handler(shttp.MethodPost, "", app.WithApp(handlerDeployStart)).
		Handler(shttp.MethodDelete, "", app.WithApp(handlerDeployDelete)).
		Handler(shttp.MethodPost, "/callback", handlerDeployCallback)

	s.NewEndpoint("/app/{did:[0-9]+}/manifest").
		Handler(shttp.MethodGet, "/{deploymentId:[0-9]+}", shttp.WithRateLimit(
			app.WithApp(handlerDeployManifestGet),
			nil,
		))

	s.NewEndpoint("/app/deployments").
		Handler(shttp.MethodPost, "/publish", shttp.WithRateLimit(
			app.WithApp(handlerPublish),
			nil,
		))

	return s
}
