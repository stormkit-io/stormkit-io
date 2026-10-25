package analytics_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type ReportSuite struct {
	suite.Suite

	conn    databasetest.TestDB
	domain  types.ID
	other   types.ID
	today   time.Time
	dayFrom func(int) string
}

func (s *ReportSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)

	// The aggregate tables carry no foreign keys, so plain ids are enough.
	s.domain = types.ID(9001)
	s.other = types.ID(9002)
	s.today = time.Now().UTC().Truncate(24 * time.Hour)
	s.dayFrom = func(offset int) string { return s.today.AddDate(0, 0, offset).Format(time.DateOnly) }
}

func (s *ReportSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *ReportSuite) exec(query string, args ...any) {
	_, err := s.conn.Exec(query, args...)
	s.Require().NoError(err)
}

// pathDay seeds analytics_paths_agg.
func (s *ReportSuite) pathDay(domain types.ID, offset int, path string, visitors, server, client int) {
	s.exec(`INSERT INTO analytics_paths_agg
		(aggregate_date, domain_id, request_path, request_path_hash, unique_visitors, page_views, server_views, client_views)
		VALUES ($1::date, $2, $3, decode(md5($3), 'hex'), $4, $5, $6, $7)`,
		s.dayFrom(offset), domain, path, visitors, server+client, server, client)

	s.syncSite()
}

// syncSite sums the seeded paths into analytics_site_agg, as the rollup does.
func (s *ReportSuite) syncSite() {
	s.exec(`INSERT INTO analytics_site_agg
		SELECT domain_id, aggregate_date, SUM(page_views), SUM(server_views), SUM(client_views), SUM(engaged_seconds), SUM(engaged_views)
		FROM analytics_paths_agg
		GROUP BY domain_id, aggregate_date
		ON CONFLICT (domain_id, aggregate_date) DO UPDATE SET
			page_views = EXCLUDED.page_views,
			server_views = EXCLUDED.server_views,
			client_views = EXCLUDED.client_views,
			engaged_seconds = EXCLUDED.engaged_seconds,
			engaged_views = EXCLUDED.engaged_views`)
}

// siteDay seeds the domain-level daily visitors aggregate.
func (s *ReportSuite) siteDay(domain types.ID, offset, visitors, views int) {
	s.exec(`INSERT INTO analytics_visitors_agg_200 (aggregate_date, domain_id, unique_visitors, total_visitors)
		VALUES ($1::date, $2, $3, $4)`, s.dayFrom(offset), domain, visitors, views)
}

// testChannels is the channel the rollup job assigns to each referrer host the
// report tests use; classification itself is tested with the job.
var testChannels = map[string]string{
	"":                 "direct",
	"example.org":      "internal",
	"blog.example.org": "internal",
	"google.com":       "search",
	"bing.com":         "search",
	"x.com":            "social",
	"t.co":             "social",
	"chatgpt.com":      "ai",
	"evilexample.org":  "referral",
}

// referrer seeds analytics_channels_agg as the rollup job would.
func (s *ReportSuite) referrer(offset int, referrer, path string, visits int) {
	channel, ok := testChannels[referrer]
	s.Require().True(ok, "no test channel for %q", referrer)

	s.exec(`INSERT INTO analytics_channels_agg
		(aggregate_date, domain_id, request_path, request_path_hash, referrer, referrer_hash, channel, visits)
		VALUES ($1::date, $2, $3, decode(md5($3), 'hex'), $4, decode(md5($4), 'hex'), $5, $6)`,
		s.dayFrom(offset), s.domain, path, referrer, channel, visits)
}

func (s *ReportSuite) report(query string, domains ...types.ID) *analytics.Report {
	q, err := url.ParseQuery(query)
	s.Require().NoError(err)

	params, errs := analytics.ReportParamsFromQuery(q)
	s.Require().Empty(errs)

	if domains == nil {
		domains = []types.ID{s.domain}
	}

	params.DomainIDs = domains
	report, err := analytics.NewStore().Report(context.Background(), params)
	s.Require().NoError(err)

	return report
}

func ptr(v int64) *int64 {
	return &v
}

