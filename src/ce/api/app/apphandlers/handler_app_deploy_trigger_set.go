package apphandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerAppDeployTriggerSet generates a new token for the deploy trigger.
func handlerAppDeployTriggerSet(req *app.RequestContext) *shttp.Response {
	trigger := strings.ToLower(utils.RandomToken(48))

	if err := app.NewStore().UpdateDeployTrigger(req.Context(), req.App.ID, trigger); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]string{
			"hash": trigger,
		},
	}
}
