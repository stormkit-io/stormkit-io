package publicapiv1

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const authUsersListLimit = 25

// HandlerAuthUsersList returns a paginated list of registered auth users for the environment.
// The environment is resolved from the API key (SCOPE_ENV).
// GET /v1/auth/users
func HandlerAuthUsersList(req *RequestContext) *shttp.Response {
	env := req.Env

	if env == nil || env.AuthConf == nil || !env.AuthConf.Status || env.SchemaConf == nil {
		return shttp.NotFound()
	}

	v := &Validators{}
	errs := []string{}
	from := 0

	if fromStr := req.Query().Get("from"); fromStr != "" {
		if f, err := v.ToInt(fromStr, "from"); err != nil {
			errs = append(errs, err.Error())
		} else {
			from = f
		}
	}

	if len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
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
