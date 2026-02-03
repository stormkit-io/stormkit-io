package authwallhandlers

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerAuthDelete(req *app.RequestContext) *shttp.Response {
	ids := []types.ID{}
	pieces := strings.Split(req.Query().Get("id"), ",")

	for _, piece := range pieces {
		if id := utils.StringToID(strings.TrimSpace(piece)); id != 0 {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return shttp.BadRequest(map[string]any{
			"error": "Missing login ID(s).",
		})
	}

	err := authwall.Store().RemoveLogins(req.Context(), req.EnvID, ids)

	if err != nil {
		return shttp.Error(err)
	}

	idsStr := []string{}

	for _, id := range ids {
		idsStr = append(idsStr, id.String())
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				AuthWallDeleteLoginIDs: strings.Join(idsStr, ","),
			},
		}

		err = audit.FromRequestContext(req).
			WithAction(audit.DeleteAction, audit.TypeAuthWall).
			WithDiff(diff).
			WithEnvID(req.EnvID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}
