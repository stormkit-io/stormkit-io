package teamhandlers

import (
	"net/http"
	"net/mail"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type TeamsInviteRequest struct {
	TeamID types.ID `json:"teamId,string"`
	Email  string   `json:"email"`
	Role   string   `json:"role"`
}

// handlerTeamsInvite handles the invitation of a new member to the team.
func handlerTeamsInvite(req *user.RequestContext) *shttp.Response {
	data := &TeamsInviteRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	if data.Email == req.User.PrimaryEmail() {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "You cannot invite yourself :)",
			},
		}
	}

	if _, err := mail.ParseAddress(data.Email); err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Email is invalid.",
			},
		}
	}

	store := team.NewStore()
	myTeam, err := store.Team(req.Context(), data.TeamID, req.User.ID)

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
				"error": "You cannot add a member to your default team. Please create a new team.",
			},
		}
	}

	if !team.IsValidRole(data.Role) {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid role given.",
			},
		}
	}

	if !team.HasWriteAccess(myTeam.CurrentUserRole) {
		return shttp.NotAllowed()
	}

	token, err := req.JWT(jwt.MapClaims{
		"inviterId": req.User.ID.String(),
		"teamId":    data.TeamID.String(),
		"email":     data.Email,
		"role":      data.Role,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]string{
			"token": token,
		},
	}
}
