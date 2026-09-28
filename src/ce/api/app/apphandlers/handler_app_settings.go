package apphandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAppSettings returns the application settings.
func handlerAppSettings(req *app.RequestContext) *shttp.Response {
	settings, err := app.NewStore().Settings(req.Context(), req.App.ID)

	if err != nil {
		return shttp.Error(err)
	}

	cnf := admin.MustConfig()

	// With a deploy key, Bitbucket webhooks are registered by hand, so the
	// user needs the URL carrying this app's secret.
	if settings != nil && strings.HasPrefix(req.App.Repo, "bitbucket/") && cnf.AuthConfig != nil && cnf.AuthConfig.Bitbucket.DeployKey != "" {
		settings.InboundWebhook = cnf.ApiURL("/app/webhooks/bitbucket/" + req.App.Secret())
	}

	return &shttp.Response{
		Data: settings,
	}
}
