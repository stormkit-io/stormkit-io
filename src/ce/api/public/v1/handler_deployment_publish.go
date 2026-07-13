package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentPublish(req *RequestContext) *shttp.Response {
	id := utils.StringToID(req.Vars()["id"])

	if id == 0 {
		return shttp.NotFound()
	}

	// Verify the deployment exists and belongs to the env of the API key.
	depl, err := deploy.NewStore().MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: id,
		EnvID:        req.Env.ID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	if depl.ExitCode.ValueOrZero() != deploy.ExitCodeSuccess {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Deployment must have a successful build before it can be published"},
		})
	}

	err = deploy.Publish(req.Context(), []*deploy.PublishSettings{
		{
			EnvID:        req.Env.ID,
			DeploymentID: id,
			Percentage:   100,
		},
	})

	if err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeDeployment).
			WithDiff(&audit.Diff{New: audit.DiffFields{DeploymentID: id.String()}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"ok": true,
		},
	}
}
