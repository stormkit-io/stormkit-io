package apphandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerAppDeployTriggerDelete(req *app.RequestContext) *shttp.Response {

	if err := app.NewStore().DeleteDeployTrigger(req.Context(), req.App.ID); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}
