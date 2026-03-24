package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerDeploymentPoll returns a lightweight status for a deployment.
// Possible values mirror Deployment.Status(): "running", "success", or "failed".
func handlerDeploymentPoll(req *RequestContext) *shttp.Response {
	id := utils.StringToID(req.Vars()["id"])

	if id == 0 {
		return shttp.NotFound()
	}

	depl, err := deploy.NewStore().DeploymentStatus(req.Context(), id, req.Env.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"status": depl.Status()},
	}
}
