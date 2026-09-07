package publicapiv1

import (
	"context"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
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

	// A running deployment has no exit code yet.
	if !depl.ExitCode.Valid {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Deployment is still running and cannot be published"},
		})
	}

	if depl.ExitCode.ValueOrZero() != deploy.ExitCodeSuccess {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Deployment must have a successful build before it can be published"},
		})
	}

	// Both of these are read while the request is still alive. The licence
	// lookup queries the database on Stormkit Cloud, and the audit builder
	// captures the request's context — neither survives the response, and the
	// callback below runs long after it.
	isEnterprise := req.License().IsEnterprise()
	auditEntry := audit.FromRequestContext(req).
		WithAction(audit.UpdateAction, audit.TypeDeployment).
		WithDiff(&audit.Diff{New: audit.DiffFields{DeploymentID: id.String()}})

	bg := context.WithoutCancel(req.Context())

	err = deploy.PublishWithWarmup(req.Context(), deploy.PublishWithWarmupParams{
		EnvID:        req.Env.ID,
		DeploymentID: id,

		// The audit entry records the publish that happened, so it is written
		// once the environment has actually moved.
		OnPublished: func() {
			if !isEnterprise {
				return
			}

			if err := auditEntry.WithContext(bg).Insert(); err != nil {
				slog.Errorf("cannot audit the publish of deployment %s: %s", id.String(), err.Error())
			}
		},
	})

	if err != nil {
		return shttp.Error(err)
	}

	// The deployment is warmed up before traffic moves to it, so the publish is
	// under way rather than done. Poll the deployment's publish status to find
	// out whether it completed.
	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"ok":     true,
			"status": deploy.PublishStatusPublishing,
		},
	}
}
