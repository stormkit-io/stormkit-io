package snippetshandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/snippets").
		Handler(shttp.MethodGet, "", app.WithApp(HandlerSnippetsGet, &app.Opts{Env: true})).
		Handler(shttp.MethodPost, "", app.WithApp(HandlerSnippetsAdd, &app.Opts{Env: true})).
		Handler(shttp.MethodPut, "", app.WithApp(HandlerSnippetsPut, &app.Opts{Env: true})).
		Handler(shttp.MethodDelete, "", app.WithApp(HandlerSnippetsDelete, &app.Opts{Env: true}))

	return s
}
