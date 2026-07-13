package apphandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerOutboundWebhookList(req *app.RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]interface{}{
			"webhooks": app.NewStore().OutboundWebhooks(req.Context(), req.App.ID),
		},
	}
}
