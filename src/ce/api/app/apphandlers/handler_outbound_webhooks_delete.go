package apphandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type outboundWebhookDeleteRequestData struct {
	WebhookID types.ID `json:"whId,string"`
}

func handlerOutboundWebhookDelete(req *app.RequestContext) *shttp.Response {
	whReq := &outboundWebhookDeleteRequestData{}

	if err := req.Post(whReq); err != nil {
		return shttp.Error(err)
	}

	if err := app.NewStore().DeleteOutboundWebhook(req.Context(), req.App.ID, whReq.WebhookID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
