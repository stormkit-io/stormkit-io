package apploghandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/app/{did:[0-9]+}").
		Handler(shttp.MethodGet, "/logs", app.WithApp(handlerLogsGet))

	return s
}