func (s *ReportSuite) Test_Visitors_SiteWide() {
	s.pathDay(s.domain, 0, "/", 3, 4, 1)
	s.pathDay(s.domain, 0, "/blog", 2, 2, 0)
	s.pathDay(s.domain, -2, "/", 1, 1, 0)
	s.siteDay(s.domain, 0, 4, 7)
	s.siteDay(s.domain, -2, 1, 1)
	s.pathDay(s.other, 0, "/", 50, 50, 0)

	r := s.report("metric=visitors&span=3d")

	s.Equal(analytics.ReportRange{From: s.dayFrom(-2), To: s.dayFrom(0), Days: 3}, r.Range)
	s.Equal([]analytics.ReportDay{
		{Date: s.dayFrom(-2), PageViews: 1, Visitors: ptr(1)},
		{Date: s.dayFrom(-1), PageViews: 0, Visitors: ptr(0)},
		{Date: s.dayFrom(0), PageViews: 7, Visitors: ptr(4)},
	}, r.Series)

	// Site-wide visitors come from the domain aggregate, not a sum of pages.
	s.Equal(analytics.ReportTotals{PageViews: ptr(8), Visitors: ptr(5)}, r.Totals)
	s.Contains(r.Notes[0], "sum of daily unique visitors")
}

func (s *ReportSuite) Test_Visitors_ExactPathAndPrefix() {
	s.pathDay(s.domain, 0, "/blog/a", 3, 3, 0)
	s.pathDay(s.domain, 0, "/blog/b", 2, 1, 1)
	s.pathDay(s.domain, 0, "/about", 9, 9, 0)

	exact := s.report("metric=visitors&span=1d&path=/blog/a")
	s.Equal(analytics.ReportTotals{PageViews: ptr(3), Visitors: ptr(3)}, exact.Totals)

	prefix := s.report("metric=visitors&span=1d&pathPrefix=/blog/")
	s.Equal(analytics.ReportTotals{PageViews: ptr(5), PageVisitors: ptr(5)}, prefix.Totals)
	s.Contains(prefix.Notes[0], "pageVisitors")
}

func (s *ReportSuite) Test_Visitors_SourceFiltersPageViews() {
	s.pathDay(s.domain, 0, "/", 3, 4, 2)
	s.siteDay(s.domain, 0, 3, 6)

	r := s.report("metric=visitors&span=1d&source=client")

	s.Equal(analytics.ReportTotals{PageViews: ptr(2), Visitors: ptr(3)}, r.Totals)
	s.Contains(r.Notes, "source filters page views only; visitor and visit counts include both sources.")
}

// engagedDay seeds foreground time on a path's day row.
func (s *ReportSuite) engagedDay(offset int, path string, seconds, views int) {
	s.exec(`UPDATE analytics_paths_agg SET engaged_seconds = $4, engaged_views = $5
		WHERE domain_id = $1 AND aggregate_date = $2::date AND request_path = $3`,
		s.domain, s.dayFrom(offset), path, seconds, views)

	s.syncSite()
}

func (s *ReportSuite) Test_TimeOnPage() {
	s.pathDay(s.domain, 0, "/blog/a", 3, 4, 0)
	s.pathDay(s.domain, 0, "/blog/b", 1, 2, 0)
	s.pathDay(s.domain, 0, "/about", 1, 1, 0)
	s.pathDay(s.domain, -1, "/blog/a", 1, 1, 0)
	s.engagedDay(0, "/blog/a", 100, 3)
	s.engagedDay(0, "/blog/b", 20, 2)
	s.engagedDay(-1, "/blog/a", 50, 1)

	paths := s.report("metric=paths&span=1d&compare=previous")
	byPath := map[string]*int64{}

	for _, row := range paths.Rows {
		byPath[row.Path] = row.AvgTimeOnPage
	}

	s.Equal(ptr(33), byPath["/blog/a"], "100s over 3 views, rounded")
	s.Equal(ptr(10), byPath["/blog/b"])
	s.Nil(byPath["/about"], "absent when no page view reported time")
	s.Equal(ptr(24), paths.Totals.AvgTimeOnPage, "120s over 5 views")
	s.Equal(ptr(50), paths.Previous.Totals.AvgTimeOnPage)
	s.Contains(paths.Notes, "avgTimeOnPage is the average foreground time per page view, in seconds, measured by the client script; it is absent when no page view reported it.")

	visitors := s.report("metric=visitors&span=2d&pathPrefix=/blog/")

	s.Equal(ptr(28), visitors.Totals.AvgTimeOnPage, "170s over 6 views")
	s.Equal(ptr(50), visitors.Series[0].AvgTimeOnPage)
	s.Equal(ptr(24), visitors.Series[1].AvgTimeOnPage)
}

