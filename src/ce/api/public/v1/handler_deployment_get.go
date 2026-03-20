package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentGet(req *RequestContext) *shttp.Response {
	id := utils.StringToID(req.Vars()["id"])

	if id == 0 {
		return shttp.NotFound()
	}

	withLogs := req.Query().Get("logs") == "true"

	depl, err := deploy.NewStore().MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: id,
		EnvID:        req.Env.ID,
		IncludeLogs:  &withLogs,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"deployment": depl.JSON(withLogs),
		},
	}
}
