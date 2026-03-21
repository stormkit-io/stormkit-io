package buildconfhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type EnvInsertRequest struct {
	Name   string `json:"name"`
	Env    string `json:"env"` // DEPRECATED: use name instead
	Branch string `json:"branch"`
}

// handlerEnvInsert inserts a build configuration for the given application.
func handlerEnvInsert(req *app.RequestContext) *shttp.Response {
	formData := &EnvInsertRequest{}

	cnf := &buildconf.Env{
		Data: &buildconf.BuildConf{},
	}

	if err := req.Post(formData); err != nil {
		return shttp.Error(err)
	}

	cnf.AppID = req.App.ID
	cnf.Name = utils.GetString(formData.Name, formData.Env) // Env is deprecated, but we still want to support it for a while
	cnf.Branch = formData.Branch

	if errs := buildconf.Validate(cnf); len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
	}

	if err := buildconf.NewStore().Insert(req.Context(), cnf); err != nil {
		if database.IsDuplicate(err) {
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
