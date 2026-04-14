package publicapiv1_test

import (
	"database/sql"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
)

// truncateAuthTables removes all rows from the auth tables that are written through
// a direct postgres connection (bypassing txdb). It is a no-op when the tables do
// not yet exist.
func truncateAuthTables() {
	db, err := sql.Open("postgres", database.ConnectionString(database.Config))

	if err != nil {
		return
	}

	defer db.Close()

	if _, err := db.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT FROM pg_tables WHERE schemaname = current_schema() AND tablename = 'stormkit_auth_users') THEN
				TRUNCATE stormkit_auth_users, stormkit_auth_providers RESTART IDENTITY CASCADE;
			END IF;
		END $$;
	`); err != nil {
		panic("truncateAuthTables: " + err.Error())
	}
}
