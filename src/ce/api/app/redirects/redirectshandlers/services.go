package redirectshandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/redirects").
		Handler(shttp.MethodPost, "/playground", app.WithApp(handlerPlayground, &app.Opts{Env: true}))

	return s
}
