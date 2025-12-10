package schemahandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerSchemaSet(req *app.RequestContext) *shttp.Response {
	name := buildconf.SchemaName(req.App.ID, req.EnvID)

	creds, err := buildconf.SchemaStore().CreateSchema(req.Context(), name)

	if err != nil {
		if err.Error() == "schema already exists" {
			return &shttp.Response{
				Status: http.StatusConflict,
				Data: map[string]any{
					"error": "Schema already exists for this environment.",
				},
			}
		}

		return shttp.Error(err)
	}

	// Store creds in build config
	if creds != nil {
		if err := buildconf.NewStore().SaveSchemaConf(req.Context(), req.EnvID, creds); err != nil {
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
