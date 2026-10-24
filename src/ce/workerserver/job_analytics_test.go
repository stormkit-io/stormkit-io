package jobs_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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

type JobAnalyticsSuite struct {
	suite.Suite
	*factory.Factory

	conn        databasetest.TestDB
	domainID    types.ID
	yesterday   string
	fiveDaysAgo string
	today       string
	randomToken string
}

func (s *JobAnalyticsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	app := s.MockApp(nil)
	env := s.MockEnv(app)
	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("my-custom-token"),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	t1 := time.Now()
	now := time.Date(t1.Year(), t1.Month(), t1.Day(), 15, 30, 0, 0, t1.Location())
	yesterday := now.Add(-24 * time.Hour)
	fiveDaysAgo := now.Add(-5 * 24 * time.Hour)

	hash := map[string][]string{
		"86.8.6.85":       {"2024-09-21 17:46:24", now.Format(time.DateTime)},
		"188.167.251.87":  {"2024-09-21 18:59:06", "2024-09-21 14:50:26"},
		"181.130.251.87":  {fiveDaysAgo.Format(time.DateTime)},
		"171.22.106.134":  {yesterday.Format(time.DateTime), yesterday.Format(time.DateTime)},
		"192.196.195.114": {yesterday.Format(time.DateTime), now.Format(time.DateTime)},
	}

	tokens := []string{}

	for range 20 {
		tokens = append(tokens, utils.RandomToken(63))
	}

	s.randomToken = strings.Join(tokens, "")

	// We have total 8 requests, so 8 referrers:
	referrers := map[string][]string{
		"86.8.6.85":       {"google.com", "google.com"},
		"188.167.251.87":  {"yahoo.com", "test.com"},
		"171.22.106.134":  {"yahoo.com", "google.com"},
		"192.196.195.114": {"example.org", fmt.Sprintf("test.com?code=%s", s.randomToken)},
		"181.130.251.87":  {"test.com"},
	}

	records := []analytics.Record{}

	for ip, visitTimes := range hash {
		for i, visitTime := range visitTimes {
			ts, err := time.Parse(time.DateTime, visitTime)
			s.NoError(err)

			unix := utils.NewUnix()
			unix.Time = ts

			records = append(records, analytics.Record{
				AppID:       app.ID,
				EnvID:       env.ID,
				VisitorIP:   ip,
				RequestTS:   unix,
				RequestPath: "/",
				StatusCode:  http.StatusOK,
				Referrer:    null.StringFrom(referrers[ip][i]),
				UserAgent:   null.StringFrom("my-\x00user\000-agent\u0000"),
				DomainID:    domain.ID,
			})
		}
	}

	ctx := context.Background()
	err := analytics.NewStore().InsertRecords(ctx, records)

	s.NoError(err)
	s.domainID = domain.ID
	s.yesterday = yesterday.Format(time.DateOnly)
	s.fiveDaysAgo = fiveDaysAgo.Format(time.DateOnly)
	s.today = now.Format(time.DateOnly)

	s.conn.Exec(`UPDATE analytics SET country_iso_code = 'US' WHERE referrer = 'google.com'`)
}

func (s *JobAnalyticsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *JobAnalyticsSuite) Test_SyncAnalyticsVisitorsDaily() {
	ctx := context.Background()

	// We create records in the BeforeTest statement. This test
	// will get all data from yesterday and sync it to the aggregate table.
	s.NoError(jobs.SyncAnalyticsVisitorsDaily(ctx))

	data, err := analytics.NewStore().Visitors(ctx, analytics.VisitorsArgs{
		StatusCode: http.StatusOK,
		Span:       analytics.SPAN_7D,
		DomainID:   s.domainID,
	})

	s.NoError(err)

	expected := map[string]any{}
	expected[s.yesterday] = map[string]int{
		"unique": 2,
		"total":  3,
	}

	s.Equal(expected, data)
}

func (s *JobAnalyticsSuite) Test_SyncAnalyticsVisitorsDaily_CustomDate() {
	// We test that we can pass a custom number of days to the job.
	ctx := context.WithValue(context.Background(), jobs.KeyContextNumberOfDays{}, 5)

	s.NoError(jobs.SyncAnalyticsVisitorsDaily(ctx))

	data, err := analytics.NewStore().Visitors(ctx, analytics.VisitorsArgs{
		StatusCode: http.StatusOK,
		Span:       analytics.SPAN_7D,
		DomainID:   s.domainID,
	})

	s.NoError(err)

	expected := map[string]any{}
	expected[s.yesterday] = map[string]int{
		"unique": 2,
		"total":  3,
	}

	expected[s.fiveDaysAgo] = map[string]int{
		"unique": 1,
		"total":  1,
	}

	s.Equal(expected, data)
}

