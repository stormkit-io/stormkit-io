package apphandlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type outboundWebhookRequestData struct {
	model.Model
	app.OutboundWebhook

	// WebhookID is used only for PUT requests
	WebhookID types.ID `json:"whId,string"`
}

// Validate implements model.Validate interface.
func (wh *outboundWebhookRequestData) Validate() *shttperr.ValidationError {
	err := &shttperr.ValidationError{}
	allowed := []string{
		app.TriggerOnDeploySuccess,
		app.TriggerOnDeployFailed,
		app.TriggerOnPublish,
		app.TriggerOnCachePurge,
	}

	// Backwards compatibility
	if wh.TriggerWhen == "on_deploy" {
		wh.TriggerWhen = app.TriggerOnDeploySuccess
	}

	if !utils.InSliceString(allowed, wh.TriggerWhen) {
		err.SetError("triggerWhen", fmt.Sprintf("Invalid triggerWhen value. Accepted values are: %s", strings.Join(allowed, " | ")))
	}

	if wh.RequestMethod != shttp.MethodPost && wh.RequestMethod != shttp.MethodGet && wh.RequestMethod != shttp.MethodHead {
		err.SetError("requestMethod", fmt.Sprintf("Invalid requestMethod value. Accepted values are: %s | %s | %s", shttp.MethodPost, shttp.MethodGet, shttp.MethodHead))
	}

	if _, uerr := url.ParseRequestURI(wh.RequestURL); uerr != nil {
		err.SetError("requesUrl", uerr.Error())
	}

	return err.ToError()
}

func handlerOutboundWebhookInsert(req *app.RequestContext) *shttp.Response {
	whReq := &outboundWebhookRequestData{}

	if err := req.Post(whReq); err != nil {
		return shttp.Error(err)
	}

	settings, err := app.NewStore().Settings(req.Context(), req.App.ID)

	if err != nil {
		slog.Errorf("error while fetching settings: %v", err)
		return shttp.Error(err)
	}

	if settings.DeployTrigger != "" {
		cnf := admin.MustConfig()

		partialTriggerUrl := cnf.ApiURL(
			fmt.Sprintf("/hooks/app/%d/deploy/%s/",
				req.App.ID,
				settings.DeployTrigger,
			),
		)

		if strings.Contains(whReq.RequestURL, partialTriggerUrl) {
			return shttp.Error(shttperr.New(http.StatusBadRequest, "Can't use Trigger deploy link as outbound request", ""))
		}
	}

	if err := app.NewStore().InsertOutboundWebhook(req.Context(), req.App.ID, whReq.OutboundWebhook); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
