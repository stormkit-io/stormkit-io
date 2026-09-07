package deployhandlers

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// publishSettings specifies the percentage which
// a deployment is published.
type publishSettings struct {
	Percentage   float64  `json:"percentage"`
	DeploymentID types.ID `json:"deploymentId,string"`
}

type publishRequest struct {
	model.Model

	// Env is the alias name that will point to the specific version.
	// If nothing is provided, then it defaults to "production".
	EnvID types.ID `json:"envId,string"`

	// Publish holds a map of ids with their respective percentage to deploy.
	Publish []publishSettings `json:"publish"`
}

// Validate impleents model.Validate interface.
func (pr *publishRequest) Validate() *shttperr.ValidationError {
	err := &shttperr.ValidationError{}

	if pr.EnvID == 0 {
		err.SetError("envId", deploy.ErrMissingEnvID.Error())
	}

	total := float64(0)

	for _, publishDetails := range pr.Publish {
		if publishDetails.DeploymentID == 0 {
			err.SetError("deploymentId", deploy.ErrMissingDeploymentID.Error())
		}

		if publishDetails.Percentage < 0 || publishDetails.Percentage > 100 {
			err.SetError("percentage", buildconf.ErrInvalidPercentage.Error())
		}

		total = total + publishDetails.Percentage
	}

	if total != 100 {
		err.SetError("percentage", buildconf.ErrInvalidPercentage.Error())
	}

	return err.ToError()
}

// handlerPublish publishes a deployment for the given environment and app.
// If the environment is not found, it returns 404.
func handlerPublish(req *app.RequestContext) *shttp.Response {
	data := &publishRequest{
		Publish: []publishSettings{},
	}

	if err := req.Post(data); err != nil {
		return shttp.ValidationError(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), data.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if env == nil {
		return shttp.NotFound()
	}

	// Extra security check to validate env belongs to app.
	if env.AppID != req.App.ID {
		return shttp.NotFound()
	}

	// An environment serves exactly one deployment. Percentage-based releases
	// are retired, so a request naming several of them cannot be honoured and
	// is rejected rather than silently applying one.
	if len(data.Publish) != 1 {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"Exactly one deployment can be published at a time"},
		})
	}

	deploymentID := data.Publish[0].DeploymentID

	// Both of these are read while the request is still alive. The licence
	// lookup queries the database on Stormkit Cloud, and the audit builder
	// captures the request's context — neither survives the response, and the
	// callback below runs long after it.
	isEnterprise := req.License().IsEnterprise()
	auditEntry := audit.FromRequestContext(req).
		WithAction(audit.UpdateAction, audit.TypeDeployment).
		WithEnvID(env.ID).
		WithDiff(&audit.Diff{New: audit.DiffFields{DeploymentID: deploymentID.String()}})

	bg := context.WithoutCancel(req.Context())

	err = PublishWithWarmup(req.Context(), deploy.PublishWithWarmupParams{
		EnvID:        env.ID,
		DeploymentID: deploymentID,

		// The audit entry records the publish that happened, so it is written
		// once the environment has actually moved.
		OnPublished: func() {
			if !isEnterprise {
				return
			}

			if err := auditEntry.WithContext(bg).Insert(); err != nil {
				slog.Errorf("cannot audit the publish of deployment %s: %s", deploymentID.String(), err.Error())
			}
		},
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"appId":  req.App.ID.String(),
			"envId":  env.ID.String(),
			"status": deploy.PublishStatusPublishing,
			"config": []any{
				map[string]any{"deploymentId": deploymentID.String(), "percentage": float64(100)},
			},
		},
	}
}

var PublishWithWarmup = deploy.PublishWithWarmup
