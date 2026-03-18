package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/domainhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/v1/apps").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerAppList, &Opts{MinimumScope: apikey.SCOPE_TEAM}))

	s.NewEndpoint("/v1/app").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerAppGet, &Opts{MinimumScope: apikey.SCOPE_APP})).
		Handler(shttp.MethodGet, "/config", WithAPIKey(handlerAppConf, &Opts{MinimumScope: apikey.SCOPE_APP}))

	s.NewEndpoint("/v1/deploy").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerDeploymentCreate, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/env").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerEnvAdd, &Opts{MinimumScope: apikey.SCOPE_APP})).
		Handler(shttp.MethodDelete, "", WithAPIKey(handlerEnvDel, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/pull", WithAPIKey(handlerEnvPull, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/snippets").
		Handler(shttp.MethodGet, "", app.WithAPIKey(snippetshandlers.HandlerSnippetsGet, &app.Opts{Env: true})).
		Handler(shttp.MethodPost, "", app.WithAPIKey(snippetshandlers.HandlerSnippetsAdd, &app.Opts{Env: true})).
		Handler(shttp.MethodPut, "", app.WithAPIKey(snippetshandlers.HandlerSnippetsPut, &app.Opts{Env: true})).
		Handler(shttp.MethodDelete, "", app.WithAPIKey(snippetshandlers.HandlerSnippetsDelete, &app.Opts{Env: true}))

	s.NewEndpoint("/v1/redirects").
		Handler(shttp.MethodGet, "", app.WithAPIKey(handlerRedirectsGet, &app.Opts{Env: true})).
		Handler(shttp.MethodPost, "", app.WithAPIKey(handlerRedirectsSet, &app.Opts{Env: true}))

	s.NewEndpoint("/v1/domains").
		Handler(shttp.MethodGet, "", app.WithAPIKey(domainhandlers.HandlerDomainsList, &app.Opts{Env: true})).
		Handler(shttp.MethodPost, "", app.WithAPIKey(domainhandlers.HandlerDomainAdd, &app.Opts{Env: true})).
		Handler(shttp.MethodDelete, "", app.WithAPIKey(domainhandlers.HandlerDomainDelete, &app.Opts{Env: true}))

	s.NewEndpoint("/v1/domains").
		Middleware(user.WithEE).
		Handler(shttp.MethodPut, "/cert", app.WithAPIKey(domainhandlers.HandlerCertPut, &app.Opts{Env: true})).
		Handler(shttp.MethodDelete, "/cert", app.WithAPIKey(domainhandlers.HandlerCertDelete, &app.Opts{Env: true}))

	s.NewEndpoint("/v1/mail").
		Handler(shttp.MethodPost, "", app.WithAPIKey(mailerhandlers.HandlerMail, &app.Opts{Env: true}))

	if config.IsDevelopment() || config.IsSelfHosted() {
		s.NewEndpoint("/v1/auth").
			Handler(shttp.MethodGet, "", HandlerAuthRedirect).
			Handler(shttp.MethodGet, "/session", HandlerSession).
			Handler(shttp.MethodGet, "/callback", HandlerAuthCallback)
	}

	if config.IsStormkitCloud() {
		s.NewEndpoint("/v1/license").
			// Temporary solution until we migrate previous licenses
			Handler(shttp.MethodGet, "", func(rc *shttp.RequestContext) *shttp.Response { return shttp.OK() }).
			Handler(shttp.MethodGet, "/check", handlerLicenseCheck)
	}

	return s
}
