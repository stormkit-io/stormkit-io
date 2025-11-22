package userhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// handlerUserSession returns the current user session, if any.
func handlerUserSession(req *user.RequestContext) *shttp.Response {
	store := user.NewStore()
	accounts, err := store.Accounts(req.User.ID)

	if err != nil {
		return shttp.Error(err)
	}

	data := map[string]any{
		"user":     req.User.JSON(),
		"accounts": accounts,
	}

	if config.IsStormkitCloud() {
		metrics, err := store.UserMetrics(req.Context(), user.UserMetricsArgs{UserID: req.User.ID})

		if err != nil {
			return shttp.Error(err)
		}

		if metrics == nil {
			metrics = &user.UserMetrics{
				Metadata: req.User.Metadata,
			}
		}

		limits, ok := config.Limits[metrics.Metadata.PackageName]

		if !ok {
			limits = config.Limits[config.PackageFree]
		}

		data["metrics"] = map[string]any{
			"used": map[string]any{
				"buildMinutes":        metrics.BuildMinutes,
				"functionInvocations": metrics.FunctionInvocations,
				"storageInBytes":      metrics.StorageUsedInBytes,
				"bandwidthInBytes":    metrics.BandwidthUsedInBytes,
			},
			"max": map[string]any{
				"buildMinutes":        limits.BuildMinutes * int64(utils.GetInt(metrics.Metadata.SeatsPurchased, 1)),
				"functionInvocations": limits.FunctionInvocations * int64(utils.GetInt(metrics.Metadata.SeatsPurchased, 1)),
				"storageInBytes":      limits.StorageInBytes * int64(utils.GetInt(metrics.Metadata.SeatsPurchased, 1)),
				"bandwidthInBytes":    limits.BandwidthInBytes * int64(utils.GetInt(metrics.Metadata.SeatsPurchased, 1)),
			},
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   data,
	}
}
