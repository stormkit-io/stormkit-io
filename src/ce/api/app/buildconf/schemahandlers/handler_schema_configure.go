package schemahandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type SchemaConfigureRequest struct {
	MigrationsEnabled bool   `json:"migrationsEnabled"`
	MigrationsPath    string `json:"migrationsPath"`
}

func handlerSchemaConfigure(req *app.RequestContext) *shttp.Response {
	data := SchemaConfigureRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err)
	}

	if env.SchemaConf == nil {
		env.SchemaConf = &buildconf.SchemaConf{}
	}

	env.SchemaConf.MigrationsEnabled = data.MigrationsEnabled
	env.SchemaConf.MigrationsPath = data.MigrationsPath

	err = buildconf.NewStore().SaveSchemaConf(req.Context(), req.EnvID, env.SchemaConf)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}
