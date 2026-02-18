package mailerhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/mailer").
		Handler(shttp.MethodPost, "", app.WithApp(HandlerMail, &app.Opts{Env: true})).
		Handler(shttp.MethodGet, "/config", app.WithApp(HandlerMailerConfigGet, &app.Opts{Env: true})).
		Handler(shttp.MethodPost, "/config", app.WithApp(HandlerMailerConfigSet, &app.Opts{Env: true}))

	return s
}
