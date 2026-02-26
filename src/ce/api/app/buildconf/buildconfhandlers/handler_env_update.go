package buildconfhandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerEnvUpdate updates a build configuration for the given application.
func handlerEnvUpdate(req *app.RequestContext) *shttp.Response {
	cnf := &buildconf.Env{
		Data: &buildconf.BuildConf{},
	}

	if err := req.Post(cnf); err != nil {
		return shttp.Error(err)
	}

	if cnf.ID == 0 {
		return shttp.NotFound()
	}

	store := buildconf.NewStore()
	env, err := store.EnvironmentByID(req.Context(), cnf.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if env == nil {
		return shttp.NotFound()
	}

	if cnf.Data.Headers != "" {
		_, err := deploy.ParseHeaders(cnf.Data.Headers)

		if err != nil {
			return shttp.BadRequest(map[string]any{
				"error": err.Error(),
			})
		}
	}

	cnf.Data.APIFolder = utils.TrimPath(cnf.Data.APIFolder)
	cnf.Data.APIPathPrefix = utils.TrimPath(cnf.Data.APIPathPrefix)

	if err := store.Update(req.Context(), cnf); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return shttp.Error(buildconf.ErrDuplicateEnvName)
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				EnvName:               env.Name,
				EnvBranch:             env.Branch,
				EnvAutoPublish:        audit.Bool(env.AutoPublish),
				EnvAutoDeploy:         audit.Bool(env.AutoDeploy),
				EnvAutoDeployBranches: env.AutoDeployBranches.ValueOrZero(),
				EnvAutoDeployCommits:  env.AutoDeployCommits.ValueOrZero(),
				EnvBuildConfig:        env.Data,
			},
			New: audit.DiffFields{
				EnvName:               cnf.Env,
				EnvBranch:             cnf.Branch,
				EnvAutoPublish:        audit.Bool(cnf.AutoPublish),
				EnvAutoDeploy:         audit.Bool(cnf.AutoDeploy),
				EnvAutoDeployBranches: cnf.AutoDeployBranches.ValueOrZero(),
				EnvBuildConfig:        cnf.Data,
			},
		}

		err = audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeEnv).
			WithDiff(diff).
			WithEnvID(env.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	// We need to reset cache because some configuration, such as
	// redirects, will update the appconf.
	if err := appcache.Service().Reset(env.ID); err != nil {
		return shttp.Error(err)
	}

	err = app.UpdateFunctionConfiguration(req.Context(), app.FunctionConfiguration{
		AppID: req.App.ID,
		EnvID: env.ID,
		Vars:  env.Data.Vars,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
