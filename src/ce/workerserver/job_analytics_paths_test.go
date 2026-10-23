package jobs_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type JobAnalyticsPathsSuite struct {
	suite.Suite
	*factory.Factory

	conn     databasetest.TestDB
	appID    types.ID
	envID    types.ID
	domainID types.ID
	today    time.Time
}

type pathAgg struct {
	Date           string
	Path           string
	UniqueVisitors int
	PageViews      int
	ServerViews    int
	ClientViews    int
}

type pathCountryAgg struct {
	Date           string
	Path           string
	Country        string
	UniqueVisitors int
}

type campaignAgg struct {
	Date           string
	Source         string
	Medium         string
	Campaign       string
	Ref            string
	Path           string
	UniqueVisitors int
	Visits         int
}

// visit describes one raw page view for insert.
type visit struct {
	IP       string
	UA       string
	Path     string
	At       time.Time
	Source   string
	Status   int
	Campaign analytics.Campaign
}

func (s *JobAnalyticsPathsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	app := s.MockApp(nil)
	env := s.MockEnv(app)
	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "paths.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("paths-token"),
	}

	s.Require().NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	s.appID = app.ID
	s.envID = env.ID
	s.domainID = domain.ID
	s.today = time.Now().UTC().Truncate(24 * time.Hour)
}

func (s *JobAnalyticsPathsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *JobAnalyticsPathsSuite) insert(visits ...visit) {
	records := []analytics.Record{}

	for _, v := range visits {
		status := v.Status

		if status == 0 {
			status = http.StatusOK
		}

		source := v.Source

		if source == "" {
			source = "server"
		}

		records = append(records, analytics.Record{
			AppID:       s.appID,
			EnvID:       s.envID,
			DomainID:    s.domainID,
			VisitorIP:   v.IP,
			UserAgent:   null.StringFrom(v.UA),
			RequestPath: v.Path,
			RequestTS:   utils.Unix{Time: v.At, Valid: true},
			StatusCode:  status,
			Source:      null.StringFrom(source),
			Campaign:    v.Campaign,
		})
	}

	s.Require().NoError(analytics.NewStore().InsertRecords(context.Background(), records))
}

func (s *JobAnalyticsPathsSuite) paths() []pathAgg {
	rows, err := s.conn.PrepareOrPanic(`
		SELECT aggregate_date::text, request_path, unique_visitors, page_views, server_views, client_views
		FROM analytics_paths_agg WHERE domain_id = $1
		ORDER BY aggregate_date, request_path`).Query(s.domainID)

	s.Require().NoError(err)

	defer rows.Close()

	out := []pathAgg{}

	for rows.Next() {
		var r pathAgg
		s.Require().NoError(rows.Scan(&r.Date, &r.Path, &r.UniqueVisitors, &r.PageViews, &r.ServerViews, &r.ClientViews))
		out = append(out, r)
	}

	return out
}

func (s *JobAnalyticsPathsSuite) pathCountries() []pathCountryAgg {
	rows, err := s.conn.PrepareOrPanic(`
		SELECT aggregate_date::text, request_path, country_iso_code, unique_visitors
		FROM analytics_path_countries_agg WHERE domain_id = $1
		ORDER BY aggregate_date, request_path, country_iso_code`).Query(s.domainID)

	s.Require().NoError(err)

	defer rows.Close()

	out := []pathCountryAgg{}

	for rows.Next() {
		var r pathCountryAgg
		s.Require().NoError(rows.Scan(&r.Date, &r.Path, &r.Country, &r.UniqueVisitors))
		out = append(out, r)
	}

	return out
}

