package migrations

import (
	"database/sql"
	"testing/fstest"
	"time"
)

// ExecMigrationParams configures ExecMigration.
type ExecMigrationParams struct {
	DB       *sql.DB
	Content  string
	Attempts int
	Delay    time.Duration
}

// ExecMigration exposes the migration runner to the external test package.
func ExecMigration(p ExecMigrationParams) error {
	runner := &migrationRunner{db: p.DB, attempts: p.Attempts, delay: p.Delay}
	return runner.exec(execParams{name: "test", content: p.Content})
}

// MigrateFilesParams configures MigrateFiles.
type MigrateFilesParams struct {
	DB             *sql.DB
	Files          fstest.MapFS
	CurrentVersion int
}

// MigrateFiles runs the migrator against the given files instead of the
// embedded ones.
func MigrateFiles(p MigrateFilesParams) (int, error) {
	m := &migrator{
		db:     p.DB,
		files:  p.Files,
		runner: &migrationRunner{db: p.DB, attempts: 1},
	}

	return m.migrate(p.CurrentVersion)
}
