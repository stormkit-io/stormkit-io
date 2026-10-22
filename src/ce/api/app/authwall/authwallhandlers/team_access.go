package authwallhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// teamAccess checks the caller's access to a team's Auth Wall. Every member can
// see the walls of the team's environments and switch them; only owners and
// admins manage the logins.
type teamAccess struct {
	req *user.RequestContext

	// manage requires the caller to be an owner or admin of the team.
	manage bool
}

// team returns the team, or the response to send when the caller has no
// access to it.
func (a teamAccess) team(teamID types.ID) (*team.Team, *shttp.Response) {
	if teamID == 0 {
		return nil, shttp.BadRequest(map[string]any{
			"error": "Team ID is required.",
		})
	}

	t, err := team.NewStore().Team(a.req.Context(), teamID, a.req.User.ID)

	if err != nil {
		return nil, shttp.Error(err)
	}

	if t == nil || (a.manage && !team.HasWriteAccess(t.CurrentUserRole)) {
		return nil, shttp.Forbidden()
	}

	return t, nil
}
