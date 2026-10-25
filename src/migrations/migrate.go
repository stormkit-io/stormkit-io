package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

//go:embed *.up.sql
var migrations embed.FS

//go:embed seed_*.sql
var seed embed.FS

var CloudMigrations = config.IsStormkitCloud()

// Migrate applies the migration files newer than currentVersion and returns
// the latest applied version. The version is saved to public.migrations after
// each version's files succeed, so a failure later in the run does not make
// the next start re-run files that were already applied.
func Migrate(currentVersion int, db *sql.DB) (int, error) {
	m := &migrator{
		db:     db,
		files:  migrations,
		cloud:  CloudMigrations,
		runner: &migrationRunner{db: db, attempts: 5, delay: 2 * time.Second},
	}

	return m.migrate(currentVersion)
}

// migrationFS is the source of migration files.
type migrationFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type migrator struct {
	db     *sql.DB
	files  migrationFS
	cloud  bool
	runner *migrationRunner
}

type migrationFile struct {
	name    string
	version int
}

func (m *migrator) migrate(currentVersion int) (int, error) {
	pending, err := m.pending(currentVersion)

	if err != nil {
		log.Fatal("error while reading dir: " + err.Error())
	}

	latestVersion := currentVersion

	for i, file := range pending {
		content, err := m.files.ReadFile(file.name)

		if err != nil {
			return latestVersion, err
		}

		if err := m.runner.exec(execParams{name: file.name, content: string(content)}); err != nil {
			return latestVersion, err
		}

		latestVersion = file.version

		// Files sharing a version (e.g. .all and .cloud) are saved together.
		if i == len(pending)-1 || pending[i+1].version != file.version {
			if err := m.saveVersion(file.version); err != nil {
				return latestVersion, err
			}
		}
	}

	return latestVersion, nil
}

// pending returns the files newer than currentVersion, sorted by name.
func (m *migrator) pending(currentVersion int) ([]migrationFile, error) {
	entries, err := m.files.ReadDir(".")

	if err != nil {
		return nil, err
	}

	names := []string{}

	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	sort.Strings(names)

	files := []migrationFile{}

	for _, name := range names {
		version := utils.StringToInt(strings.Split(name, "_")[0])

		if version <= currentVersion {
			continue
		}

		// Some files are intended only for Stormkit Cloud
		if strings.Contains(name, ".cloud.") && !m.cloud {
			continue
		}

		files = append(files, migrationFile{name: name, version: version})
	}

	return files, nil
}

// saveVersion records the applied migration version, creating the row on a
// fresh database where seeds have not run yet.
func (m *migrator) saveVersion(version int) error {
	res, err := m.db.Exec("UPDATE public.migrations SET migration_version = $1", version)

	if err != nil {
		return err
	}

	if affected, err := res.RowsAffected(); err != nil || affected > 0 {
		return err
	}

	_, err = m.db.Exec(
		"INSERT INTO public.migrations (migration_version, seed_version, dirty) VALUES ($1, 0, false)",
		version,
	)

	return err
}

// migrationRunner executes migration files, retrying the ones that time out
// waiting for a lock. Each file runs as a single implicit transaction, so a
// failed attempt leaves nothing behind.
type migrationRunner struct {
	db       *sql.DB
	attempts int
	delay    time.Duration
}

type execParams struct {
	name    string
	content string
}

func (r *migrationRunner) exec(p execParams) error {
	delay := r.delay

	for attempt := 1; ; attempt++ {
		_, err := r.db.Exec(p.content)

		if err == nil || !database.IsLockNotAvailable(err) || attempt >= r.attempts {
			return err
		}

		slog.Infof("migration %s timed out waiting for a lock (attempt %d/%d), retrying in %s", p.name, attempt, r.attempts, delay)
		time.Sleep(delay)
		delay *= 2
	}
}

func Seed(currentVersion int, db *sql.DB) (int, error) {
	// No need to continue after this point for tests
	if config.IsTest() {
		return 0, nil
	}

	files, err := seed.ReadDir(".")

	if err != nil {
		log.Fatal(err.Error())
	}

	latestVersion := currentVersion
	fileNames := []string{}

	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	sort.Strings(fileNames)

	for _, name := range fileNames {
		version := utils.StringToInt(strings.Split(strings.Replace(name, "seed_", "", 1), "_")[0])

		if version <= currentVersion {
			continue
		}

		content, err := seed.ReadFile(name)

		if err != nil {
			return latestVersion, err
		}

		if _, err := db.Exec(string(content)); err != nil {
			return latestVersion, err
		}

		latestVersion = version
	}

	return latestVersion, nil
}

// Up migrates the database to the latest version specified in the *.sql files.
func Up(db *sql.DB, conf database.DBConf) bool {
	if !config.IsProduction() && config.Get().Database.WipeOnStart {
		slog.Info("Wipe on start is enabled, deleting all data in the database")

		db.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE;`, conf.Schema))
		db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", conf.Schema))

		slog.Info("Wiped database")
	}

	var migrationVersion int
	var seedVersion int
	var isDirty bool

	expo := 1
	store := database.NewStore()

	slog.Infof("acquiring lock for migrating database")

	// If the table already exists, another instance is running migrations
	for store.AdvisoryLock(0) != nil {
		slog.Infof("another instance is running migrations, waiting for %d second(s)...", expo)
		time.Sleep(time.Duration(expo) * time.Second)
		expo *= 2
	}

	defer store.AdvisoryUnlock(0)

	_ = db.
		QueryRow("SELECT migration_version, seed_version, dirty FROM public.migrations").
		Scan(&migrationVersion, &seedVersion, &isDirty)

	if isDirty {
		slog.Errorf("latest migration is marked as dirty, please migrate manually.")
		return false
	}

	mVer, err := Migrate(migrationVersion, db)

	if err != nil {
		log.Fatal("migration error: " + err.Error())
	}

	sVer, err := Seed(seedVersion, db)

	if err != nil {
		slog.Errorf("error while seeding: %s", err.Error())
	}

	if mVer == migrationVersion && sVer == seedVersion {
		slog.Infof("nothing to migrate")
		return true
	} else {
		slog.Infof("migrated to version=%d, seed=%d", mVer, sVer)
	}

	_, err = db.Exec("DELETE FROM public.migrations")

	if err != nil {
		slog.Errorf("error while deleting row: %s", err.Error())
	}

	_, err = db.Exec(
		"INSERT INTO public.migrations (migration_version, seed_version, dirty) VALUES ($1, $2, $3)",
		mVer, sVer, err != nil,
	)

	if err != nil {
		log.Fatal("update migrations error: " + err.Error())
	}

	return true
}
