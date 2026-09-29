package apphandlers

import (
	"crypto/subtle"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"gopkg.in/guregu/null.v3"
)

type hookParams struct {
	Branch  null.String `json:"branch"`
	Publish null.Bool   `json:"publish"`
}

// handlerAppHooksDeploy deploys an application.
func handlerAppHooksDeploy(req *app.RequestContext) *shttp.Response {
	var err error
	var settings *app.Settings

	vars := req.Vars()
	hash := vars["hash"]
	envVar := vars["env"]

	if settings, err = app.NewStore().Settings(req.Context(), req.App.ID); err != nil {
		return shttp.Error(err)
	}

	if hash == "" || subtle.ConstantTimeCompare([]byte(settings.DeployTrigger), []byte(hash)) != 1 {
		return shttp.NotAllowed()
	}

	params := &hookParams{}

	if req.Method == shttp.MethodPost {
		if err := req.Post(params); err != nil {
			return shttp.Error(err)
		}
	} else {
		query := req.Query()
		shouldPublish := query.Get("publish")
		branch := query.Get("branch")

		params.Branch = null.NewString(branch, branch != "")

		switch shouldPublish {
		case "true":
			params.Publish = null.NewBool(true, true)
		case "false":
			params.Publish = null.NewBool(false, true)
		}
	}

	var env *buildconf.Env

	if envID, convErr := strconv.Atoi(envVar); convErr == nil {
		env, err = buildconf.NewStore().EnvironmentByID(req.Context(), types.ID(envID))

		if err != nil {
			return shttp.Error(err)
		}
	}

	// The trigger hash only authorizes deployments of its own app. When the
	// segment is not one of the app's environment IDs, it is an environment
	// name, which may itself be numeric.
	if env == nil || env.AppID != req.App.ID {
		if env, err = buildconf.NewStore().Environment(req.Context(), req.App.ID, envVar); err != nil {
			return shttp.Error(err)
		}
	}

	if env == nil {
		return shttp.NotFound()
	}

	depl := deploy.New(req.App)
	depl.PopulateFromEnv(env)
	depl.IsAutoDeploy = true

	if params.Publish.Valid {
		depl.ShouldPublish = params.Publish.ValueOrZero()
	}

	if params.Branch.Valid {
		depl.Branch = params.Branch.ValueOrZero()
	}

	if err := deployservice.New().Deploy(req.Context(), req.App, depl); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
