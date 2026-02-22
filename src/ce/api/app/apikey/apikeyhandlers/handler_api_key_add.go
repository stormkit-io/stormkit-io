package apikeyhandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type APIKeyGenerateRequest struct {
	Name   string   `json:"name"`
	Scope  string   `json:"scope"`
	EnvID  types.ID `json:"envId,string"`
	AppID  types.ID `json:"appId,string"`
	TeamID types.ID `json:"teamId,string"`
	UserID types.ID `json:"userId,string"`
}

func handlerAPIKeyAdd(req *user.RequestContext) *shttp.Response {
	data := &APIKeyGenerateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	key := &apikey.Token{
		AppID:  data.AppID,
		EnvID:  data.EnvID,
		TeamID: data.TeamID,
		UserID: data.UserID,
		Name:   strings.TrimSpace(data.Name),
		Scope:  data.Scope,
		Value:  apikey.GenerateTokenValue(),
	}

	if key.Scope == "" {
		if key.TeamID != 0 {
			key.Scope = apikey.SCOPE_TEAM
		} else if key.UserID != 0 {
			key.Scope = apikey.SCOPE_USER
		} else if key.EnvID != 0 {
			key.Scope = apikey.SCOPE_ENV
		} else if key.AppID != 0 {
			key.Scope = apikey.SCOPE_APP
		} else {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": "Scope is required when no other identifiers are provided.",
				},
			}
		}
	}

	if data.UserID != 0 {
		if data.UserID != req.User.ID {
			return shttp.Forbidden()
		}
	}

	if data.EnvID != 0 {
		envStore := buildconf.NewStore()
		env, err := envStore.EnvironmentByID(req.Context(), data.EnvID)

		if err != nil {
			return shttp.Error(err)
		}

		if env == nil {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": "Environment not found.",
				},
			}
		}

		// Make sure the key is not malformed
		key.AppID = env.AppID

		// Make sure user has access to the environment
		if !envStore.IsMember(req.Context(), env.ID, req.User.ID) {
			return shttp.Forbidden()
		}
	}

	if data.EnvID == 0 && data.AppID != 0 {
		app, err := app.NewStore().AppByID(req.Context(), data.AppID)

		if err != nil {
			return shttp.Error(err)
		}

		if app == nil {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": "Application not found.",
				},
			}
		}

		// Make sure user has access to the application
		if !team.NewStore().IsMember(req.Context(), app.ID, req.User.ID) {
			return shttp.Forbidden()
		}
	}

	if data.TeamID != 0 {
		teamStore := team.NewStore()
		team, err := teamStore.Team(req.Context(), data.TeamID, req.User.ID)

		if err != nil {
			return shttp.Error(err)
		}

		if team == nil {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": "Team not found.",
				},
			}
		}
	}

	if !apikey.IsScopeValid(key.Scope) {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": fmt.Sprintf("Invalid scope. Allowes scopes are: %s", strings.Join(apikey.AllowedScopes, ", ")),
			},
		}
	}

	if key.Name == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Key name is a required field.",
			},
		}
	}

	if err := apikey.NewStore().AddAPIKey(req.Context(), key); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data:   key,
	}
}
