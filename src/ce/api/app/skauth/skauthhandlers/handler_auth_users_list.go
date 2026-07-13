package skauthhandlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const authUsersListLimit = 25

// handlerAuthUsersList returns a paginated list of registered auth users for the
// environment resolved from the request.
// GET /skauth/users
func handlerAuthUsersList(req *app.RequestContext) *shttp.Response {
	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), req.EnvID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch environment: %s", err.Error()))
	}

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	from := 0

	if fromStr := req.Query().Get("from"); fromStr != "" {
		f, err := strconv.Atoi(fromStr)

		if err != nil {
			return shttp.BadRequest(map[string]any{"errors": []string{"from must be a number"}})
		}

		from = f
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	users, err := store.ListAuthUsers(req.Context(), from, authUsersListLimit+1)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to list auth users: %s", err.Error()))
	}

	hasNextPage := len(users) > authUsersListLimit

	if hasNextPage {
		users = users[:authUsersListLimit]
	}

	results := make([]map[string]any, 0, len(users))

	for _, u := range users {
		results = append(results, u.JSON())
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"results":     results,
			"hasNextPage": hasNextPage,
		},
	}
}