func (s *JobAnalyticsPathsSuite) campaigns() []campaignAgg {
	rows, err := s.conn.PrepareOrPanic(`
		SELECT aggregate_date::text, utm_source, utm_medium, utm_campaign, ref, request_path, unique_visitors, visits
		FROM analytics_campaigns_agg WHERE domain_id = $1
		ORDER BY aggregate_date, utm_source, ref, request_path`).Query(s.domainID)

	s.Require().NoError(err)

	defer rows.Close()

	out := []campaignAgg{}

	for rows.Next() {
		var r campaignAgg
		s.Require().NoError(rows.Scan(&r.Date, &r.Source, &r.Medium, &r.Campaign, &r.Ref, &r.Path, &r.UniqueVisitors, &r.Visits))
		out = append(out, r)
	}

	return out
}

func (s *JobAnalyticsPathsSuite) day(offset int) string {
	return s.today.AddDate(0, 0, offset).Format(time.DateOnly)
}

func (s *JobAnalyticsPathsSuite) Test_Paths_CountsViewsVisitorsAndSources() {
	noon := s.today.Add(12 * time.Hour)

	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/blog/post-1", At: noon},
		visit{IP: "1.1.1.1", UA: "a", Path: "/blog/post-1", At: noon, Source: "client"},
		visit{IP: "2.2.2.2", UA: "a", Path: "/blog/post-1", At: noon},
		visit{IP: "1.1.1.1", UA: "b", Path: "/blog/post-1", At: noon, Source: "client"},
		visit{IP: "3.3.3.3", UA: "a", Path: "/missing", At: noon, Status: http.StatusNotFound},
	)

	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	// One IP with two browsers is two visitors; the 404 is not a page view.
	s.Equal([]pathAgg{
		{Date: s.day(0), Path: "/blog/post-1", UniqueVisitors: 3, PageViews: 4, ServerViews: 2, ClientViews: 2},
	}, s.paths())
}

func (s *JobAnalyticsPathsSuite) Test_Paths_RerunReplacesTheDay() {
	noon := s.today.Add(12 * time.Hour)

	s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/", At: noon})
	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.insert(visit{IP: "2.2.2.2", UA: "a", Path: "/", At: noon})
	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.Equal([]pathAgg{
		{Date: s.day(0), Path: "/", UniqueVisitors: 2, PageViews: 2, ServerViews: 2, ClientViews: 0},
	}, s.paths())
}

// Test_Paths_BackfillsOnFirstRun verifies that the first run rolls up every
// day still in the raw table, while later runs only touch yesterday and today.
func (s *JobAnalyticsPathsSuite) Test_Paths_BackfillsOnFirstRun() {
	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/old", At: s.today.AddDate(0, 0, -20).Add(time.Hour)},
		visit{IP: "1.1.1.1", UA: "a", Path: "/", At: s.today.Add(time.Hour)},
	)

	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.Equal([]string{"/old", "/"}, []string{s.paths()[0].Path, s.paths()[1].Path})

	s.insert(visit{IP: "2.2.2.2", UA: "a", Path: "/old", At: s.today.AddDate(0, 0, -20).Add(2 * time.Hour)})
	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.Equal(1, s.paths()[0].PageViews, "a later run does not recompute old days")
}

// Test_Paths_BackfillResumes verifies that a backfill that stopped part-way
// fills the remaining older days on the next run instead of leaving a gap.
func (s *JobAnalyticsPathsSuite) Test_Paths_BackfillResumes() {
	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/oldest", At: s.today.AddDate(0, 0, -30).Add(time.Hour)},
		visit{IP: "1.1.1.1", UA: "a", Path: "/older", At: s.today.AddDate(0, 0, -12).Add(time.Hour)},
		visit{IP: "1.1.1.1", UA: "a", Path: "/", At: s.today.Add(time.Hour)},
	)

	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	// Simulate a first run that only finished the newest chunks.
	_, err := s.conn.Exec(`DELETE FROM analytics_paths_agg WHERE domain_id = $1 AND aggregate_date < $2::date`, s.domainID, s.day(-10))
	s.Require().NoError(err)

	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	paths := []string{}

	for _, p := range s.paths() {
		paths = append(paths, p.Path)
	}

	s.Equal([]string{"/oldest", "/older", "/"}, paths)
}

