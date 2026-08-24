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

	// Always load the logs for this single deployment, even when the caller did
	// not ask for the full array: a failed deployment derives its inline
	// failureSummary from them. JSON(withLogs) still withholds the full array
	// unless it was requested.
	includeLogs := true

	depl, err := deploy.NewStore().MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: id,
		EnvID:        req.Env.ID,
		IncludeLogs:  &includeLogs,
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
