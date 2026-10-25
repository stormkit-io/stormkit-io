package migrations_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	_ "github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/migrations"
)

const lockProbeTable = "migrate_lock_probe"

type MigrateSuite struct {
	suite.Suite
	db *sql.DB
}

func (s *MigrateSuite) SetupSuite() {
	db, err := sql.Open("postgres", database.ConnectionString(database.Config))
	s.Require().NoError(err)

	s.db = db
}

func (s *MigrateSuite) TearDownSuite() {
	s.db.Close()
}

func (s *MigrateSuite) SetupTest() {
	_, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s; CREATE TABLE %s (id bigint);", lockProbeTable, lockProbeTable))
	s.Require().NoError(err)
}

func (s *MigrateSuite) TearDownTest() {
	_, err := s.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", lockProbeTable))
	s.Require().NoError(err)
}

// holdLock locks the probe table from another connection until the returned
// transaction ends.
func (s *MigrateSuite) holdLock() *sql.Tx {
	tx, err := s.db.Begin()
	s.Require().NoError(err)

	_, err = tx.Exec(fmt.Sprintf("LOCK TABLE %s IN ACCESS SHARE MODE", lockProbeTable))
	s.Require().NoError(err)

	return tx
}

func (s *MigrateSuite) alterProbe(attempts int) error {
	return migrations.ExecMigration(migrations.ExecMigrationParams{
		DB:       s.db,
		Content:  fmt.Sprintf("SET LOCAL lock_timeout = '50ms'; ALTER TABLE %s ADD COLUMN IF NOT EXISTS probe text;", lockProbeTable),
		Attempts: attempts,
		Delay:    100 * time.Millisecond,
	})
}

// Test_Exec_RetriesOnLockTimeout verifies that a migration blocked by a reader
// succeeds once the reader lets go.
func (s *MigrateSuite) Test_Exec_RetriesOnLockTimeout() {
	tx := s.holdLock()

	go func() {
		time.Sleep(250 * time.Millisecond)
		tx.Rollback()
	}()

	s.NoError(s.alterProbe(5))

	var exists bool

	err := s.db.QueryRow(
		"SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = 'probe')",
		lockProbeTable,
	).Scan(&exists)

	s.Require().NoError(err)
	s.True(exists)
}

func (s *MigrateSuite) Test_Exec_GivesUpAfterAttempts() {
	tx := s.holdLock()
	defer tx.Rollback()

	err := s.alterProbe(2)

	s.Error(err)
	s.True(database.IsLockNotAvailable(err))
}

func (s *MigrateSuite) Test_Exec_DoesNotRetryOtherErrors() {
	start := time.Now()

	err := migrations.ExecMigration(migrations.ExecMigrationParams{
		DB:       s.db,
		Content:  "SELECT * FROM table_that_does_not_exist;",
		Attempts: 5,
		Delay:    time.Second,
	})

	s.Error(err)
	s.False(database.IsLockNotAvailable(err))
	s.Less(time.Since(start), time.Second)
}

// Test_Migrate_SavesProgressOnFailure verifies that versions applied before a
// failing file are recorded, so the next start does not re-run them.
func (s *MigrateSuite) Test_Migrate_SavesProgressOnFailure() {
	var migrationVersion, seedVersion int
	var dirty bool

	err := s.db.QueryRow("SELECT migration_version, seed_version, dirty FROM public.migrations").
		Scan(&migrationVersion, &seedVersion, &dirty)

	s.Require().NoError(err)

	defer func() {
		_, err := s.db.Exec(
			"UPDATE public.migrations SET migration_version = $1, seed_version = $2, dirty = $3",
			migrationVersion, seedVersion, dirty,
		)

		s.Require().NoError(err)
	}()

	version, err := migrations.MigrateFiles(migrations.MigrateFilesParams{
		DB: s.db,
		Files: fstest.MapFS{
			"1001_a.up.sql": {Data: []byte(fmt.Sprintf("ALTER TABLE %s ADD COLUMN applied text;", lockProbeTable))},
			"1002_b.up.sql": {Data: []byte("SELECT * FROM table_that_does_not_exist;")},
		},
		CurrentVersion: 1000,
	})

	s.Error(err)
	s.Equal(1001, version)

	var saved int

	s.Require().NoError(s.db.QueryRow("SELECT migration_version FROM public.migrations").Scan(&saved))
	s.Equal(1001, saved)
}

// Test_DateFirstAggregates_UseDomainIndex verifies that per-domain range reads on
// the date-first aggregate tables use the domain-first index.
func (s *MigrateSuite) Test_DateFirstAggregates_UseDomainIndex() {
	tables := []string{
		"analytics_visitors_agg_200",
		"analytics_visitors_agg_404",
		"analytics_visitors_agg_hourly_200",
		"analytics_visitors_agg_hourly_404",
		"analytics_visitors_by_countries",
		"analytics_events_agg",
	}

	for _, table := range tables {
		tx, err := s.db.Begin()
		s.Require().NoError(err)

		// The tables are empty, so without this the planner prefers a seq scan.
		_, err = tx.Exec("SET LOCAL enable_seqscan = off")
		s.Require().NoError(err)

		rows, err := tx.Query(fmt.Sprintf(
			"EXPLAIN SELECT * FROM %s WHERE domain_id = ANY('{1,2}'::bigint[]) AND aggregate_date BETWEEN '2026-01-01' AND '2026-01-31'",
			table,
		))

		s.Require().NoError(err)

		plan := []string{}

		for rows.Next() {
			var line string
			s.Require().NoError(rows.Scan(&line))
			plan = append(plan, line)
		}

		rows.Close()
		tx.Rollback()

		s.Contains(strings.Join(plan, "\n"), fmt.Sprintf("idx_%s_domain_date", table), table)
	}
}

func TestMigrateSuite(t *testing.T) {
	suite.Run(t, new(MigrateSuite))
}
