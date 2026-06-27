package jobs

import (
	"context"
	"os"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	defaultAnalyticsRetentionDays = 180
	analyticsRetentionBatchSize   = 10000
	analyticsRetentionMaxBatches  = 1000
)

// RemoveOldAnalytics deletes raw analytics rows older than the retention window
// in batches to avoid long locks and table bloat. The window defaults to 180
// days and can be overridden via STORMKIT_ANALYTICS_RETENTION_DAYS. Aggregated
// analytics tables are intentionally kept and not touched here.
func RemoveOldAnalytics(ctx context.Context) error {
	days := utils.StringToInt(os.Getenv("STORMKIT_ANALYTICS_RETENTION_DAYS"))

	if days <= 0 {
		days = defaultAnalyticsRetentionDays
	}

	store := NewStore()

	var total int64

	for range analyticsRetentionMaxBatches {
		deleted, err := store.RemoveOldAnalytics(ctx, RemoveOldAnalyticsParams{
			RetentionDays: days,
			BatchSize:     analyticsRetentionBatchSize,
		})

		if err != nil {
			slog.Errorf("could not remove old analytics: %s", err.Error())
			return err
		}

		total += deleted

		if deleted < analyticsRetentionBatchSize {
			break
		}
	}

	if total > 0 {
		slog.Infof("removed %d old analytics records (retention=%d days)", total, days)
	}

	return nil
}
