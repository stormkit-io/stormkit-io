package jobs_test

import (
	"context"
	"testing"
	"time"

	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type JobAccessLogRetentionSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *JobAccessLogRetentionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *JobAccessLogRetentionSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *JobAccessLogRetentionSuite) partitions() map[string]bool {
	names, err := jobs.NewStore().AccessLogPartitions(context.Background())
	s.NoError(err)

	set := map[string]bool{}

	for _, n := range names {
		set[n] = true
	}

	return set
}

func name(day time.Time) string {
	return "request_logs_" + day.UTC().Format("20060102")
}

func (s *JobAccessLogRetentionSuite) Test_DropsOldAndKeepsRecentPartitions() {
	ctx := context.Background()
	store := jobs.NewStore()
	today := time.Now().UTC()

	old := today.AddDate(0, 0, -60)
	recent := today.AddDate(0, 0, -10)

	s.NoError(store.CreateAccessLogPartition(ctx, old))
	s.NoError(store.CreateAccessLogPartition(ctx, recent))

	before := s.partitions()
	s.True(before[name(old)], "old partition should exist before maintenance")

	s.NoError(jobs.MaintainAccessLogPartitions(ctx))

	after := s.partitions()

	// Older than the 30-day window is dropped; within-window and the DEFAULT
	// safety-net partition are kept; create-ahead keeps today available.
	s.False(after[name(old)], "60-day-old partition should be dropped")
	s.True(after[name(recent)], "10-day-old partition should be kept")
	s.True(after[name(today)], "today's partition should exist")
	s.True(after["request_logs_default"], "default partition must never be dropped")
}

func (s *JobAccessLogRetentionSuite) Test_HonorsEnvRetentionWindow() {
	ctx := context.Background()
	store := jobs.NewStore()
	today := time.Now().UTC()

	day := today.AddDate(0, 0, -10)
	s.NoError(store.CreateAccessLogPartition(ctx, day))
	s.True(s.partitions()[name(day)])

	// A 5-day window makes the 10-day-old partition expire.
	s.T().Setenv("STORMKIT_ACCESS_LOG_RETENTION_DAYS", "5")
	s.NoError(jobs.MaintainAccessLogPartitions(ctx))

	s.False(s.partitions()[name(day)], "10-day-old partition should expire under a 5-day window")
}

func TestJobAccessLogRetentionSuite(t *testing.T) {
	suite.Run(t, &JobAccessLogRetentionSuite{})
}
