package teamhandlers

import (
	"encoding/json"
	"net/http"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type TeamsInvitationAcceptRequest struct {
	Token string `json:"token"`
}

// handlerTeamsInvitationAccept enrolls a user to the given team.
func handlerTeamsInvitationAccept(req *user.RequestContext) *shttp.Response {
	data := &TeamsInvitationAcceptRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: data.Token, MaxMins: 24 * 60 * 7})

	if claims == nil {
		return &shttp.Response{
			Status: http.StatusUnauthorized,
			Data: map[string]string{
				"error": "Invitation token is either invalid or expired.",
			},
		}
	}

	claimsStruct := struct {
		InviterID string `json:"inviterId"`
		TeamID    string `json:"teamId"`
		Email     string `json:"email"`
		Role      string `json:"role"`
	}{}

	b, err := json.Marshal(claims)

	if err != nil {
		return shttp.Error(err)
	}

	if err := json.Unmarshal(b, &claimsStruct); err != nil {
		return shttp.Error(err)
	}

	if !req.User.HasEmail(claimsStruct.Email) {
		return shttp.NotAllowed()
	}

	if !team.IsValidRole(claimsStruct.Role) {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "Invalid role given: " + claimsStruct.Role,
			},
		}
	}

	err = team.NewStore().AddMemberToTeam(req.Context(), &team.Member{
		InvitedBy: utils.StringToID(claimsStruct.InviterID),
		TeamID:    utils.StringToID(claimsStruct.TeamID),
		UserID:    req.User.ID,
		Role:      claimsStruct.Role,
		Status:    true,
	})

	if err != nil {
		duplicateErrCode := "23505"

		if err, ok := err.(*pq.Error); ok && err.Code == pq.ErrorCode(duplicateErrCode) {
			return &shttp.Response{
				Status: http.StatusConflict,
				Data: map[string]string{
					"error": "Looks like you already accepted the invitation.",
				},
			}
		}

		return shttp.Error(err)
	}

	myTeam, err := team.NewStore().Team(req.Context(), utils.StringToID(claimsStruct.TeamID), req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   myTeam.ToMap(),
	}
}
