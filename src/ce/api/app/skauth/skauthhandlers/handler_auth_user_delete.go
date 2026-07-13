package skauthhandlers

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAuthUserDelete removes a registered auth user (and its attached
// provider records, via ON DELETE CASCADE). The user is addressed by its
// external UUID.
// DELETE /skauth/users/{id}
func handlerAuthUserDelete(req *app.RequestContext) *shttp.Response {
	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch environment: %s", err.Error()))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	uuid := req.Vars()["id"]

	if uuid == "" {
		return shttp.NotFound()
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	deleted, err := store.DeleteAuthUser(req.Context(), uuid)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to delete auth user: %s", err.Error()))
	}

	if !deleted {
		return shttp.NotFound()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"ok": true},
	}
}
