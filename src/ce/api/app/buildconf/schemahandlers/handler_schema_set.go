package schemahandlers

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func handlerSchemaSet(req *app.RequestContext) *shttp.Response {
	name := buildconf.SchemaName(req.App.ID, req.EnvID)

	creds, err := buildconf.SchemaStore().CreateSchema(req.Context(), name)

	if err != nil {
		if errors.Is(err, buildconf.ErrSchemaExists) {
			return &shttp.Response{
				Status: http.StatusConflict,
				Data: map[string]any{
					"error": "Schema already exists for this environment.",
				},
			}
		}

		if errors.Is(err, buildconf.ErrInvalidSchemaName) {
			return shttp.BadRequest(map[string]any{
				"error": "Invalid schema name.",
			})
		}

		return shttp.Error(err)
	}

	// Store creds in build config
	if creds != nil {
		if err := buildconf.NewStore().SaveSchemaConf(req.Context(), req.EnvID, creds); err != nil {
			if err := buildconf.SchemaStore().DropSchema(req.Context(), name); err != nil {
				slog.Errorf("failed to clean up schema after build config save failure: %v", err)
			}

			return shttp.Error(err)
		}
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			New: audit.DiffFields{
				SchemaName: name,
			},
		}

		err = audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeSchema).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"schema": name,
		},
	}
}
