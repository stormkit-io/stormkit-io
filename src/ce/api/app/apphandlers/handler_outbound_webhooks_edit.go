package apphandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerOutboundWebhookUpdate(req *app.RequestContext) *shttp.Response {
	whReq := &outboundWebhookRequestData{}

	if err := req.Post(whReq); err != nil {
		return shttp.Error(err)
	}

	store := app.NewStore()
	wh := store.OutboundWebhook(req.Context(), req.App.ID, whReq.WebhookID)

	if wh == nil {
		return shttp.NotFound()
	}

	wh.RequestMethod = whReq.RequestMethod
	wh.RequestHeaders = whReq.RequestHeaders
	wh.RequestPayload = whReq.RequestPayload
	wh.RequestURL = whReq.RequestURL
	wh.TriggerWhen = whReq.TriggerWhen

	if err := app.NewStore().UpdateOutboundWebhook(req.Context(), req.App.ID, wh); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
