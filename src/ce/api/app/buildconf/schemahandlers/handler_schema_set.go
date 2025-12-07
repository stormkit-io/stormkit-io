package schemahandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerSchemaSet(req *app.RequestContext) *shttp.Response {
	name := buildconf.SchemaName(req.App.ID, req.EnvID)

	if err := buildconf.SchemaStore().CreateSchema(req.Context(), name); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"schema": name,
		},
	}
}
