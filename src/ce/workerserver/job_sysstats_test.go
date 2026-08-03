package jobs_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/sysstats"
	"github.com/stretchr/testify/suite"
)

type JobSysStatsSuite struct {
	suite.Suite
	conn     databasetest.TestDB
	store    *sysstats.Store
	exporter *httptest.Server
	ctx      context.Context
}

func (s *JobSysStatsSuite) BeforeTest(suiteName, _ string) {
	s.ctx = context.Background()
	s.conn = databasetest.InitTx(suiteName)
	s.store = sysstats.NewStore(sysstats.NewStoreParams{})

	idle, busy := 1000.0, 200.0

	s.exporter = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idle += 75
		busy += 25

		fmt.Fprintf(w, `# TYPE node_cpu_seconds_total counter
node_cpu_seconds_total{cpu="0",mode="idle"} %f
node_cpu_seconds_total{cpu="0",mode="user"} %f
# TYPE node_memory_MemTotal_bytes gauge
node_memory_MemTotal_bytes 1.6e+10
# TYPE node_filesystem_size_bytes gauge
node_filesystem_size_bytes{device="/dev/sda1",fstype="ext4",mountpoint="/"} 5.0e+10
# TYPE node_filesystem_avail_bytes gauge
node_filesystem_avail_bytes{device="/dev/sda1",fstype="ext4",mountpoint="/"} 2.0e+10
`, idle, busy)
	}))
}

func (s *JobSysStatsSuite) AfterTest(_, _ string) {
	s.exporter.Close()
	_ = s.store.Drop(s.ctx, s.exporter.URL)
	s.conn.CloseTx()
}

// configureTarget points the job at the fake exporter through the manual target
// list, which is the path a machine with no Stormkit process takes.
func (s *JobSysStatsSuite) configureTarget(target string) {
	cfg := admin.MustConfig()
	cfg.MonitoringConfig = &admin.MonitoringConfig{Targets: []string{target}}

	s.Require().NoError(admin.Store().UpsertConfig(s.ctx, cfg))
}

func (s *JobSysStatsSuite) Test_CollectSystemStats_StoresSample() {
	s.configureTarget(s.exporter.URL)

	s.Require().NoError(jobs.CollectSystemStats(s.ctx))

	sample, err := s.store.Latest(s.ctx, s.exporter.URL)
	s.Require().NoError(err)
	s.Require().NotNil(sample)

	s.True(sample.Reachable)
	s.Equal(uint64(16000000000), sample.MemTotalBytes)
	s.Require().Len(sample.Filesystems, 1)
	s.Equal("/", sample.Filesystems[0].Mountpoint)
}

// The collector is shared across runs so the CPU counters survive, otherwise
// every scrape would look like a first scrape and never produce a reading.
func (s *JobSysStatsSuite) Test_CollectSystemStats_DerivesCPUAcrossRuns() {
	s.configureTarget(s.exporter.URL)

	s.Require().NoError(jobs.CollectSystemStats(s.ctx))
	s.Require().NoError(jobs.CollectSystemStats(s.ctx))

	sample, err := s.store.Latest(s.ctx, s.exporter.URL)
	s.Require().NoError(err)
	s.Require().NotNil(sample)

	s.True(sample.CPUValid, "the second run has a previous reading to compare against")
	s.InDelta(25, sample.CPUPercent, 0.001)
}

// A machine that is not answering must still be recorded, so the UI can report
// that node_exporter is not running there.
func (s *JobSysStatsSuite) Test_CollectSystemStats_RecordsUnreachableTarget() {
	target := "127.0.0.1:1"
	defer func() { _ = s.store.Drop(s.ctx, target) }()

	s.configureTarget(target)

	s.Require().NoError(jobs.CollectSystemStats(s.ctx))

	sample, err := s.store.Latest(s.ctx, target)
	s.Require().NoError(err)
	s.Require().NotNil(sample)
	s.False(sample.Reachable)
	s.NotEmpty(sample.Error)
}

func (s *JobSysStatsSuite) Test_CollectSystemStats_SnapshotsDependencies() {
	s.Require().NoError(jobs.CollectSystemStats(s.ctx))

	health, err := s.store.ReadDependencies(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(health)

	s.True(health.Postgres.Reachable)
	s.True(health.Redis.Reachable)
	s.NotZero(health.Timestamp)
	s.WithinDuration(time.Now(), time.Unix(health.Timestamp, 0), time.Minute)
}

// No configured targets is a normal state on a fresh install, not an error.
func (s *JobSysStatsSuite) Test_CollectSystemStats_NoTargets() {
	s.NoError(jobs.CollectSystemStats(s.ctx))
}

func TestJobSysStatsSuite(t *testing.T) {
	suite.Run(t, new(JobSysStatsSuite))
}
