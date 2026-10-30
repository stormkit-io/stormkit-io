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
		Handler(shttp.MethodGet, "/config", app.WithApp(handlerAuthConfigGet, opts)).
		Handler(shttp.MethodPost, "/config", app.WithApp(handlerAuthConfigSet, opts))

	s.NewEndpoint("/auth-wall").
		Handler(shttp.MethodPost, "/login", shttp.WithRateLimit(handlerAuth, nil)).
		Handler(shttp.MethodPost, "/magic", shttp.WithRateLimit(handlerAuthMagic, nil)).
		Handler(shttp.MethodGet, "/magic", handlerAuthMagicContinue).
		Handler(shttp.MethodPost, "/magic/verify", shttp.WithRateLimit(handlerAuthMagicVerify, nil))

	s.NewEndpoint("/team/auth-wall").
		Middleware(user.WithEE).
		Handler(shttp.MethodGet, "/logins", user.WithAuth(handlerTeamLogins)).
		Handler(shttp.MethodPost, "/logins", user.WithAuth(handlerTeamLoginCreate)).
		Handler(shttp.MethodPatch, "/logins", user.WithAuth(handlerTeamLoginUpdate)).
		Handler(shttp.MethodDelete, "/logins", user.WithAuth(handlerTeamLoginsDelete)).
		Handler(shttp.MethodGet, "/envs", user.WithAuth(handlerTeamEnvs)).
		Handler(shttp.MethodPut, "/envs", user.WithAuth(handlerTeamEnvUpdate))

	return s
}
