package apphandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAppSettings returns the application settings.
func handlerAppSettings(req *app.RequestContext) *shttp.Response {
	settings, err := app.NewStore().Settings(req.Context(), req.App.ID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: settings,
	}
}