func (s *JobAnalyticsSuite) Test_SyncAnalyticsVisitorsHourly() {
	// A manual resync over the last two days picks up yesterday's and today's
	// hours; the scheduled run's two-hour window is tested separately.
	ctx := context.WithValue(context.Background(), jobs.KeyContextNumberOfDays{}, 2)
	ctx = context.WithValue(ctx, jobs.KeyContextNow{}, s.at(16, 10))

	s.NoError(jobs.SyncAnalyticsVisitorsHourly(ctx))

	data, err := analytics.NewStore().Visitors(ctx, analytics.VisitorsArgs{
		StatusCode: http.StatusOK,
		Span:       analytics.SPAN_24h,
		DomainID:   s.domainID,
	})

	s.NoError(err)

	expected := map[string]any{}

	// We receive yesterday as well because the hourly view
	// fetches last 24 records sorted by date.
	expected[fmt.Sprintf("%s 15:00", s.yesterday)] = map[string]int{
		"unique": 2,
		"total":  3,
	}

	expected[fmt.Sprintf("%s 15:00", s.today)] = map[string]int{
		"unique": 2,
		"total":  2,
	}

	s.Equal(expected, data)
}

// Test_SyncAnalyticsVisitorsHourly_RecentHours verifies that the scheduled run
// only recomputes the current and the previous hour.
func (s *JobAnalyticsSuite) Test_SyncAnalyticsVisitorsHourly_RecentHours() {
	// The records at 15:30 today fall in the window of a run at 16:10, and
	// yesterday's do not.
	ctx := context.WithValue(context.Background(), jobs.KeyContextNow{}, s.at(16, 10))

	s.NoError(jobs.SyncAnalyticsVisitorsHourly(ctx))

	var buckets int

	row := s.conn.QueryRow(`SELECT COUNT(*) FROM analytics_visitors_agg_hourly_200 WHERE domain_id = $1`, s.domainID)
	s.Require().NoError(row.Scan(&buckets))
	s.Equal(1, buckets)
}

// Test_SyncAnalyticsRollups_RebuildsHours verifies that the hourly rollup run
// rebuilds every hour of its days, so hours whose page views were ingested
// after the five-minute job moved on are repaired.
func (s *JobAnalyticsSuite) Test_SyncAnalyticsRollups_RebuildsHours() {
	ctx := context.WithValue(context.Background(), jobs.KeyContextNow{}, s.at(23, 10))

	s.NoError(jobs.SyncAnalyticsRollups(ctx))

	rows, err := s.conn.Query(`SELECT TO_CHAR(aggregate_date, 'YYYY-MM-DD HH24:MI') FROM analytics_visitors_agg_hourly_200 WHERE domain_id = $1 ORDER BY 1`, s.domainID)
	s.Require().NoError(err)

	defer rows.Close()

	hours := []string{}

	for rows.Next() {
		var h string
		s.Require().NoError(rows.Scan(&h))
		hours = append(hours, h)
	}

	s.Equal([]string{s.yesterday + " 15:00", s.today + " 15:00"}, hours)
}

func (s *JobAnalyticsSuite) Test_SyncAnalyticsReferrers() {
	ctx := context.Background()

	// We create records in the BeforeTest statement. This test
	// will get all data from yesterday and sync it to the aggregate table.
	s.NoError(jobs.SyncAnalyticsRollups(ctx))

	data, err := analytics.NewStore().TopReferrers(ctx, analytics.TopReferrersArgs{
		DomainID: s.domainID,
	})

	s.NoError(err)

	expected := map[string]int{
		"google.com":  2,
		"yahoo.com":   1,
		"example.org": 1,
		fmt.Sprintf("test.com?code=%s", s.randomToken): 1,
	}

	s.Equal(expected, data)
}

func (s *JobAnalyticsSuite) Test_SyncAnalyticsByCountries() {
	ctx := context.Background()

	// We create records in the BeforeTest statement. This test
	// will get all data from yesterday and sync it to the aggregate table.
	s.NoError(jobs.SyncAnalyticsRollups(ctx))

	data, err := analytics.NewStore().ByCountries(ctx, analytics.ByCountriesArgs{
		DomainID: s.domainID,
	})

	s.NoError(err)

	expected := map[string]int{
		"US": 2,
	}

	s.Equal(expected, data)
}

// at returns a time today at hour:minute in the wall clock the records were
// stored with (BeforeTest stores them as UTC).
func (s *JobAnalyticsSuite) at(hour, minute int) time.Time {
	t, err := time.Parse(time.DateOnly, s.today)
	s.Require().NoError(err)

	return t.Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
}

func TestJobAnalyticsSuite(t *testing.T) {
	suite.Run(t, &JobAnalyticsSuite{})
}
