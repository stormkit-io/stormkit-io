package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentStop(req *RequestContext) *shttp.Response {
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

	if !depl.ExitCode.Valid {
		if err := store.StopDeployment(req.Context(), depl.ID); err != nil {
			return shttp.Error(err)
		}
	}

	if depl.HasStatusChecks() {
		if err := store.StopStatusChecks(req.Context(), depl.ID); err != nil {
			return shttp.Error(err)
		}
	}

	if depl.GithubRunID.ValueOrZero() != 0 {
		_ = deployservice.Github().StopDeployment(depl.GithubRunID.ValueOrZero())
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeDeployment).
			WithDiff(&audit.Diff{New: audit.DiffFields{DeploymentID: id.String(), Stopped: utils.Ptr(true)}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"ok": true},
	}
}
