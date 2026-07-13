package teamhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerTeamStats(req *user.RequestContext) *shttp.Response {
	teamID := utils.StringToID(req.Query().Get("teamId"))
	store := team.NewStore()
	ctx := req.Context()

	if !store.IsMember(ctx, req.User.ID, teamID) {
		return shttp.NotAllowed()
	}

	astore := analytics.NewStore()
	period, err := astore.TotalRequestsByTeam(ctx, teamID)

	if err != nil {
		return shttp.Error(err)
	}

	if period == nil {
		period = &analytics.TotalRequestsByTeam{}
	}

	totalApps, err := astore.TotalAppsByTeam(ctx, teamID)

	if err != nil {
		return shttp.Error(err)
	}

	if totalApps == nil {
		totalApps = &analytics.TotalAppsByTeam{}
	}

	totalDeployments, err := astore.TotalDeploymentsByTeam(ctx, teamID)

	if err != nil {
		return shttp.Error(err)
	}

	avgDeployments, err := astore.AvgDeploymentDurationByTeam(ctx, teamID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"totalRequests": map[string]any{
				"current":  period.CurrentPeriod,
				"previous": period.PreviousPeriod,
			},
			"totalApps": map[string]any{
				"total":   totalApps.Total,
				"new":     totalApps.New,
				"deleted": totalApps.Deleted,
			},
			"totalDeployments": map[string]any{
				"total":    totalDeployments.Total,
				"current":  totalDeployments.Current,
				"previous": totalDeployments.Previous,
			},
			"avgDeploymentDuration": map[string]any{
				"current":  avgDeployments.Current,
				"previous": avgDeployments.Previous,
			},
		},
	}
}
