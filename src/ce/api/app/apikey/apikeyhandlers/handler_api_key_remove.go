package apikeyhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerAPIKeyRemove(req *user.RequestContext) *shttp.Response {
	keyID := utils.StringToID(req.Query().Get("keyId"))

	if keyID == 0 {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid keyId query parameter.",
			},
		}
	}

	store := apikey.NewStore()
	key, err := store.APIKeyByID(req.Context(), keyID)

	if err != nil {
		return shttp.Error(err)
	}

	if key == nil {
		return shttp.NotFound()
	}

	if key.EnvID != 0 {
		if !buildconf.NewStore().IsMember(req.Context(), key.EnvID, req.User.ID) {
			return shttp.Forbidden()
		}
	} else if key.TeamID != 0 {
		if !team.NewStore().IsMember(req.Context(), req.User.ID, key.TeamID) {
			return shttp.Forbidden()
		}
	} else if key.UserID != 0 {
		if key.UserID != req.User.ID {
			return shttp.Forbidden()
		}
	}

	if err := apikey.NewStore().RemoveAPIKey(req.Context(), keyID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
