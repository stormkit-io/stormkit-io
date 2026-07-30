package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/mailerhandlers"
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
		Handler(shttp.MethodGet, "", WithAPIKey(handlerAppGet, &Opts{MinimumScope: apikey.SCOPE_APP})).                // With API Key
		Handler(shttp.MethodGet, "/{appId:[0-9]+}", WithAPIKey(handlerAppGet, &Opts{MinimumScope: apikey.SCOPE_APP})). // Without API Key
		Handler(shttp.MethodGet, "/config", WithAPIKey(handlerAppConf, &Opts{MinimumScope: apikey.SCOPE_APP}))

	s.NewEndpoint("/v1/deploy").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerDeploymentCreate, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/deployments").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerDeploymentList, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/deployments/{id:[0-9]+}").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerDeploymentGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/poll", WithAPIKey(handlerDeploymentPoll, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/runtime-logs", WithAPIKey(handlerDeploymentRuntimeLogsGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "/publish", WithAPIKey(handlerDeploymentPublish, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "/prioritize", WithAPIKey(handlerDeploymentPrioritize, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "/restart", WithAPIKey(handlerDeploymentRestart, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "/stop", WithAPIKey(handlerDeploymentStop, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(handlerDeploymentDelete, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/env").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerEnvAdd, &Opts{MinimumScope: apikey.SCOPE_APP})).
		Handler(shttp.MethodPut, "", WithAPIKey(handlerEnvUpdate, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(handlerEnvDel, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/pull", WithAPIKey(handlerEnvPull, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/envs").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerEnvList, &Opts{MinimumScope: apikey.SCOPE_APP}))

	s.NewEndpoint("/v1/snippets").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerSnippetsGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "", WithAPIKey(handlerSnippetsAdd, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPut, "", WithAPIKey(handlerSnippetsPut, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(handlerSnippetsDelete, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/redirects").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerRedirectsGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "", WithAPIKey(handlerRedirectsSet, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/domains").
		Handler(shttp.MethodGet, "", WithAPIKey(HandlerDomainsList, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "", WithAPIKey(HandlerDomainAdd, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(HandlerDomainDelete, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/domains").
		Middleware(user.WithEE).
		Handler(shttp.MethodPut, "/cert", WithAPIKey(HandlerCertPut, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "/cert", WithAPIKey(HandlerCertDelete, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/trigger").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerFunctionTriggerCreate, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPatch, "", WithAPIKey(handlerFunctionTriggerUpdate, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodDelete, "", WithAPIKey(handlerFunctionTriggerDelete, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodPost, "/invoke", WithAPIKey(handlerFunctionTriggerInvoke, &Opts{MinimumScope: apikey.SCOPE_ENV})).
		Handler(shttp.MethodGet, "/logs", WithAPIKey(handlerFunctionTriggerLogsGet, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/triggers").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerFunctionTriggersGet, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/mail").
		Handler(shttp.MethodPost, "", app.WithAPIKey(mailerhandlers.HandlerMail, &app.Opts{Env: true}))

	s.NewEndpoint("/v1/volumes").
		Middleware(volumes.LimitRequestBody()).
		Handler(shttp.MethodPost, "", WithAPIKey(handlerVolumesPost, &Opts{MinimumScope: apikey.SCOPE_ENV}))

	s.NewEndpoint("/v1/teams").
		Handler(shttp.MethodGet, "", WithAPIKey(handlerTeamList, &Opts{MinimumScope: apikey.SCOPE_USER})).
		Handler(shttp.MethodPost, "", WithAPIKey(handlerTeamCreate, &Opts{MinimumScope: apikey.SCOPE_USER}))

	s.NewEndpoint("/v1/mcp").
		Handler(shttp.MethodPost, "", WithAPIKey(handlerMCP, &Opts{MinimumScope: apikey.SCOPE_USER})).
		Handler(shttp.MethodGet, "", WithAPIKey(handlerMCPStream, &Opts{MinimumScope: apikey.SCOPE_USER}))

	if authConfigEnabled() {
		s.NewEndpoint("/v1/auth/config").
			Handler(shttp.MethodGet, "", WithAPIKey(handlerAuthConfigGet, &Opts{MinimumScope: apikey.SCOPE_ENV})).
			Handler(shttp.MethodPost, "", WithAPIKey(handlerAuthConfigSet, &Opts{MinimumScope: apikey.SCOPE_ENV}))
	}

	if config.IsDevelopment() || config.IsSelfHosted() {
		s.NewEndpoint("/v1/schema").
			Handler(shttp.MethodGet, "", app.WithAPIKey(handlerSchemaGet, &app.Opts{Env: true})).
			Handler(shttp.MethodPost, "", app.WithAPIKey(handlerSchemaSet, &app.Opts{Env: true})).
			Handler(shttp.MethodDelete, "", app.WithAPIKey(handlerSchemaDelete, &app.Opts{Env: true})).
			Handler(shttp.MethodPost, "/configure", app.WithAPIKey(handlerSchemaConfigure, &app.Opts{Env: true}))
	}

	if config.IsStormkitCloud() {
		s.NewEndpoint("/v1/license").
			Handler(shttp.MethodGet, "/check", handlerLicenseCheck). // Backwards compatibility for old licenses, to be removed in the future.
			Handler(shttp.MethodPost, "/check", handlerLicenseCheck)
	}

	return s
}
