package migrations

import (
	"database/sql"
	"embed"
	"fmt"
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

func Migrate(currentVersion int, db *sql.DB) (int, error) {
	files, err := migrations.ReadDir(".")

	if err != nil {
		log.Fatal("error while reading dir: " + err.Error())
	}

	latestVersion := currentVersion
	fileNames := []string{}

	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	sort.Strings(fileNames)

	for _, name := range fileNames {
		version := utils.StringToInt(strings.Split(name, "_")[0])

		if version <= currentVersion {
			continue
		}

		// Some files are intended only for Stormkit Cloud
		if strings.Contains(name, ".cloud.") && !CloudMigrations {
			continue
		}

		content, err := migrations.ReadFile(name)

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
