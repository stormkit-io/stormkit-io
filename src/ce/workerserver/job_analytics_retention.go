package jobs

import (
	"context"
	"os"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	defaultAnalyticsRetentionDays = 180
	retentionBatchSize            = 10000
	retentionMaxBatches           = 1000
)

// RemoveOldAnalytics deletes raw analytics and custom-event rows older than the
// retention window in batches to avoid long locks and table bloat. The window
// defaults to 180 days and can be overridden via STORMKIT_ANALYTICS_RETENTION_DAYS.
// Aggregated analytics tables are intentionally kept and not touched here.
func RemoveOldAnalytics(ctx context.Context) error {
	days := utils.StringToInt(os.Getenv("STORMKIT_ANALYTICS_RETENTION_DAYS"))

	if days <= 0 {
		days = defaultAnalyticsRetentionDays
	}

	store := NewStore()

	hits, err := batchDeleteOldRows(ctx, days, store.RemoveOldAnalytics)

	if err != nil {
		slog.Errorf("could not remove old analytics: %s", err.Error())
		return err
	}

	events, err := batchDeleteOldRows(ctx, days, store.RemoveOldAnalyticsEvents)

	if err != nil {
		slog.Errorf("could not remove old analytics events: %s", err.Error())
		return err
	}

	if hits > 0 || events > 0 {
		slog.Infof("removed %d analytics and %d event records (retention=%d days)", hits, events, days)
	}

	return nil
}

// batchDeleteOldRows repeatedly invokes a batched delete until a batch comes
// back smaller than the batch size (i.e. the backlog is drained) or the per-run
// cap is reached, returning the total number of rows deleted.
func batchDeleteOldRows(ctx context.Context, days int, del func(context.Context, RemoveOldRowsParams) (int64, error)) (int64, error) {
	var total int64

	for range retentionMaxBatches {
		deleted, err := del(ctx, RemoveOldRowsParams{
			RetentionDays: days,
			BatchSize:     retentionBatchSize,
		})

		if err != nil {
			return total, err
		}

		total += deleted

		if deleted < retentionBatchSize {
			break
		}
	}

	return total, nil
}
