package publicapiv1

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// SchemaConfigureRequest is a partial update — only fields whose pointer is
// non-nil are applied. Omitting a field leaves the stored value untouched.
type SchemaConfigureRequest struct {
	InjectEnvVars     *bool   `json:"injectEnvVars,omitempty"`
	MigrationsEnabled *bool   `json:"migrationsEnabled,omitempty"`
	MigrationsFolder  *string `json:"migrationsFolder,omitempty"`
}

// handlerSchemaConfigure updates the schema configuration flags (migrations,
// env-var injection) for an existing schema. Only fields present in the
// request body are touched; omitted fields retain their current value.
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

	if data.MigrationsEnabled != nil {
		env.SchemaConf.MigrationsEnabled = *data.MigrationsEnabled
	}

	if data.InjectEnvVars != nil {
		env.SchemaConf.InjectEnvVars = *data.InjectEnvVars
	}

	if data.MigrationsFolder != nil {
		env.SchemaConf.MigrationsFolder = utils.TrimPath(*data.MigrationsFolder)
	}

	err = buildconf.NewStore().SaveSchemaConf(req.Context(), req.EnvID, env.SchemaConf)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}
