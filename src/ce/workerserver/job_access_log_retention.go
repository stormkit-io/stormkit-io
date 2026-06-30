package jobs

import (
	"context"
	"os"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	defaultAccessLogRetentionDays = 30
	// accessLogCreateAheadDays keeps a few days of partitions ready ahead of
	// time so ingestion never lands on a missing partition between job runs.
	accessLogCreateAheadDays = 3
)

// MaintainAccessLogPartitions keeps the daily-partitioned request_logs table
// bounded: it creates the partitions for the next few days and drops the ones
// older than the retention window. DROP PARTITION is an instant metadata
// operation, unlike the batched ctid-DELETE used for the analytics table.
//
// The window defaults to 30 days and can be overridden via
// STORMKIT_ACCESS_LOG_RETENTION_DAYS.
func MaintainAccessLogPartitions(ctx context.Context) error {
	days := utils.StringToInt(os.Getenv("STORMKIT_ACCESS_LOG_RETENTION_DAYS"))

	if days <= 0 {
		days = defaultAccessLogRetentionDays
	}

	store := NewStore()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for i := 0; i <= accessLogCreateAheadDays; i++ {
		if err := store.CreateAccessLogPartition(ctx, today.AddDate(0, 0, i)); err != nil {
			slog.Errorf("could not create access log partition: %s", err.Error())
			return err
		}
	}

	partitions, err := store.AccessLogPartitions(ctx)

	if err != nil {
		slog.Errorf("could not list access log partitions: %s", err.Error())
		return err
	}

	cutoff := today.AddDate(0, 0, -days)
	dropped := 0

	for _, name := range partitions {
		match := accessLogPartitionName.FindStringSubmatch(name)

		if match == nil {
			continue
		}

		day, err := time.Parse(accessLogPartitionDateLayout, match[1])

		if err != nil || !day.Before(cutoff) {
			continue
		}

		if err := store.DropAccessLogPartition(ctx, name); err != nil {
			slog.Errorf("could not drop access log partition %s: %s", name, err.Error())
			return err
		}

		dropped++
	}

	if dropped > 0 {
		slog.Infof("dropped %d access log partitions (retention=%d days)", dropped, days)
	}

	return nil
}
