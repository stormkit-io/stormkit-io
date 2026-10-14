package jobs_test

import (
	"context"
	"testing"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type JobTriggerLogsRetentionSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *JobTriggerLogsRetentionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	app := s.MockApp(nil)
	env := s.MockEnv(app)
	trigger := s.MockTriggerFunction(env)

	// Two rows older than the 30-day default, two within the window.
	ages := []string{"45 days", "31 days", "10 days", "1 day"}

	for _, age := range ages {
		_, err := s.conn.Exec(`
			INSERT INTO skitapi.function_trigger_logs
				(trigger_id, request, response, created_at)
			VALUES
				($1, '{}'::jsonb, '{}'::jsonb, timezone('utc', now()) - $2::interval);
		`, trigger.ID, age)

		s.NoError(err)
	}
}

func (s *JobTriggerLogsRetentionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *JobTriggerLogsRetentionSuite) countLogs() int {
	var count int
	s.NoError(s.conn.QueryRow(`SELECT count(*) FROM skitapi.function_trigger_logs`).Scan(&count))
	return count
}

func (s *JobTriggerLogsRetentionSuite) Test_RemoveOldTriggerLogs_DeletesOnlyOldRows() {
	s.Equal(4, s.countLogs())

	deleted, err := jobs.NewStore().RemoveOldTriggerLogs(context.Background(), jobs.RemoveOldRowsParams{
		RetentionDays: 30,
		BatchSize:     10000,
	})

	s.NoError(err)
	s.Equal(int64(2), deleted)
	s.Equal(2, s.countLogs())
}

func (s *JobTriggerLogsRetentionSuite) Test_RemoveOldTriggerLogs_NoOpWhenNothingExpired() {
	deleted, err := jobs.NewStore().RemoveOldTriggerLogs(context.Background(), jobs.RemoveOldRowsParams{
		RetentionDays: 365,
		BatchSize:     10000,
	})

	s.NoError(err)
	s.Equal(int64(0), deleted)
	s.Equal(4, s.countLogs())
}

func (s *JobTriggerLogsRetentionSuite) Test_RemoveOldLogs_JobHonorsEnvWindow() {
	s.T().Setenv("STORMKIT_TRIGGER_LOGS_RETENTION_DAYS", "5")

	s.NoError(jobs.RemoveOldLogs(context.Background()))

	// Only the 1-day-old row survives a 5-day window.
	s.Equal(1, s.countLogs())
}

func TestJobTriggerLogsRetentionSuite(t *testing.T) {
	suite.Run(t, &JobTriggerLogsRetentionSuite{})
}
