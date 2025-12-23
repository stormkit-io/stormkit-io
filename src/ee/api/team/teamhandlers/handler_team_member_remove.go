package teamhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerTeamMemberRemove(req *user.RequestContext) *shttp.Response {
	teamID := utils.StringToID(req.Query().Get("teamId"))
	memberID := utils.StringToID(req.Query().Get("memberId"))
	store := team.NewStore()

	myTeam, err := store.Team(req.Context(), teamID, req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	if !team.HasWriteAccess(myTeam.CurrentUserRole) {
		return shttp.NotAllowed()
	}

	member, err := store.TeamMembers(req.Context(), team.TeamMemberFilters{
		TeamID:   teamID,
		MemberID: memberID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if len(member) == 0 {
		return shttp.NotAllowed()
	}

	// Check user is not the last owner
	if member[0].Role == team.ROLE_OWNER {
		owners, err := store.TeamMembers(req.Context(), team.TeamMemberFilters{
			TeamID: teamID,
			Role:   team.ROLE_OWNER,
		})

		if err != nil {
			return shttp.Error(err)
		}

		if len(owners) == 1 {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": "Cannot remove the only owner of this team. Delete the team instead.",
				},
			}
		}

	}

	if err := store.RemoveTeamMember(req.Context(), teamID, memberID); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
