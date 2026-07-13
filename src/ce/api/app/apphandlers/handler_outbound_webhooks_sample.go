package apphandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerOutboundWebhookSample(req *app.RequestContext) *shttp.Response {
	wh := app.NewStore().OutboundWebhook(req.Context(), req.App.ID, utils.StringToID(req.Vars()["wid"]))

	if wh == nil {
		return shttp.NotFound()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: wh.Dispatch(app.OutboundWebhookSettings{
			AppID:                  req.App.ID,
			DeploymentID:           req.App.ID,
			EnvironmentName:        config.AppDefaultEnvironmentName,
			DeploymentEndpoint:     "https://www.stormkit.io/examples/deployment",
			DeploymentLogsEndpoint: "https://www.stormkit.io/examples/deployment/logs",
			DeploymentStatus:       "success",
		}),
	}
}
