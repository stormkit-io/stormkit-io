package jobs

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var (
	analyticsContext = context.Background()
)

type KeyContextNumberOfDays struct{}

// SyncAnalyticsVisitorsDaily syncs the daily analytics stats with the
// aggregation table. This allows us to effectively query the read table,
// while maintaining a raw write table that is easily scalable.
func SyncAnalyticsVisitorsDaily(ctx context.Context) error {
	store := NewStore()
	status := map[string][]int{
		"200": {200, 304},
		"404": {404},
	}

	days, _ := ctx.Value(KeyContextNumberOfDays{}).(int)

	if days <= 0 {
		days = 1
	}

	if days > 30 {
		days = 30
	}

	for statusCode, params := range status {
		tmpl, err := template.New("syncAnalytics").Parse(stmt.syncAnalyticsVisitors)

		if err != nil {
			slog.Errorf("cannot parse syncAnalytics template: %s", err.Error())
			return err
		}

		var qb strings.Builder

		data := map[string]any{
			"tableName": fmt.Sprintf("analytics_visitors_agg_%s", statusCode),
			"interval":  fmt.Sprintf("CURRENT_DATE - INTERVAL '%d days'", days),
			"column":    "DATE(a.request_timestamp)",
		}

		if err := tmpl.Execute(&qb, data); err != nil {
			slog.Errorf("error executing query template: %s", err.Error())
			return err
		}

		_, err = store.Exec(ctx, qb.String(), pq.Array(params))

		if err != nil {
			slog.Errorf("could not sync visitors data: %s", err.Error())
		}
	}

	return nil
}

// SyncAnalyticsVisitorsHourly syncs the analytics for the current day.
func SyncAnalyticsVisitorsHourly(ctx context.Context) error {
	store := NewStore()
	status := map[string][]int{
		"200": {200, 304},
		"404": {404},
	}

	for statusCode, params := range status {
		tmpl, err := template.New("syncAnalytics").Parse(stmt.syncAnalyticsVisitors)

		if err != nil {
			slog.Errorf("cannot parse syncAnalytics template: %s", err.Error())
			return err
		}

		var qb strings.Builder

		data := map[string]any{
			"tableName": fmt.Sprintf("analytics_visitors_agg_hourly_%s", statusCode),
			"interval":  "CURRENT_DATE - INTERVAL '1 days'",
			"column":    "TO_CHAR(DATE_TRUNC('hour', a.request_timestamp), 'YYYY-MM-DD HH24:MI')::timestamp",
		}

		if err := tmpl.Execute(&qb, data); err != nil {
			slog.Errorf("error executing query template: %s", err.Error())
			return err
		}

		_, err = store.Exec(ctx, qb.String(), pq.Array(params))

		if err != nil {
			slog.Errorf("could not sync visitors data: %s", err.Error())
		}
	}

	return nil
}

func SyncAnalyticsReferrers(ctx context.Context) error {
	_, err := NewStore().Exec(ctx, stmt.syncAnalyticsReferrers)

	if err != nil {
		slog.Errorf("could not sync referrers data: %s", err.Error())
	}

	return err
}

func SyncAnalyticsByCountries(ctx context.Context) error {
	_, err := NewStore().Exec(ctx, stmt.syncAnalyticsCountries)

	if err != nil {
		slog.Errorf("could not sync countries data: %s", err.Error())
	}

	return err
}

func SyncAnalyticsEvents(ctx context.Context) error {
	_, err := NewStore().Exec(ctx, stmt.syncAnalyticsEvents)

	if err != nil {
		slog.Errorf("could not sync events data: %s", err.Error())
	}

	return err
}

// analyticsPathsBackfillChunkDays bounds how many days one backfill query
// covers, so the first run after an upgrade does not hold one long query over
// the whole raw table.
const analyticsPathsBackfillChunkDays = 7

// SyncAnalyticsPaths rolls raw page views up into the per-path, per-path-country
// and campaign aggregates for yesterday and today, then backfills older days
// the aggregates do not cover yet.
//
// The backfill walks from the newest uncovered day to the oldest raw page view,
// so a run that fails part-way resumes below the oldest day it finished. Once
// raw rows age out, the aggregates reach further back than the raw table and
// nothing is left to backfill.
func SyncAnalyticsPaths(ctx context.Context) error {
	store := NewStore()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	if err := store.SyncAnalyticsPaths(ctx, SyncAnalyticsPathsParams{From: yesterday, To: today.AddDate(0, 0, 1)}); err != nil {
		slog.Errorf("could not sync path aggregates: %s", err.Error())
		return err
	}

	rawOldest, err := store.OldestRawPageViewDay(ctx)

	if err != nil || rawOldest == nil {
		return err
	}

	covered, err := store.OldestAnalyticsPathsDay(ctx)

	if err != nil {
		return err
	}

	end := yesterday

	if covered != nil && covered.Before(end) {
		end = *covered
	}

	for end.After(*rawOldest) {
		start := end.AddDate(0, 0, -analyticsPathsBackfillChunkDays)

		if start.Before(*rawOldest) {
			start = *rawOldest
		}

		if err := store.SyncAnalyticsPaths(ctx, SyncAnalyticsPathsParams{From: start, To: end}); err != nil {
			slog.Errorf("could not backfill path aggregates: %s", err.Error())
			return err
		}

		end = start
	}

	return nil
}
