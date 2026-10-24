package jobs

import (
	"context"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var (
	analyticsContext = context.Background()
)

type KeyContextNumberOfDays struct{}

// KeyContextNow overrides the current time of the rollups, for tests.
type KeyContextNow struct{}

// SyncAnalyticsVisitorsDaily recomputes the daily visitor aggregates for the
// last KeyContextNumberOfDays days (1 to 30, default 1). The scheduled rollups
// run through SyncAnalyticsRollups; this is the admin's manual resync.
func SyncAnalyticsVisitorsDaily(ctx context.Context) error {
	days, _ := ctx.Value(KeyContextNumberOfDays{}).(int)
	days = min(max(days, 1), 30)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	err := NewStore().SyncAnalyticsVisitors(ctx, SyncAnalyticsVisitorsParams{
		From: today.AddDate(0, 0, -days),
		To:   today.AddDate(0, 0, 1),
	})

	if err != nil {
		slog.Errorf("could not sync visitors data: %s", err.Error())
	}

	return err
}

// analyticsHourlyBuckets is how many hour buckets the hourly visitors rollup
// recomputes: the current one, and the previous one for page views ingested
// after it ended. Older buckets are final.
const analyticsHourlyBuckets = 2

// SyncAnalyticsVisitorsHourly recomputes the hourly visitor aggregates of the
// last analyticsHourlyBuckets hours, or of the last KeyContextNumberOfDays days
// when set (the admin's manual resync). It keeps the current hours fresh;
// SyncAnalyticsRollups rebuilds every hour of its days, which repairs hours
// whose page views were ingested late.
func SyncAnalyticsVisitorsHourly(ctx context.Context) error {
	now := time.Now().UTC()

	if t, ok := ctx.Value(KeyContextNow{}).(time.Time); ok {
		now = t.UTC()
	}

	hour := now.Truncate(time.Hour)
	from := hour.Add(-(analyticsHourlyBuckets - 1) * time.Hour)

	if days, _ := ctx.Value(KeyContextNumberOfDays{}).(int); days > 0 {
		from = hour.AddDate(0, 0, -days)
	}

	err := NewStore().SyncAnalyticsVisitors(ctx, SyncAnalyticsVisitorsParams{
		From:   from,
		To:     hour.Add(time.Hour),
		Hourly: true,
	})

	if err != nil {
		slog.Errorf("could not sync hourly visitors data: %s", err.Error())
	}

	return err
}

// analyticsFinalizeAfter is how long after midnight a run must start to mark
// yesterday final, so page views and events still in the ingest queue at
// midnight are counted first.
const analyticsFinalizeAfter = 3 * time.Hour

// SyncAnalyticsRollups rolls raw page views and events up into the daily
// aggregates (visitors, referrers, countries, events, and the per-path,
// per-path-country, campaign and channel tables) and the hourly visitor
// buckets for today, and for yesterday
// until a run started at least analyticsFinalizeAfter past midnight has
// committed it. Earlier days are never recomputed.
func SyncAnalyticsRollups(ctx context.Context) error {
	now := time.Now().UTC()

	if t, ok := ctx.Value(KeyContextNow{}).(time.Time); ok {
		now = t.UTC()
	}

	store := NewStore()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	params := SyncAnalyticsRollupsParams{From: today, To: today.AddDate(0, 0, 1)}

	finalized, err := store.IsAnalyticsDayFinalized(ctx, yesterday)

	if err != nil {
		slog.Errorf("could not check finalized analytics day: %s", err.Error())
		return err
	}

	if !finalized {
		params.From = yesterday

		if now.Sub(today) >= analyticsFinalizeAfter {
			params.Finalize = yesterday
		}
	}

	if err := store.SyncAnalyticsRollups(ctx, params); err != nil {
		slog.Errorf("could not sync analytics rollups: %s", err.Error())
		return err
	}

	return nil
}
