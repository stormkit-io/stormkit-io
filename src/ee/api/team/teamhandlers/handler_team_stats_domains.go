package teamhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handleTeamStatsDomains(req *user.RequestContext) *shttp.Response {
	teamID := utils.StringToID(req.Query().Get("teamId"))
	store := team.NewStore()
	ctx := req.Context()

	if !store.IsMember(ctx, req.User.ID, teamID) {
		return shttp.NotAllowed()
	}

	domains, err := analytics.NewStore().TopPerformingDomains(ctx, teamID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"domains": domains,
		},
	}
}
