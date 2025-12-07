package schemahandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/schema").
		Handler(shttp.MethodGet, "", app.WithApp(handlerSchemaGet, &app.Opts{Env: true})).
		Handler(shttp.MethodPost, "", app.WithApp(handlerSchemaSet, &app.Opts{Env: true}))

	return s
}
