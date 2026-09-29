package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// handlerSchemaDelete drops the environment's schema and clears its
// configuration. Requires write access on the team.
func handlerSchemaDelete(req *app.RequestContext) *shttp.Response {
	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if env.SchemaConf == nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Schema is not configured for this environment.",
			},
		}
	}

	// Dropping a schema needs a person with write access on the team: the
	// logged-in user, or the owner of a user-level key. Keys that carry no
	// user have no team role to check, so they are refused.
	var userID types.ID

	if req.User != nil {
		userID = req.User.ID
	} else if req.Token != nil {
		userID = req.Token.UserID
	}

	if userID == 0 {
		return shttp.Forbidden()
	}

	myTeam, err := team.NewStore().Team(req.Context(), req.App.TeamID, userID)

	if err != nil {
		return shttp.Error(err)
	}

	if myTeam == nil || !team.HasWriteAccess(myTeam.CurrentUserRole) {
		return shttp.Forbidden()
	}

	schemaName := env.SchemaConf.SchemaName

	err = buildconf.SchemaStore().DropSchema(req.Context(), schemaName)

	if err != nil {
		return shttp.Error(err)
	}

	if err := buildconf.NewStore().SaveSchemaConf(req.Context(), req.EnvID, nil); err != nil {
		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				SchemaName: schemaName,
			},
		}

		err = audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeSchema).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}
