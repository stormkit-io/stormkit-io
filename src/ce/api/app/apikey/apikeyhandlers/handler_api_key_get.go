package apikeyhandlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerAPIKeyGet(req *user.RequestContext) *shttp.Response {
	scope := apikey.SCOPE_ENV
	envID := utils.StringToID(req.Query().Get("envId"))

	var id types.ID

	if envID != 0 {
		id = envID

		if !buildconf.NewStore().IsMember(req.Context(), envID, req.User.ID) {
			return shttp.Forbidden()
		}
	}

	if teamID := req.Query().Get("teamId"); teamID != "" {
		parsed, _ := strconv.ParseInt(teamID, 10, 64)

		if parsed != 0 {
			scope = apikey.SCOPE_TEAM
			id = types.ID(parsed)
		}

		// Double check access
		if !team.NewStore().IsMember(req.Context(), req.User.ID, id) {
			return shttp.Forbidden()
		}
	}

	if userId := req.Query().Get("userId"); userId != "" {
		parsed, _ := strconv.ParseInt(userId, 10, 64)

		if parsed != 0 {
			scope = apikey.SCOPE_USER
			id = types.ID(parsed)
		}

		if id != req.User.ID {
			return shttp.Forbidden()
		}
	}

	keys, err := apikey.NewStore().APIKeys(req.Context(), id, scope)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("Failed to retrieve API keys for %s with ID %d, err: %s", scope, id, err.Error()))
	}

	keysJson := []map[string]any{}

	for _, key := range keys {
		teamID := ""

		if key.TeamID != 0 {
			teamID = key.TeamID.String()
		}

		keysJson = append(keysJson, map[string]any{
			"id":     key.ID,
			"appId":  key.AppID,
			"envId":  key.EnvID,
			"teamId": teamID,
			"name":   key.Name,
			"scope":  key.Scope,
		})
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"keys": keysJson,
		},
	}
}
