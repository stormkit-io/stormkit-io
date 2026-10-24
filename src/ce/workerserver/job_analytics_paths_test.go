package jobs_test

import (
	"context"
	"fmt"
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
	Referrer string
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
			Referrer:    null.NewString(v.Referrer, v.Referrer != ""),
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

// sync runs the rollups as at noon today, when only today is recomputed.
func (s *JobAnalyticsPathsSuite) sync() error {
	return s.syncAt(s.today.Add(12 * time.Hour))
}

func (s *JobAnalyticsPathsSuite) syncAt(now time.Time) error {
	return jobs.SyncAnalyticsRollups(context.WithValue(context.Background(), jobs.KeyContextNow{}, now))
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

	s.Require().NoError(s.sync())

	// One IP with two browsers is two visitors; the 404 is not a page view.
	s.Equal([]pathAgg{
		{Date: s.day(0), Path: "/blog/post-1", UniqueVisitors: 3, PageViews: 4, ServerViews: 2, ClientViews: 2},
	}, s.paths())
}

func (s *JobAnalyticsPathsSuite) Test_Paths_RerunReplacesTheDay() {
	noon := s.today.Add(12 * time.Hour)

	s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/", At: noon})
	s.Require().NoError(s.sync())

	s.insert(visit{IP: "2.2.2.2", UA: "a", Path: "/", At: noon})
	s.Require().NoError(s.sync())

	s.Equal([]pathAgg{
		{Date: s.day(0), Path: "/", UniqueVisitors: 2, PageViews: 2, ServerViews: 2, ClientViews: 0},
	}, s.paths())
}

// Test_Paths_RecomputedDays verifies that yesterday is rebuilt until a run
// well after midnight commits it, after which only today is, and that older
// days are never rebuilt.
func (s *JobAnalyticsPathsSuite) Test_Paths_RecomputedDays() {
	yesterday := s.today.AddDate(0, 0, -1).Add(time.Hour)

	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/old", At: s.today.AddDate(0, 0, -2).Add(time.Hour)},
		visit{IP: "1.1.1.1", UA: "a", Path: "/yesterday", At: yesterday},
		visit{IP: "1.1.1.1", UA: "a", Path: "/", At: s.today.Add(time.Hour)},
	)

	views := func() map[string]int {
		out := map[string]int{}

		for _, p := range s.paths() {
			out[p.Path] = p.PageViews
		}

		return out
	}

	// Shortly after midnight yesterday is rebuilt but not yet final, since
	// its last page views may still be in the ingest queue.
	s.Require().NoError(s.syncAt(s.today.Add(10 * time.Minute)))
	s.insert(visit{IP: "2.2.2.2", UA: "a", Path: "/yesterday", At: yesterday})
	s.Require().NoError(s.syncAt(s.today.Add(20 * time.Minute)))
	s.Equal(map[string]int{"/yesterday": 2, "/": 1}, views())

	// A later run rebuilds and finalizes it; after that it is left alone.
	s.insert(visit{IP: "3.3.3.3", UA: "a", Path: "/yesterday", At: yesterday})
	s.Require().NoError(s.sync())
	s.insert(visit{IP: "4.4.4.4", UA: "a", Path: "/yesterday", At: yesterday})
	s.Require().NoError(s.sync())
	s.Equal(map[string]int{"/yesterday": 3, "/": 1}, views())
}

// Test_Paths_FoldsJunkPaths verifies that single-view paths beyond the limit,
// e.g. scanner probes, are folded into '(other)' in every path rollup, while
// paths with repeat views are kept.
func (s *JobAnalyticsPathsSuite) Test_Paths_FoldsJunkPaths() {
	noon := s.today.Add(12 * time.Hour)
	visits := []visit{
		{IP: "1.1.1.1", UA: "a", Path: "/pricing", At: noon, Referrer: "t.co"},
		{IP: "2.2.2.2", UA: "a", Path: "/pricing", At: noon},
	}

	for i := 0; i < 1005; i++ {
		visits = append(visits, visit{IP: "3.3.3.3", UA: "a", Path: fmt.Sprintf("/probe-%04d", i), At: noon, Referrer: "t.co"})
	}

	s.insert(visits...)
	s.Require().NoError(s.sync())

	byPath := map[string]pathAgg{}

	for _, p := range s.paths() {
		byPath[p.Path] = p
	}

	s.Len(byPath, 1002, "the repeated path, 1000 single-view paths and (other)")
	s.Equal(2, byPath["/pricing"].PageViews)
	s.Equal(5, byPath["(other)"].PageViews)

	var otherVisits int

	row := s.conn.QueryRow(`SELECT COALESCE(SUM(visits), 0) FROM analytics_channels_agg WHERE domain_id = $1 AND request_path = '(other)'`, s.domainID)
	s.Require().NoError(row.Scan(&otherVisits))
	s.Equal(5, otherVisits, "the channel rollup folds the same paths")

	// A re-run that folds a path kept earlier leaves no stale row behind.
	s.insert(visit{IP: "4.4.4.4", UA: "a", Path: "/probe-0000a", At: noon})
	s.Require().NoError(s.sync())

	var total int

	row = s.conn.QueryRow(`SELECT SUM(page_views) FROM analytics_paths_agg WHERE domain_id = $1`, s.domainID)
	s.Require().NoError(row.Scan(&total))
	s.Equal(1008, total, "every page view is counted exactly once")
}

