package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentDelete(req *RequestContext) *shttp.Response {
	id := utils.StringToID(req.Vars()["id"])

	if id == 0 {
		return shttp.NotFound()
	}

	store := deploy.NewStore()

	depl, err := store.MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: id,
		EnvID:        req.Env.ID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	if err := store.MarkDeploymentsAsDeleted(req.Context(), []types.ID{id}); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(depl.EnvID); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"ok": true},
	}
}
