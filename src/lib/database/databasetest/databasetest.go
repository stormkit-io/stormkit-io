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

func InitTx(suiteName string) TestDB {
	conn, err := sql.Open("txdb", suiteName)

	if err != nil {
		panic(err)
	}

	// Reset all sequences to 1
	_, err = conn.Exec(`
		DO $$
		DECLARE
    		r RECORD;
		BEGIN
		FOR r IN
        		SELECT c.relname 
        		FROM pg_class c 
        		WHERE c.relkind = 'S'
    		LOOP
        		EXECUTE format('ALTER SEQUENCE %I RESTART WITH 1', r.relname);
    		END LOOP;
		END $$;
	`)

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
