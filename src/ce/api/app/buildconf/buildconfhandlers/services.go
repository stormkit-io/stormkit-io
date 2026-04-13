package buildconfhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	// Endpoints with environment name.
	s.NewEndpoint("/app/{did:[0-9]+}/envs").
		Handler(shttp.MethodGet, "/{env:[0-9a-zA-Z-]+}", app.WithApp(handlerEnv))

	s.NewEndpoint("/app/env").
		Handler(shttp.MethodDelete, "", app.WithApp(handlerEnvDelete)).
		Handler(shttp.MethodPost, "", app.WithApp(handlerEnvInsert))

	return s
}
