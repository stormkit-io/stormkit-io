package skauthhandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerAuthUserUpdate updates the email and name of a registered auth user.
// The user is addressed by its external UUID.
// PUT /skauth/users/{id}
func handlerAuthUserUpdate(req *app.RequestContext) *shttp.Response {
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

	data := struct {
		Email     string `json:"email"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}{}

	if err := req.Post(&data); err != nil {
		return shttp.BadRequest(map[string]any{"errors": []string{"invalid request body"}})
	}

	data.Email = strings.TrimSpace(data.Email)

	if !utils.IsValidEmail(data.Email) {
		return shttp.BadRequest(map[string]any{"errors": []string{"email is invalid"}})
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to get schema store: %s", err.Error()))
	}

	authUser, err := store.UpdateAuthUser(req.Context(), buildconf.UpdateAuthUserParams{
		UUID:      uuid,
		Email:     data.Email,
		FirstName: strings.TrimSpace(data.FirstName),
		LastName:  strings.TrimSpace(data.LastName),
	})

	if err != nil {
		if database.IsDuplicate(err) {
			return shttp.BadRequest(map[string]any{"errors": []string{"an account with this email already exists"}})
		}

		return shttp.Error(err, fmt.Sprintf("failed to update auth user: %s", err.Error()))
	}

	if authUser == nil {
		return shttp.NotFound()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   authUser.JSON(),
	}
}
