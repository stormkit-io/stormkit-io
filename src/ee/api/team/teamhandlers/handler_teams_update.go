package teamhandlers

import (
	"net/http"
	"strings"

	"github.com/gosimple/slug"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type TeamUpdateRequest struct {
	ID   types.ID `json:"teamId,string"`
	Name string   `json:"name"`
}

func handlerTeamsUpdate(req *user.RequestContext) *shttp.Response {
	data := TeamUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	data.Name = strings.TrimSpace(data.Name)

	if data.Name == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Team name is a required field.",
			},
		}
	}

	store := team.NewStore()
	myTeam, err := store.Team(req.Context(), data.ID, req.User.ID)

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
				"error": "You cannot modify a default team.",
			},
		}
	}

	if myTeam.CurrentUserRole != team.ROLE_ADMIN &&
		myTeam.CurrentUserRole != team.ROLE_OWNER {
		return shttp.NotAllowed()
	}

	myTeam.Name = data.Name
	myTeam.Slug = slug.Make(myTeam.Name)

	if err := store.UpdateTeam(req.Context(), myTeam); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]string{
			"id":   myTeam.ID.String(),
			"name": myTeam.Name,
			"slug": myTeam.Slug,
		},
	}
}
