package adminhandlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/sysstats"
)

func handlerMetricsHistory(req *user.RequestContext) *shttp.Response {
	query := req.Query()
	target := query.Get("target")

	if target == "" {
		return shttp.BadRequest(map[string]any{
			"error": "target is required",
		})
	}

	since := time.Now().Add(-sysstats.Retention)

	if raw := query.Get("since"); raw != "" {
		unix, err := strconv.ParseInt(raw, 10, 64)

		if err != nil {
			return shttp.BadRequest(map[string]any{
				"error": "since must be a unix timestamp",
			})
		}

		// Nothing older than the retention window exists, so a smaller value is
		// clamped rather than rejected.
		if requested := time.Unix(unix, 0); requested.After(since) {
			since = requested
		}
	}

	samples, err := sysstats.NewStore(sysstats.NewStoreParams{}).Read(req.Context(), target, since)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"target":  target,
			"since":   since.Unix(),
			"samples": samples,
		},
	}
}
