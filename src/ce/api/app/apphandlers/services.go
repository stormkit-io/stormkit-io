package apphandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/deploy").Handler(shttp.MethodGet, "", handlerOneClickDeploy)

	s.NewEndpoint("/apps").
		Handler(shttp.MethodGet, "", user.WithAuth(handlerAppIndex))

	s.NewEndpoint("/app").
		Handler(shttp.MethodPut, "", app.WithApp(handlerAppUpdate)).
		Handler(shttp.MethodPost, "", user.WithAuth(handlerAppInsert)).
		Handler(shttp.MethodPost, "/webhooks/{provider:github|gitlab|bitbucket}", handlerInboundWebhooks).
		Handler(shttp.MethodPost, "/webhooks/{provider:github|gitlab|bitbucket}/{secret-id}", handlerInboundWebhooks). // Backwards compatibility
		Handler(shttp.MethodPost, "/proxy", app.WithApp(handlerAppProxy)).
		Handler(shttp.MethodDelete, "", app.WithApp(handlerAppDelete)).
		Handler(shttp.MethodGet, "/{did:[0-9]+}/settings", app.WithApp(handlerAppSettings)).
		Handler(shttp.MethodPut, "/deploy-trigger", app.WithApp(handlerAppDeployTriggerSet)).
		Handler(shttp.MethodDelete, "/{did:[0-9]+}/deploy-trigger", app.WithApp(handlerAppDeployTriggerDelete)).
		Handler(shttp.MethodGet, "/{did:[0-9]+}/outbound-webhooks", app.WithApp(handlerOutboundWebhookList)).
		Handler(shttp.MethodGet, "/{did:[0-9]+}/outbound-webhooks/{wid:[0-9]+}/trigger", app.WithApp(handlerOutboundWebhookSample)).
		Handler(shttp.MethodPost, "/outbound-webhooks", app.WithApp(handlerOutboundWebhookInsert)).
		Handler(shttp.MethodPut, "/outbound-webhooks", app.WithApp(handlerOutboundWebhookUpdate)).
		Handler(shttp.MethodDelete, "/outbound-webhooks", app.WithApp(handlerOutboundWebhookDelete))

	s.NewEndpoint("/hooks").
		Handler(shttp.MethodGet, "/app/{did:[0-9]+}/deploy/{hash}/{env}", app.WithAppNoAuth(handlerAppHooksDeploy)).
		Handler(shttp.MethodPost, "/app/{did:[0-9]+}/deploy/{hash}/{env}", app.WithAppNoAuth(handlerAppHooksDeploy))

	return s
}
