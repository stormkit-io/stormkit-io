package buildconfhandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerEnvInsert inserts a build configuration for the given application.
func handlerEnvInsert(req *app.RequestContext) *shttp.Response {
	cnf := &buildconf.Env{
		Data: &buildconf.BuildConf{},
	}

	if err := req.Post(cnf); err != nil {
		return shttp.Error(err)
	}

	cnf.AppID = req.App.ID
	cnf.Name = cnf.Env

	if err := buildconf.NewStore().Insert(req.Context(), cnf); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return shttp.Error(buildconf.ErrDuplicateEnvName)
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeEnv).
			WithDiff(&audit.Diff{New: audit.DiffFields{EnvName: cnf.Name, EnvID: cnf.ID.String()}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]any{
			"envId": cnf.ID.String(),
		},
	}
}
