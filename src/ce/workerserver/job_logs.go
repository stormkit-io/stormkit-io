package jobs

import (
	"context"
	"os"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const defaultTriggerLogsRetentionDays = 30

// RemoveOldLogs removes application logs and function trigger logs older than
// 30 days. The trigger log window can be overridden with
// STORMKIT_TRIGGER_LOGS_RETENTION_DAYS.
func RemoveOldLogs(ctx context.Context) error {
	store := NewStore()

	if err := store.RemoveOldLogs(ctx); err != nil {
		slog.Errorf("could not remove old app logs: %s", err.Error())
		return err
	}

	days := utils.StringToInt(os.Getenv("STORMKIT_TRIGGER_LOGS_RETENTION_DAYS"))

	if days <= 0 {
		days = defaultTriggerLogsRetentionDays
	}

	removed, err := batchDeleteOldRows(ctx, days, store.RemoveOldTriggerLogs)

	if err != nil {
		slog.Errorf("could not remove old trigger logs: %s", err.Error())
		return err
	}

	if removed > 0 {
		slog.Infof("removed %d trigger log records (retention=%d days)", removed, days)
	}

	return nil
}
