package authwallhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerTeamEnvs returns the team's apps with their environments and the mode
// of each environment's wall.
func handlerTeamEnvs(req *user.RequestContext) *shttp.Response {
	t, res := teamAccess{req: req}.team(utils.StringToID(req.Query().Get("teamId")))

	if res != nil {
		return res
	}

	envs, err := authwall.Store().TeamEnvs(req.Context(), authwall.TeamEnvsParams{TeamID: t.ID})

	if err != nil {
		return shttp.Error(err)
	}

	apps := []map[string]any{}

	for i, env := range envs {
		if i == 0 || envs[i-1].AppID != env.AppID {
			apps = append(apps, map[string]any{
				"id":           env.AppID.String(),
				"displayName":  env.AppName,
				"environments": []map[string]any{},
			})
		}

		last := apps[len(apps)-1]
		last["environments"] = append(last["environments"].([]map[string]any), map[string]any{
			"id":       env.EnvID.String(),
			"name":     env.EnvName,
			"authWall": env.Status,
		})
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"apps": apps,
		},
	}
}
