package maintenancehandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service. Maintenance mode is available
// on all editions, so the endpoints are not gated behind a license check.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	opts := &app.Opts{Env: true}

	s.NewEndpoint("/maintenance").
		Handler(shttp.MethodGet, "/config", app.WithApp(handlerConfigGet, opts)).
		Handler(shttp.MethodPost, "/config", app.WithApp(handlerConfigSet, opts))

	return s
}
