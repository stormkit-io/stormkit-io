package authwallhandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerTeamLoginsDelete deletes Auth Wall logins of the team. Visitors
// already logged in keep their session until it expires.
func handlerTeamLoginsDelete(req *user.RequestContext) *shttp.Response {
	t, res := teamAccess{req: req, manage: true}.team(utils.StringToID(req.Query().Get("teamId")))

	if res != nil {
		return res
	}

	ids := []types.ID{}

	for _, piece := range strings.Split(req.Query().Get("id"), ",") {
		if id := utils.StringToID(strings.TrimSpace(piece)); id != 0 {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "Missing login ID(s).",
		})
	}

	deleted, err := authwall.Store().DeleteLogins(req.Context(), authwall.DeleteLoginsParams{
		TeamID:   t.ID,
		LoginIDs: ids,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if len(deleted) > 0 && req.License().IsEnterprise() {
		idsStr := []string{}

		for _, id := range deleted {
			idsStr = append(idsStr, id.String())
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeAuthWall).
			WithDiff(&audit.Diff{
				Old: audit.DiffFields{
					AuthWallDeleteLoginIDs: strings.Join(idsStr, ","),
				},
			}).
			WithTeamID(t.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
