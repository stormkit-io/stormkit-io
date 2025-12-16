package schemahandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerSchemaGet(req *app.RequestContext) *shttp.Response {
	name := buildconf.SchemaName(req.App.ID, req.EnvID)

	schema, err := buildconf.SchemaStore().GetSchema(req.Context(), name)

	if err != nil {
		return shttp.Error(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	data := map[string]any{
		"schema": nil,
	}

	if schema != nil {
		mappedData := schema.Map()

		if env.SchemaConf != nil {
			mappedData["migrationsEnabled"] = env.SchemaConf.MigrationsEnabled
			mappedData["migrationsFolder"] = env.SchemaConf.MigrationsFolder
		}

		data["schema"] = mappedData
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   data,
	}
}