func (s *ReportSuite) Test_Paths_RankedAndLimited() {
	s.pathDay(s.domain, 0, "/blog/a", 3, 5, 0)
	s.pathDay(s.domain, -1, "/blog/a", 2, 2, 0)
	s.pathDay(s.domain, 0, "/blog/b", 1, 6, 0)
	s.pathDay(s.domain, 0, "/about", 1, 1, 0)

	r := s.report("metric=paths&span=7d&pathPrefix=/blog/&limit=1")

	s.Require().Len(r.Rows, 1)
	s.Equal("/blog/a", r.Rows[0].Path)
	s.Equal(ptr(7), r.Rows[0].PageViews)
	s.Equal(ptr(5), r.Rows[0].Visitors)

	// Totals cover every matching path, not only the returned rows.
	s.Equal(ptr(13), r.Totals.PageViews)
}

func (s *ReportSuite) Test_Referrers_ForOnePath() {
	s.referrer(0, "google.com", "/blog/a", 4)
	s.referrer(0, "x.com", "/blog/a", 1)
	s.referrer(0, "", "/blog/a", 2)
	s.referrer(0, "google.com", "/about", 10)

	r := s.report("metric=referrers&span=7d&path=/blog/a")

	s.Equal([]string{"google.com", "x.com"}, []string{r.Rows[0].Referrer, r.Rows[1].Referrer})
	s.Equal(ptr(4), r.Rows[0].Visits)
	s.Equal(analytics.ReportTotals{Visits: ptr(7), Direct: ptr(2), Internal: ptr(0)}, r.Totals)
	s.Equal("search", r.Rows[0].Channel)
	s.Equal("social", r.Rows[1].Channel)
}

// Test_Referrers_ExcludeInternal verifies that internal referrers are left out
// of the rows and counted in totals.internal.
func (s *ReportSuite) Test_Referrers_ExcludeInternal() {
	s.referrer(0, "example.org", "/a", 20)
	s.referrer(0, "blog.example.org", "/a", 5)
	s.referrer(0, "evilexample.org", "/a", 1)
	s.referrer(0, "google.com", "/a", 3)

	r := s.report("metric=referrers&span=1d")

	s.Equal([]string{"google.com", "evilexample.org"}, []string{r.Rows[0].Referrer, r.Rows[1].Referrer})
	s.Equal("referral", r.Rows[1].Channel)
	s.Equal(analytics.ReportTotals{Visits: ptr(29), Direct: ptr(0), Internal: ptr(25)}, r.Totals)
}

// Test_Referrers_OneRowPerReferrer verifies that a referrer whose channel
// changed within the range is one row, labelled with its main channel.
func (s *ReportSuite) Test_Referrers_OneRowPerReferrer() {
	s.exec(`INSERT INTO analytics_channels_agg
		(aggregate_date, domain_id, request_path, request_path_hash, referrer, referrer_hash, channel, visits)
		VALUES
		($1::date, $3, '/', decode(md5('/'), 'hex'), 'new.example', decode(md5('new.example'), 'hex'), 'referral', 4),
		($2::date, $3, '/', decode(md5('/'), 'hex'), 'new.example', decode(md5('new.example'), 'hex'), 'social', 1)`,
		s.dayFrom(-1), s.dayFrom(0), s.domain)

	r := s.report("metric=referrers&span=2d&compare=previous")

	s.Require().Len(r.Rows, 1)
	s.Equal("new.example", r.Rows[0].Referrer)
	s.Equal("referral", r.Rows[0].Channel)
	s.Equal(ptr(5), r.Rows[0].Visits)
}

func (s *ReportSuite) Test_Channels_GroupsAndTotals() {
	s.referrer(0, "", "/blog/a", 1)
	s.referrer(0, "example.org", "/blog/a", 2)
	s.referrer(0, "google.com", "/blog/a", 3)
	s.referrer(0, "bing.com", "/blog/a", 4)
	s.referrer(0, "chatgpt.com", "/blog/a", 5)
	s.referrer(0, "t.co", "/about", 50)
	s.pathDay(s.domain, 0, "/blog/a", 9, 15, 0)

	r := s.report("metric=channels&span=1d&path=/blog/a")

	s.Equal([]string{"search", "ai", "internal", "direct"}, []string{r.Rows[0].Channel, r.Rows[1].Channel, r.Rows[2].Channel, r.Rows[3].Channel})
	s.Equal(ptr(7), r.Rows[0].Visits)

	// One call also answers how many people read the page.
	s.Equal(analytics.ReportTotals{PageViews: ptr(15), Visitors: ptr(9), Visits: ptr(15)}, r.Totals)

	withSource := s.report("metric=channels&span=1d&path=/blog/a&source=client")
	s.Contains(withSource.Notes, "source filters page views only; visitor and visit counts include both sources.")
}

