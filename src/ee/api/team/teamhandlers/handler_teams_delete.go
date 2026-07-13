package teamhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerTeamsDelete(req *user.RequestContext) *shttp.Response {
	id := utils.StringToID(req.Query().Get("teamId"))
	store := team.NewStore()
	myTeam, err := store.Team(req.Context(), id, req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if myTeam == nil {
		return shttp.NotFound()
	}

	if myTeam.IsDefault {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "You cannot delete a default team.",
			},
		}
	}

	if myTeam.CurrentUserRole != team.ROLE_ADMIN &&
		myTeam.CurrentUserRole != team.ROLE_OWNER {
		return shttp.NotAllowed()
	}

	if err := store.MarkTeamAsSofDeleted(req.Context(), myTeam.ID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
