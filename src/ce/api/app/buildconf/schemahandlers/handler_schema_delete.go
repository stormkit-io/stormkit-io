package schemahandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

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

	member, err := team.NewStore().TeamMember(req.Context(), req.App.TeamID, req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if member == nil || !team.HasWriteAccess(member.Role) {
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

	if req.License().Enterprise {
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