func (s *ReportSuite) Test_Channels_ComparePrevious() {
	s.referrer(0, "google.com", "/", 5)
	s.referrer(-1, "google.com", "/", 2)
	s.referrer(-1, "t.co", "/", 9)

	r := s.report("metric=channels&span=1d&compare=previous")

	s.Require().Len(r.Rows, 1)
	s.Equal("search", r.Rows[0].Channel)
	s.Equal(ptr(2), r.Rows[0].Previous)
	s.Equal(ptr(11), r.Previous.Totals.Visits)
}

// Test_Entries verifies that landing pages count only visits arriving from
// outside the site, with their channel breakdown.
func (s *ReportSuite) Test_Entries() {
	s.referrer(0, "google.com", "/blog/a", 4)
	s.referrer(0, "", "/blog/a", 2)
	s.referrer(0, "example.org", "/blog/a", 50)
	s.referrer(0, "t.co", "/pricing", 3)
	s.referrer(0, "example.org", "/pricing", 70)
	s.referrer(0, "example.org", "/docs", 90)

	r := s.report("metric=entries&span=1d")

	s.Require().Len(r.Rows, 2, "a page reached only from inside the site is no entry")
	s.Equal("/blog/a", r.Rows[0].Path)
	s.Equal(ptr(6), r.Rows[0].Visits)
	s.Equal(map[string]int64{"search": 4, "direct": 2}, r.Rows[0].Channels)
	s.Equal(map[string]int64{"social": 3}, r.Rows[1].Channels)
	s.Equal(ptr(9), r.Totals.Visits)

	limited := s.report("metric=entries&span=1d&limit=1&compare=previous")

	s.Require().Len(limited.Rows, 1)
	s.Equal(ptr(0), limited.Rows[0].Previous)
}

func (s *ReportSuite) Test_Countries_SiteWideAndPerPath() {
	s.exec(`INSERT INTO analytics_visitors_by_countries (aggregate_date, country_iso_code, domain_id, visit_count)
		VALUES ($1::date, 'DE', $2, 5), ($1::date, 'TR', $2, 7)`, s.dayFrom(0), s.domain)
	s.exec(`INSERT INTO analytics_path_countries_agg
		(aggregate_date, domain_id, request_path, request_path_hash, country_iso_code, unique_visitors)
		VALUES ($1::date, $2, '/blog', decode(md5('/blog'), 'hex'), 'DE', 3)`, s.dayFrom(0), s.domain)

	site := s.report("metric=countries&span=1d")
	s.Equal([]string{"TR", "DE"}, []string{site.Rows[0].Country, site.Rows[1].Country})

	path := s.report("metric=countries&span=1d&path=/blog")
	s.Require().Len(path.Rows, 1)
	s.Equal(ptr(3), path.Rows[0].Visitors)

	prefix := s.report("metric=countries&span=1d&pathPrefix=/bl")
	s.Require().Len(prefix.Rows, 1)
	s.Nil(prefix.Rows[0].Visitors)
	s.Equal(ptr(3), prefix.Rows[0].PageVisitors)
	s.Contains(prefix.Notes[0], "pageVisitors")
}

func (s *ReportSuite) Test_Campaigns() {
	s.exec(`INSERT INTO analytics_campaigns_agg
		(aggregate_date, domain_id, utm_source, utm_medium, utm_campaign, ref, campaign_hash,
		 request_path, request_path_hash, unique_visitors, visits)
		VALUES
		($1::date, $2, 'x', 'social', 'launch', '', decode(md5('a'), 'hex'), '/', decode(md5('/'), 'hex'), 2, 3),
		($1::date, $2, 'x', 'social', 'launch', '', decode(md5('a'), 'hex'), '/pricing', decode(md5('/pricing'), 'hex'), 1, 1),
		($1::date, $2, '', '', '', 'hn', decode(md5('b'), 'hex'), '/', decode(md5('/'), 'hex'), 5, 5)`,
		s.dayFrom(0), s.domain)

	r := s.report("metric=campaigns&span=1d")

	s.Require().Len(r.Rows, 2)
	s.Equal("hn", r.Rows[0].Ref)
	s.Equal("launch", r.Rows[1].UTMCampaign)
	s.Equal(ptr(4), r.Rows[1].Visits, "landing paths are summed per campaign")
	s.Equal(ptr(9), r.Totals.Visits)

	// Visitors summed over landing pages would count a person once per page,
	// so they are only reported for one landing path.
	s.Nil(r.Rows[1].Visitors)
	s.Contains(r.Notes[0], "visitors are only reported with an exact path")

	landing := s.report("metric=campaigns&span=1d&path=/")

	s.Equal("launch", landing.Rows[1].UTMCampaign)
	s.Equal(ptr(2), landing.Rows[1].Visitors)
}

