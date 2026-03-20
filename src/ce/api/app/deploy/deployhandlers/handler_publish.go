package deployhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
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

	settings := []*deploy.PublishSettings{}

	for _, publishDetails := range data.Publish {
		settings = append(settings, &deploy.PublishSettings{
			EnvID:        env.ID,
			DeploymentID: publishDetails.DeploymentID,
			Percentage:   publishDetails.Percentage,
		})
	}

	if err := Publish(req.Context(), settings); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		for _, publishDetails := range data.Publish {
			err := audit.FromRequestContext(req).
				WithAction(audit.UpdateAction, audit.TypeDeployment).
				WithEnvID(env.ID).
				WithDiff(&audit.Diff{New: audit.DiffFields{DeploymentID: publishDetails.DeploymentID.String()}}).
				Insert()

			if err != nil {
				return shttp.Error(err)
			}
		}
	}

	var publishConfig []any

	if data.Publish != nil {
		publishConfig = []any{}

		for _, cnf := range data.Publish {
			publishConfig = append(publishConfig, map[string]any{
				"percentage":   cnf.Percentage,
				"deploymentId": cnf.DeploymentID.String(),
			})
		}
	}

	return &shttp.Response{
		Data: map[string]any{
			"appId":  req.App.ID.String(),
			"envId":  env.ID.String(),
			"config": publishConfig,
		},
	}
}

var Publish = deploy.Publish
