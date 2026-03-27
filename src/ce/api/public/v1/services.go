package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/domainhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
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
		Handler(shttp.MethodPost, "", WithAPIKey(handlerAppCreate, &Opts{MinimumScope: apikey.SCOPE_TEAM})).
		Handler(shttp.MethodGet, "", WithAPIKey(handlerAppGet, &Opts{MinimumScope: apikey.SCOPE_APP})).
		Handler(shttp.MethodGet, "/config", WithAPIKey(handlerAppConf, &Opts{MinimumScope: apikey.SCOPE_APP}))

	s.NewEndpoint("/v1/deploy").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerDeploymentCreate, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/deployments/{id:[0-9]+}").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerDeploymentGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/poll", WithAPIKey(handlerDeploymentPoll, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "/publish", WithAPIKey(handlerDeploymentPublish, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/env").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerEnvAdd, &Opts{MinimumScope: apikey.SCOPE_APP})).
		Handler(shttp.MethodPut, "", WithAPIKey(handlerEnvUpdate, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(handlerEnvDel, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/pull", WithAPIKey(handlerEnvPull, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/snippets").
		Handler(shttp.MethodGet, "", WithAPIKey(adaptAppHandler(snippetshandlers.HandlerSnippetsGet), &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "", WithAPIKey(adaptAppHandler(snippetshandlers.HandlerSnippetsAdd), &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPut, "", WithAPIKey(adaptAppHandler(snippetshandlers.HandlerSnippetsPut), &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(adaptAppHandler(snippetshandlers.HandlerSnippetsDelete), &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/redirects").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerRedirectsGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "", WithAPIKey(handlerRedirectsSet, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/domains").
		Handler(shttp.MethodGet, "", WithAPIKey(adaptAppHandler(domainhandlers.HandlerDomainsList), &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "", WithAPIKey(adaptAppHandler(domainhandlers.HandlerDomainAdd), &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(adaptAppHandler(domainhandlers.HandlerDomainDelete), &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/domains").
		Middleware(user.WithEE).
		Handler(shttp.MethodPut, "/cert", WithAPIKey(adaptAppHandler(domainhandlers.HandlerCertPut), &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "/cert", WithAPIKey(adaptAppHandler(domainhandlers.HandlerCertDelete), &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/mail").
		Handler(shttp.MethodPost, "", WithAPIKey(adaptAppHandler(mailerhandlers.HandlerMail), &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/volumes").
		Middleware(volumes.LimitRequestBody()).
		Handler(shttp.MethodPost, "", WithAPIKey(handlerVolumesPost, &Opts{MinimumScope: apikey.SCOPE_ENV}))

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
