package jobs_test

import (
	"context"
	"fmt"
	"time"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// pathDayTotals is one day of analytics_paths_agg summed over its paths: the
// way site-wide reports were computed before analytics_site_agg.
type pathDayTotals struct {
	views   int64
	seconds int64
	engaged int64
}

// seedSiteDays fills four days ending today with page views from both
// sources, foreground time, referrers, and on one day more single-view paths
// than a day keeps, so some are folded into '(other)'. It rolls them all up.
func (s *JobAnalyticsPathsSuite) seedSiteDays() {
	visits := []visit{}
	engagements := []analytics.Engagement{}

	for offset := -3; offset <= 0; offset++ {
		noon := s.today.AddDate(0, 0, offset).Add(12 * time.Hour)

		for i := 0; i < 5-offset; i++ {
			ip := fmt.Sprintf("10.0.%d.%d", -offset, i)
			visits = append(visits,
				visit{IP: ip, UA: "a", Path: "/", At: noon, Referrer: "google.com"},
				visit{IP: ip, UA: "a", Path: "/pricing", At: noon, Source: "client"},
				visit{IP: ip, UA: "a", Path: fmt.Sprintf("/blog/%d", i%3), At: noon, Referrer: "paths.stormkit.io"},
			)

			engagements = append(engagements, analytics.Engagement{
				DomainID:    s.domainID,
				RequestPath: "/pricing",
				ViewID:      fmt.Sprintf("view-%d-%d", offset, i),
				Seconds:     7*i + 3,
				EngagedTS:   utils.Unix{Time: noon, Valid: true},
			})
		}
	}

	for i := 0; i < 1005; i++ {
		visits = append(visits, visit{IP: "10.9.9.9", UA: "a", Path: fmt.Sprintf("/probe-%04d", i), At: s.today.AddDate(0, 0, -1).Add(time.Hour)})
	}

	s.insert(visits...)
	s.Require().NoError(analytics.NewStore().InsertEngagements(context.Background(), engagements))

	s.Require().NoError(jobs.NewStore().SyncAnalyticsRollups(context.Background(), jobs.SyncAnalyticsRollupsParams{
		From: s.today.AddDate(0, 0, -3),
		To:   s.today.AddDate(0, 0, 1),
	}))
}

// pathDays sums analytics_paths_agg per day for the views column of source.
func (s *JobAnalyticsPathsSuite) pathDays(source string) map[string]pathDayTotals {
	column := map[string]string{"": "page_views", analytics.SourceServer: "server_views", analytics.SourceClient: "client_views"}[source]
	rows, err := s.conn.Query(fmt.Sprintf(`
		SELECT aggregate_date::text, SUM(%s), SUM(engaged_seconds), SUM(engaged_views)
		FROM analytics_paths_agg WHERE domain_id = $1
		GROUP BY aggregate_date`, column), s.domainID)

	s.Require().NoError(err)

	defer rows.Close()

	out := map[string]pathDayTotals{}

	for rows.Next() {
		var date string
		var t pathDayTotals

		s.Require().NoError(rows.Scan(&date, &t.views, &t.seconds, &t.engaged))
		out[date] = t
	}

	return out
}

// sumDays totals the days [from, to] of days, with the average time on page.
func (s *JobAnalyticsPathsSuite) sumDays(days map[string]pathDayTotals, from, to int) (int64, *int64) {
	var total pathDayTotals

	for offset := from; offset <= to; offset++ {
		d := days[s.day(offset)]
		total.views += d.views
		total.seconds += d.seconds
		total.engaged += d.engaged
	}

	if total.engaged == 0 {
		return total.views, nil
	}

	avg := (total.seconds + total.engaged/2) / total.engaged

	return total.views, &avg
}

func (s *JobAnalyticsPathsSuite) report(metric, source string) *analytics.Report {
	report, err := analytics.NewStore().Report(context.Background(), analytics.ReportParams{
		Metric:    metric,
		DomainIDs: []types.ID{s.domainID},
		From:      s.today.AddDate(0, 0, -1),
		To:        s.today,
		Source:    source,
		Compare:   true,
		Limit:     analytics.ReportDefaultLimit,
	})

	s.Require().NoError(err)

	return report
}

// Test_SiteTotals_MatchPathSums verifies that the site-wide numbers read from
// analytics_site_agg equal the per-path sums they replace, for every source,
// folded paths included.
func (s *JobAnalyticsPathsSuite) Test_SiteTotals_MatchPathSums() {
	s.seedSiteDays()

	var other int64

	s.Require().NoError(s.conn.QueryRow(`SELECT COALESCE(SUM(page_views), 0) FROM analytics_paths_agg WHERE domain_id = $1 AND request_path = '(other)'`, s.domainID).Scan(&other))
	s.Equal(int64(5), other, "the seed folds paths into (other)")

	for _, source := range []string{"", analytics.SourceServer, analytics.SourceClient} {
		days := s.pathDays(source)
		views, avg := s.sumDays(days, -1, 0)
		prevViews, prevAvg := s.sumDays(days, -3, -2)

		s.NotZero(views, source)

		paths := s.report(analytics.MetricPaths, source)
		s.Equal(views, *paths.Totals.PageViews, "paths %q", source)
		s.Equal(avg, paths.Totals.AvgTimeOnPage, "paths %q", source)
		s.Equal(prevViews, *paths.Previous.Totals.PageViews, "paths previous %q", source)
		s.Equal(prevAvg, paths.Previous.Totals.AvgTimeOnPage, "paths previous %q", source)

		visitors := s.report(analytics.MetricVisitors, source)
		s.Equal(views, *visitors.Totals.PageViews, "visitors %q", source)
		s.Equal(avg, visitors.Totals.AvgTimeOnPage, "visitors %q", source)
		s.Equal(prevViews, *visitors.Previous.Totals.PageViews, "visitors previous %q", source)
		s.Equal(prevAvg, visitors.Previous.Totals.AvgTimeOnPage, "visitors previous %q", source)
		s.Require().Len(visitors.Series, 2)

		for i, day := range visitors.Series {
			dayViews, dayAvg := s.sumDays(days, i-1, i-1)

			s.Equal(s.day(i-1), day.Date)
			s.Equal(dayViews, day.PageViews, "%s %q", day.Date, source)
			s.Equal(dayAvg, day.AvgTimeOnPage, "%s %q", day.Date, source)
		}

		channels := s.report(analytics.MetricChannels, source)
		s.Equal(views, *channels.Totals.PageViews, "channels %q", source)
		s.Equal(prevViews, *channels.Previous.Totals.PageViews, "channels previous %q", source)
		s.Equal(visitors.Totals.Visitors, channels.Totals.Visitors, "channels %q", source)
		s.Equal(visitors.Previous.Totals.Visitors, channels.Previous.Totals.Visitors, "channels previous %q", source)
	}

	// Site-wide visitors still come from the domain's daily uniques: a visitor
	// of three pages on one day counts once.
	visitors := s.report(analytics.MetricVisitors, "")
	s.Equal(int64(5+7), *visitors.Totals.Visitors)
	s.Equal(int64(7+8), *visitors.Previous.Totals.Visitors)
}

// Test_SiteTotals_RerunReplacesTheDay verifies that a re-run rebuilds the
// site totals of the days it recomputes instead of adding to them.
func (s *JobAnalyticsPathsSuite) Test_SiteTotals_RerunReplacesTheDay() {
	noon := s.today.Add(12 * time.Hour)

	s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/", At: noon})
	s.Require().NoError(s.sync())

	s.insert(visit{IP: "2.2.2.2", UA: "a", Path: "/a", At: noon, Source: "client"})
	s.Require().NoError(s.sync())

	var views, server, client int

	s.Require().NoError(s.conn.QueryRow(`
		SELECT page_views, server_views, client_views
		FROM analytics_site_agg WHERE domain_id = $1 AND aggregate_date = $2::date`, s.domainID, s.day(0)).Scan(&views, &server, &client))

	s.Equal(2, views)
	s.Equal(1, server)
	s.Equal(1, client)
}
