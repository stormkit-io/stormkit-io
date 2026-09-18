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

type JobAnalyticsRetentionSuite struct {
	suite.Suite
	*factory.Factory

	conn     databasetest.TestDB
	domainID types.ID
}

func (s *JobAnalyticsRetentionSuite) BeforeTest(suiteName, _ string) {
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

	now := time.Now()
	ages := []time.Duration{
		200 * 24 * time.Hour, // older than the 180-day default
		190 * 24 * time.Hour, // older than the 180-day default
		10 * 24 * time.Hour,  // within retention
		1 * 24 * time.Hour,   // within retention
	}

	records := []analytics.Record{}

	for _, age := range ages {
		unix := utils.NewUnix()
		unix.Time = now.Add(-age)

		records = append(records, analytics.Record{
			AppID:       app.ID,
			EnvID:       env.ID,
			VisitorIP:   "86.8.6.85",
			RequestTS:   unix,
			RequestPath: "/",
			StatusCode:  http.StatusOK,
			DomainID:    domain.ID,
		})
	}

	s.NoError(analytics.NewStore().InsertRecords(context.Background(), records))
	s.domainID = domain.ID
}

func (s *JobAnalyticsRetentionSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *JobAnalyticsRetentionSuite) countAnalytics() int {
	var count int
	s.NoError(s.conn.QueryRow(`SELECT count(*) FROM analytics`).Scan(&count))
	return count
}

func (s *JobAnalyticsRetentionSuite) Test_RemoveOldAnalytics_DeletesOnlyOldRows() {
	s.Equal(4, s.countAnalytics())

	deleted, err := jobs.NewStore().RemoveOldAnalytics(context.Background(), jobs.RemoveOldRowsParams{
		RetentionDays: 180,
		BatchSize:     10000,
	})

	s.NoError(err)
	s.Equal(int64(2), deleted)
	s.Equal(2, s.countAnalytics())
}

func (s *JobAnalyticsRetentionSuite) Test_RemoveOldAnalytics_NoOpWhenNothingExpired() {
	deleted, err := jobs.NewStore().RemoveOldAnalytics(context.Background(), jobs.RemoveOldRowsParams{
		RetentionDays: 365,
		BatchSize:     10000,
	})

	s.NoError(err)
	s.Equal(int64(0), deleted)
	s.Equal(4, s.countAnalytics())
}

func (s *JobAnalyticsRetentionSuite) Test_RemoveOldAnalytics_JobHonorsEnvWindow() {
	s.T().Setenv("STORMKIT_ANALYTICS_RETENTION_DAYS", "5")

	s.NoError(jobs.RemoveOldAnalytics(context.Background()))

	// Only the 1-day-old row survives a 5-day window.
	s.Equal(1, s.countAnalytics())
}

func TestJobAnalyticsRetentionSuite(t *testing.T) {
	suite.Run(t, &JobAnalyticsRetentionSuite{})
}