func (s *JobAnalyticsPathsSuite) Test_Paths_NoPageViews() {
	s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/missing", At: s.today.Add(time.Hour), Status: http.StatusNotFound})

	s.Require().NoError(s.sync())
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
	s.Require().NoError(s.sync())

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

	s.Require().NoError(s.sync())

	s.Equal([]campaignAgg{
		{Date: s.day(0), Ref: "indiefounders", Path: "/pricing", UniqueVisitors: 1, Visits: 1},
		{Date: s.day(0), Source: "x", Medium: "social", Campaign: "launch", Path: "/", UniqueVisitors: 2, Visits: 3},
	}, s.campaigns())
}

// Test_Paths_Engagement verifies that foreground time is rolled up per path,
// counting a page view that reported several times once.
func (s *JobAnalyticsPathsSuite) Test_Paths_Engagement() {
	noon := s.today.Add(12 * time.Hour)
	at := utils.Unix{Time: noon, Valid: true}

	s.insert(
		visit{IP: "1.1.1.1", UA: "a", Path: "/blog", At: noon},
		visit{IP: "2.2.2.2", UA: "a", Path: "/blog", At: noon},
	)

	s.Require().NoError(analytics.NewStore().InsertEngagements(context.Background(), []analytics.Engagement{
		{DomainID: s.domainID, RequestPath: "/blog", ViewID: "a", Seconds: 30, EngagedTS: at},
		{DomainID: s.domainID, RequestPath: "/blog", ViewID: "a", Seconds: 10, EngagedTS: at},
		{DomainID: s.domainID, RequestPath: "/blog", ViewID: "b", Seconds: 20, EngagedTS: at},
		{DomainID: s.domainID, RequestPath: "/blog", ViewID: "c", Seconds: 99, EngagedTS: utils.Unix{Time: noon.AddDate(0, 0, -5), Valid: true}},
	}))

	s.Require().NoError(s.sync())

	var seconds, views int

	row := s.conn.QueryRow(`SELECT engaged_seconds, engaged_views FROM analytics_paths_agg WHERE domain_id = $1 AND request_path = '/blog'`, s.domainID)
	s.Require().NoError(row.Scan(&seconds, &views))
	s.Equal(60, seconds)
	s.Equal(2, views, "view a reported twice")
}

// channels returns the stored channel of every referrer host on the domain.
func (s *JobAnalyticsPathsSuite) channels() map[string]string {
	rows, err := s.conn.PrepareOrPanic(`SELECT referrer, channel FROM analytics_channels_agg WHERE domain_id = $1`).Query(s.domainID)
	s.Require().NoError(err)

	defer rows.Close()

	out := map[string]string{}

	for rows.Next() {
		var referrer, channel string
		s.Require().NoError(rows.Scan(&referrer, &channel))
		out[referrer] = channel
	}

	return out
}

func (s *JobAnalyticsPathsSuite) Test_Channels_Classification() {
	// Another domain of the same environment is the same site; a domain of
	// another environment is not.
	s.addDomain(s.envID, "www.example.org")
	other := s.MockEnv(s.MockApp(nil))
	s.addDomain(other.ID, "other.org")

	noon := s.today.Add(12 * time.Hour)
	referrers := map[string]string{
		"":                                    "direct",
		"paths.stormkit.io":                   "internal",
		"www.example.org":                     "internal",
		"blog.example.org":                    "internal",
		"evilexample.org":                     "referral",
		"other.org":                           "referral",
		"google.com":                          "search",
		"www.google.com":                      "search",
		"https://www.google.co.uk/search?q=x": "search",
		"com.google.android.googlequicksearchbox": "search",
		"accounts.google.com":                     "referral",
		"notgoogle.com":                           "referral",
		"gemini.google.com":                       "ai",
		"www.meta.ai":                             "ai",
		"chatgpt.com":                             "ai",
		"mail.google.com":                         "email",
		"com.google.android.gm":                   "email",
		"t.co":                                    "social",
		"news.ycombinator.com":                    "social",
	}

	for referrer := range referrers {
		s.insert(visit{IP: "1.1.1.1", UA: "a", Path: "/", At: noon, Referrer: referrer})
	}

	s.Require().NoError(s.sync())

	// Hosts are stored without scheme, path or "www.".
	expected := map[string]string{
		"":                  "direct",
		"paths.stormkit.io": "internal",
		"example.org":       "internal",
		"blog.example.org":  "internal",
		"evilexample.org":   "referral",
		"other.org":         "referral",
		"google.com":        "search",
		"google.co.uk":      "search",
		"com.google.android.googlequicksearchbox": "search",
		"accounts.google.com":                     "referral",
		"notgoogle.com":                           "referral",
		"gemini.google.com":                       "ai",
		"meta.ai":                                 "ai",
		"chatgpt.com":                             "ai",
		"mail.google.com":                         "email",
		"com.google.android.gm":                   "email",
		"t.co":                                    "social",
		"news.ycombinator.com":                    "social",
	}

	s.Equal(expected, s.channels())
}

func (s *JobAnalyticsPathsSuite) addDomain(envID types.ID, name string) {
	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), envID)
	s.Require().NoError(err)

	d := &buildconf.DomainModel{
		AppID:      env.AppID,
		EnvID:      envID,
		Name:       name,
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom(name),
	}

	s.Require().NoError(buildconf.DomainStore().Insert(context.Background(), d))
}

func TestJobAnalyticsPathsSuite(t *testing.T) {
	suite.Run(t, &JobAnalyticsPathsSuite{})
}
