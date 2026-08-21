package mailerhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
//
// Only the sent-email list lives here. Reading and writing the mailer
// configuration and sending mail are served by the public API, which the
// dashboard calls directly. The list keeps a session-authenticated route of
// its own because the dashboard renders message bodies, which the public API
// withholds.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/mailer").
		Handler(shttp.MethodGet, "", app.WithApp(handlerMailList, &app.Opts{Env: true}))

	return s
}
