package buildconfhandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type deleteEnvRequest struct {
	Env string `json:"env"`
}

// handlerEnvDelete removes a build configuration from the given application.
func handlerEnvDelete(req *app.RequestContext) *shttp.Response {
	deleteReq := deleteEnvRequest{}

	if err := req.Post(&deleteReq); err != nil {
		return shttp.Error(err)
	}

	store := buildconf.NewStore()

	var env *buildconf.Env
	var err error

	if deleteReq.Env != "" {
		env, err = store.Environment(req.Context(), req.App.ID, deleteReq.Env)
	} else {
		env, err = store.EnvironmentByID(req.Context(), req.EnvID)
	}

	if err != nil {
		return shttp.Error(err)
	}

	if env == nil {
		return shttp.NotFound()
	}

	if strings.ToLower(env.Name) == config.AppDefaultEnvironmentName {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]any{
				"errors": map[string]string{
					"env": buildconf.ErrCantRemoveProd.Error(),
				},
			},
		}
	}

	deleted, err := buildconf.NewStore().MarkAsDeleted(req.Context(), env.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if !deleted {
		return shttp.NotFound()
	}

	if req.License().IsEnterprise() {
		err = audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeEnv).
			WithDiff(&audit.Diff{Old: audit.DiffFields{EnvName: env.Name}}).
			WithAppID(req.App.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
