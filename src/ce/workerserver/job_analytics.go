package jobs

import (
	"context"
	"fmt"
	"strings"
	"text/template"

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
