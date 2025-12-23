package teamhandlers

import (
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerTeamMembers(req *user.RequestContext) *shttp.Response {
	teamID := utils.StringToID(req.Query().Get("teamId"))
	store := team.NewStore()

	if !store.IsMember(req.Context(), req.User.ID, teamID) {
		return shttp.NotAllowed()
	}

	teamMembers, err := store.TeamMembers(req.Context(), team.TeamMemberFilters{
		TeamID: teamID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	members := []map[string]any{}

	for _, m := range teamMembers {
		members = append(members, map[string]any{
			"id":          m.ID.String(),
			"teamId":      teamID.String(),
			"userId":      m.UserID.String(),
			"firstName":   m.FirstName.ValueOrZero(),
			"lastName":    m.LastName.ValueOrZero(),
			"avatar":      m.Avatar.ValueOrZero(),
			"displayName": m.DisplayName,
			"fullName":    strings.Join([]string{m.FirstName.ValueOrZero(), m.LastName.ValueOrZero()}, " "),
			"email":       m.Email,
			"role":        m.Role,
			"status":      m.Status,
		})
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   members,
	}
}