func (s *ReportSuite) Test_Events() {
	s.exec(`INSERT INTO analytics_events_agg (aggregate_date, domain_id, event_name, total_count, unique_actors)
		VALUES ($1::date, $2, 'signup', 4, 3), ($1::date, $2, 'visit_site', 9, 2)`, s.dayFrom(0), s.domain)

	r := s.report("metric=events&span=1d&path=/x")

	s.Equal([]string{"visit_site", "signup"}, []string{r.Rows[0].Event, r.Rows[1].Event})
	s.Equal(ptr(13), r.Totals.Count)
	s.Contains(r.Notes, "events are counted for the whole site; path filters do not apply.")
}

// Test_ComparePrevious verifies that the previous period's totals and each
// row's previous number are returned, including rows outside last period's
// top N.
func (s *ReportSuite) Test_ComparePrevious() {
	s.pathDay(s.domain, 0, "/a", 1, 5, 0)
	s.pathDay(s.domain, -6, "/a", 1, 2, 0)
	s.pathDay(s.domain, -6, "/b", 1, 9, 0)
	s.pathDay(s.domain, -6, "/c", 1, 8, 0)

	r := s.report("metric=paths&span=3d&compare=previous&limit=1")

	s.Require().NotNil(r.Previous)
	s.Equal(analytics.ReportRange{From: s.dayFrom(-5), To: s.dayFrom(-3), Days: 3}, r.Previous.Range)
	s.Equal(ptr(0), r.Previous.Totals.PageViews)

	wide := s.report("metric=paths&span=5d&compare=previous&limit=1")

	s.Equal("/a", wide.Rows[0].Path)
	s.Equal(ptr(2), wide.Rows[0].Previous, "found even though /a was third in the previous period")
	s.Equal(ptr(19), wide.Previous.Totals.PageViews)
}

// Test_ComparePrevious_Referrers verifies the previous-period lookup for keys
// other than paths, including a key with no previous traffic.
func (s *ReportSuite) Test_ComparePrevious_Referrers() {
	s.referrer(0, "google.com", "/", 5)
	s.referrer(0, "x.com", "/", 3)
	s.referrer(-1, "google.com", "/", 2)
	s.referrer(-1, "bing.com", "/", 9)

	r := s.report("metric=referrers&span=1d&compare=previous")

	s.Require().Len(r.Rows, 2)
	s.Equal(ptr(2), r.Rows[0].Previous)
	s.Equal(ptr(0), r.Rows[1].Previous)
	s.Equal(ptr(11), r.Previous.Totals.Visits)
}

func (s *ReportSuite) Test_NoDomains() {
	r := s.report("metric=visitors", []types.ID{}...)

	s.Contains(r.Notes, "this environment has no custom domains; analytics are only recorded for custom domains.")
}

func (s *ReportSuite) Test_ParamsValidation() {
	cases := map[string]string{
		"metric=bogus":                  "metric must be one of",
		"path=/a&pathPrefix=/b":         "path and pathPrefix cannot be combined",
		"path=blog":                     "must start with '/'",
		"source=edge":                   "source must be server or client",
		"compare=yoy":                   "compare must be 'previous'",
		"limit=500":                     "limit must be between 1 and 50",
		"span=2w":                       "span must be a number of days",
		"from=2026-01-01&span=7d":       "from and span cannot be combined",
		"from=yesterday":                "from must be a date",
		"from=2026-02-01&to=2026-01-01": "from must not be after to",
		"from=2024-01-01&to=2026-01-01": "a report covers at most 400 days",
	}

	for query, want := range cases {
		q, err := url.ParseQuery(query)
		s.Require().NoError(err)

		_, errs := analytics.ReportParamsFromQuery(q)

		s.Require().NotEmpty(errs, query)
		s.Contains(errs[0], want, query)
	}
}

func TestReportSuite(t *testing.T) {
	suite.Run(t, new(ReportSuite))
}
