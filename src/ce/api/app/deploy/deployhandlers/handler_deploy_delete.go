package deployhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type deployDeleteRequest struct {
	DeploymentID types.ID `json:"deploymentId,string"`
}

// handlerDeployDelete deletes a deployment and all of the associated
// artifacts. These artifacts are the lambda version and s3 files.
func handlerDeployDelete(req *app.RequestContext) *shttp.Response {
	ddr := &deployDeleteRequest{}

	if err := req.Post(ddr); err != nil {
		return shttp.Error(err)
	}

	store := deploy.NewStore()
	depl, err := store.DeploymentByID(req.Context(), ddr.DeploymentID)

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NoContent()
	}

	if err := store.MarkDeploymentsAsDeleted(req.Context(), []types.ID{ddr.DeploymentID}); err != nil {
		return shttp.Error(err)
	}

	if err := appcache.Service().Reset(depl.EnvID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
