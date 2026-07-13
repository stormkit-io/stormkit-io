package databasetest

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/DATA-DOG/go-txdb"
	"github.com/joho/godotenv"

	"github.com/stripe/stripe-go"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/migrations"
)

// TestDB represents a test database.
type TestDB struct {
	*sql.DB
	Cfg database.DBConf
}

func init() {
	stripe.DefaultLeveledLogger = &stripe.LeveledLogger{
		Level: stripe.LevelError,
	}

	// Reset these because they modify the behaviour of all tests.
	// Default behaviour is github.
	os.Setenv("STORMKIT_DEPLOYER_SERVICE", "")
	os.Setenv("GITHUB_PRIVATE_KEY", "")

	_, b, _, _ := runtime.Caller(0)
	envFile := filepath.Join(filepath.Dir(b), "../../../../.env")

	if err := godotenv.Load(envFile); err != nil {
		panic(err)
	}

	cnf := config.New()
	cnf.Database = &config.DatabaseConfig{}
	cnf.Env = "test"
	cnf.Hash = "random-hash"
	cnf.AppSecret = "gS9u8RZ*3^7^3*jRfDdnTVv9@rrqqr#5"

	// Clean up schema_migrations
	var stmts = []string{
		"DROP TABLE IF EXISTS public.migrations;",
		fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE`, database.Config.Schema),
	}

	conn := database.NewConnection()

	for _, stmt := range stmts {
		if _, err := conn.Exec(stmt); err != nil {
			panic(err)
		}
	}

	migrations.CloudMigrations = true
	migrations.Up(conn, database.Config)

	txdb.Register("txdb", "postgres", database.ConnectionString(database.Config))
}

// PrepareOrPanic prepares the statement and returns it. It panics
// when an error is thrown.
func (db *TestDB) PrepareOrPanic(str string) *sql.Stmt {
	st, err := db.Prepare(str)

	if err != nil {
		panic(err)
	}

	return st
}

func (db *TestDB) CloseTx() {
	err := db.DB.Close()

	if err != nil {
		panic(err)
	}
}

// resetSequences resets all sequences in the test schema to 1 using a direct
// postgres connection so that the locks are held only for the duration of the
// statement. Running setval inside a txdb transaction would hold
// RowExclusiveLock on every sequence for the entire test, which conflicts with
// concurrent migration DDL (which needs AccessExclusiveLock) and can produce
// deadlocks when multiple test packages run in parallel.
//
// The query joins pg_namespace and restricts to current_schema() to avoid
// touching pg_catalog sequences or sequences from other schemas, and uses a
// schema-qualified format string for the regclass cast to prevent ambiguous
// name resolution via search_path.
func resetSequences() {
	direct, err := sql.Open("postgres", database.ConnectionString(database.Config))

	if err != nil {
		panic(err)
	}

	defer direct.Close()

	_, err = direct.Exec(`
		DO $$
		DECLARE
			r RECORD;
		BEGIN
			FOR r IN
				SELECT n.nspname, c.relname
				FROM pg_class c
				JOIN pg_namespace n ON n.oid = c.relnamespace
				WHERE c.relkind = 'S'
				  AND n.nspname = current_schema()
			LOOP
				PERFORM setval(format('%I.%I', r.nspname, r.relname)::regclass, 1, false);
			END LOOP;
		END $$;
	`)

	if err != nil {
		panic(err)
	}
}

func InitTx(suiteName string) TestDB {
	// Reset sequences before opening the txdb transaction so locks are released
	// immediately and do not interfere with concurrent migration DDL.
	resetSequences()

	conn, err := sql.Open("txdb", suiteName)

	if err != nil {
		panic(err)
	}

	db := TestDB{
		Cfg: database.Config,
		DB:  conn,
	}

	database.SetConnection(conn)

	// Set default URL
	admin.MustConfig().SetURL("http://stormkit:8888")

	return db
}
