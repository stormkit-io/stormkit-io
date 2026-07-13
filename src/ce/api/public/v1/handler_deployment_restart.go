package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerDeploymentRestart(req *RequestContext) *shttp.Response {
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

	if depl.Status() != "failed" {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Only failed deployments can be restarted"},
		})
	}

	if err := store.Restart(req.Context(), depl); err != nil {
		return shttp.Error(err)
	}

	if depl.CheckoutRepo == "" {
		depl.CheckoutRepo = req.App.Repo
	}

	if err := deployservice.New().Deploy(req.Context(), req.App, depl); err != nil {
		if err == oauth.ErrRepoNotFound || err == oauth.ErrCredsInvalidPermissions {
			return shttp.NotFound()
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeDeployment).
			WithDiff(&audit.Diff{New: audit.DiffFields{DeploymentID: id.String(), Restarted: utils.Ptr(true)}}).
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
