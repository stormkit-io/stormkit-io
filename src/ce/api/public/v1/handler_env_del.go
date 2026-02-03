package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerEnvDel(req *app.RequestContext) *shttp.Response {
	envID := utils.StringToID(utils.GetString(req.Query().Get("envId"), req.Query().Get("id")))

	if envID == 0 {
		return &shttp.Response{
			Data: map[string]string{
				"error": "Environment ID is required",
			},
		}
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err)
	}

	if env.AppID != req.Token.AppID {
		return shttp.NotAllowed()
	}

	deleted, err := buildconf.NewStore().MarkAsDeleted(req.Context(), env.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if !deleted {
		return shttp.NotFound()
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeEnv).
			WithDiff(&audit.Diff{Old: audit.DiffFields{EnvName: env.Name}}).
			WithEnvID(env.ID).
			WithAppID(req.App.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
