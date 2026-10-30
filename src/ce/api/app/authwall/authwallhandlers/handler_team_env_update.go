package authwallhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type teamEnvUpdateRequest struct {
	TeamID      types.ID  `json:"teamId"`
	EnvID       types.ID  `json:"envId"`
	AuthWall    *string   `json:"authWall,omitempty"`
	PublicPaths *[]string `json:"publicPaths,omitempty"`
}

// handlerTeamEnvUpdate sets the mode and public paths of the wall of an
// environment of the team's apps. Fields left out are kept. Every member of
// the team can.
func handlerTeamEnvUpdate(req *user.RequestContext) *shttp.Response {
	data := &teamEnvUpdateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	t, res := teamAccess{req: req}.team(data.TeamID)

	if res != nil {
		return res
	}

	envs, err := authwall.Store().TeamEnvs(req.Context(), authwall.TeamEnvsParams{
		TeamID: t.ID,
		EnvID:  data.EnvID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if data.EnvID == 0 || len(envs) == 0 {
		return shttp.NotFound()
	}

	if data.AuthWall == nil && data.PublicPaths == nil {
		return shttp.BadRequest(map[string]any{
			"error": "Nothing to update. Send authWall, publicPaths or both.",
		})
	}

	update := envStatus{ctx: req.Context()}

	if req.License().IsEnterprise() {
		update.audit = audit.FromRequestContext(req).
			WithTeamID(t.ID).
			WithAppID(envs[0].AppID)
	}

	change := envChange{Status: data.AuthWall, PublicPaths: data.PublicPaths}

	if res := update.update(data.EnvID, change); res != nil {
		return res
	}

	return shttp.OK()
}
