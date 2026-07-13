package authwallhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	opts := &app.Opts{Env: true}

	s.NewEndpoint("/auth-wall").
		Middleware(user.WithEE).
		Handler(shttp.MethodGet, "", app.WithApp(handlerAuthLogins, opts)).
		Handler(shttp.MethodPost, "", app.WithApp(handlerAuthCreate, opts)).
		Handler(shttp.MethodDelete, "", app.WithApp(handlerAuthDelete, opts)).
		Handler(shttp.MethodGet, "/config", app.WithApp(handlerAuthConfigGet, opts)).
		Handler(shttp.MethodPost, "/config", app.WithApp(handlerAuthConfigSet, opts))

	s.NewEndpoint("/auth-wall").
		Handler(shttp.MethodPost, "/login", shttp.WithRateLimit(handlerAuth, nil))

	return s
}
