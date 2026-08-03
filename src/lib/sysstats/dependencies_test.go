package sysstats

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type DependenciesSuite struct {
	suite.Suite
	ctx context.Context
	db  databasetest.TestDB
}

func (s *DependenciesSuite) SetupTest() {
	s.ctx = context.Background()
	s.db = databasetest.InitTx("sysstats_dependencies")
}

func (s *DependenciesSuite) TearDownTest() {
	s.db.CloseTx()
}

func (s *DependenciesSuite) Test_Postgres() {
	health := CollectDependencies(s.ctx, CollectDependenciesParams{
		DB:    database.Connection(),
		Cache: rediscache.Client(),
	})

	s.Require().True(health.Postgres.Reachable)
	s.Empty(health.Postgres.Error)
	s.Positive(health.Postgres.DatabaseBytes)
	s.Positive(health.Postgres.Connections)
	s.Positive(health.Postgres.MaxConnections)
	s.NotEmpty(health.Postgres.LargestTables, "the schema has tables to report")

	for _, table := range health.Postgres.LargestTables {
		s.NotEmpty(table.Name)
	}
}

func (s *DependenciesSuite) Test_Redis() {
	health := CollectDependencies(s.ctx, CollectDependenciesParams{
		DB:    database.Connection(),
		Cache: rediscache.Client(),
	})

	s.True(health.Redis.Reachable)
	s.Empty(health.Redis.Error)
	s.Positive(health.Redis.UsedMemoryBytes)
}

// A missing dependency is reported, not panicked over — this is the path taken
// when Postgres is down, which is exactly when the panel matters.
func (s *DependenciesSuite) Test_ReportsMissingConnections() {
	health := CollectDependencies(s.ctx, CollectDependenciesParams{})

	s.False(health.Postgres.Reachable)
	s.NotEmpty(health.Postgres.Error)
	s.False(health.Redis.Reachable)
	s.NotEmpty(health.Redis.Error)
	s.NotZero(health.Timestamp)
}

func (s *DependenciesSuite) Test_SaveAndReadDependencies() {
	store := NewStore(NewStoreParams{})

	health := CollectDependencies(s.ctx, CollectDependenciesParams{
		DB:    database.Connection(),
		Cache: rediscache.Client(),
	})

	s.Require().NoError(store.SaveDependencies(s.ctx, health))

	read, err := store.ReadDependencies(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(read)
	s.True(read.Postgres.Reachable)
	s.Equal(health.Postgres.DatabaseBytes, read.Postgres.DatabaseBytes)
	s.Len(read.Postgres.LargestTables, len(health.Postgres.LargestTables))
}

func (s *DependenciesSuite) Test_NewPoolStats() {
	stats := NewPoolStats(database.Connection())

	// MaxOpenConnections is zero when the pool is unlimited, so only the
	// counters are asserted here.
	s.GreaterOrEqual(stats.Open, 0)
	s.GreaterOrEqual(stats.InUse, 0)
	s.GreaterOrEqual(stats.WaitCount, int64(0))

	s.Equal(PoolStats{}, NewPoolStats(nil), "a nil pool is zero, not a panic")
}

func (s *DependenciesSuite) Test_ParseInfoUint() {
	info := "# Memory\r\nused_memory:1234567\r\nused_memory_human:1.18M\r\n"

	s.Equal(uint64(1234567), parseInfoUint(info, "used_memory"))
	s.Equal(uint64(0), parseInfoUint(info, "used_memory_human"), "non-numeric fields yield zero")
	s.Equal(uint64(0), parseInfoUint(info, "missing_field"))
}

func TestDependenciesSuite(t *testing.T) {
	suite.Run(t, new(DependenciesSuite))
}