// Test_Paths_AggregatesOlderThanRawRows verifies that once raw rows have aged
// out, aggregates reaching further back do not trigger a backfill.
func (s *JobAnalyticsPathsSuite) Test_Paths_AggregatesOlderThanRawRows() {
	_, err := s.conn.Exec(`
		INSERT INTO analytics_paths_agg
			(aggregate_date, domain_id, request_path, request_path_hash, unique_visitors, page_views, server_views, client_views)
		VALUES ($1::date, $2, '/ancient', decode(md5('/ancient'), 'hex'), 9, 9, 9, 0)`, s.day(-400), s.domainID)

	s.Require().NoError(err)

	s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/", At: s.today.Add(time.Hour)})
	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.Equal([]pathAgg{
		{Date: s.day(-400), Path: "/ancient", UniqueVisitors: 9, PageViews: 9, ServerViews: 9, ClientViews: 0},
		{Date: s.day(0), Path: "/", UniqueVisitors: 1, PageViews: 1, ServerViews: 1, ClientViews: 0},
	}, s.paths())
}

func (s *JobAnalyticsPathsSuite) Test_Paths_NoPageViews() {
	s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/missing", At: s.today.Add(time.Hour), Status: http.StatusNotFound})

	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))
	s.Empty(s.paths())
}

func (s *JobAnalyticsPathsSuite) Test_PathCountries() {
	noon := s.today.Add(12 * time.Hour)

	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/blog", At: noon},
		visit{IP: "1.1.1.1", UA: "a", Path: "/blog", At: noon},
		visit{IP: "2.2.2.2", UA: "a", Path: "/blog", At: noon},
		visit{IP: "3.3.3.3", UA: "a", Path: "/blog", At: noon},
	)

	_, err := s.conn.Exec(`UPDATE analytics SET country_iso_code = CASE WHEN visitor_ip = (
		SELECT visitor_ip FROM analytics WHERE domain_id = $1 ORDER BY analytics_id LIMIT 1
	) THEN 'TR' ELSE 'DE' END WHERE domain_id = $1`, s.domainID)

	s.Require().NoError(err)
	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.Equal([]pathCountryAgg{
		{Date: s.day(0), Path: "/blog", Country: "DE", UniqueVisitors: 2},
		{Date: s.day(0), Path: "/blog", Country: "TR", UniqueVisitors: 1},
	}, s.pathCountries())
}

func (s *JobAnalyticsPathsSuite) Test_Campaigns() {
	noon := s.today.Add(12 * time.Hour)
	launch := analytics.CampaignFromRawQuery("utm_source=x&utm_medium=social&utm_campaign=launch")
	directory := analytics.CampaignFromRawQuery("ref=indiefounders")

	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/", At: noon, Campaign: launch},
		visit{IP: "1.1.1.1", UA: "a", Path: "/", At: noon, Campaign: launch},
		visit{IP: "2.2.2.2", UA: "a", Path: "/", At: noon, Campaign: launch},
		visit{IP: "3.3.3.3", UA: "a", Path: "/pricing", At: noon, Campaign: directory},
		visit{IP: "4.4.4.4", UA: "a", Path: "/", At: noon},
	)

	s.Require().NoError(jobs.SyncAnalyticsPaths(context.Background()))

	s.Equal([]campaignAgg{
		{Date: s.day(0), Ref: "indiefounders", Path: "/pricing", UniqueVisitors: 1, Visits: 1},
		{Date: s.day(0), Source: "x", Medium: "social", Campaign: "launch", Path: "/", UniqueVisitors: 2, Visits: 3},
	}, s.campaigns())
}

func TestJobAnalyticsPathsSuite(t *testing.T) {
	suite.Run(t, &JobAnalyticsPathsSuite{})
}
