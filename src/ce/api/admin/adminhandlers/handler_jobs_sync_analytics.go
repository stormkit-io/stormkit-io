package adminhandlers

import (
	"context"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerJobsSyncAnalytics(req *user.RequestContext) *shttp.Response {
	ts := req.Query().Get("ts")

	if ts == "24h" {
		if err := jobs.SyncAnalyticsVisitorsHourly(req.Context()); err != nil {
			return shttp.Error(err)
		}

		return shttp.OK()
	}

	var ctx context.Context

	if ts == "7d" {
		ctx = context.WithValue(req.Context(), jobs.KeyContextNumberOfDays{}, 7)
	} else if ts == "30d" {
		ctx = context.WithValue(req.Context(), jobs.KeyContextNumberOfDays{}, 30)
	}

	if ctx != nil {
		if err := jobs.SyncAnalyticsVisitorsDaily(ctx); err != nil {
			return shttp.Error(err)
		}

		return shttp.OK()
	}

	return &shttp.Response{
		Status: http.StatusBadRequest,
		Data: map[string]any{
			"error": "invalid ts: expected 7d, 30d or 24h",
		},
	}
}
