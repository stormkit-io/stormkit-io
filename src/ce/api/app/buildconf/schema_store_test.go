package buildconf_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type SchemaStoreSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	schemaName string
}

func (s *SchemaStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.schemaName = "test_schema"
}

func (s *SchemaStoreSuite) AfterTest(_, _ string) {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	// Clean up schema if it still exists
	if err := store.DropSchema(ctx, s.schemaName); err != nil {
		// Don't panic on cleanup errors, just log them
		fmt.Printf("cleanup error (ignored): %v\n", err)
	}

	s.conn.CloseTx()
}

func (s *SchemaStoreSuite) Test_CreateSchema_Success() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	result, err := store.CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	// Verify credentials are returned
	s.NotEmpty(result.AppUserName, "app user name should be returned")
	s.NotEmpty(result.AppPassword, "app password should be returned")
	s.NotEmpty(result.MigrationUserName, "migration user name should be returned")
	s.NotEmpty(result.MigrationPassword, "migration password should be returned")

	// Verify names match expected pattern
	s.Equal(s.schemaName+"_user", result.AppUserName)
	s.Equal(s.schemaName+"_migration_user", result.MigrationUserName)

	// Verify passwords are 32 characters (from utils.RandomToken(32))
	s.Len(result.AppPassword, 32, "app password should be 32 characters")
	s.Len(result.MigrationPassword, 32, "migration password should be 32 characters")

	// Verify schema exists
	var exists bool
	row := s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "schema should exist")

	// Verify migration user was created
	row = s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, result.MigrationUserName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "migration user should exist")

	// Verify app user was created
	row = s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, result.AppUserName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "app user should exist")

	// Verify migration user has correct permissions
	row = s.conn.QueryRow(`SELECT has_schema_privilege($1, $2, 'USAGE')`, result.MigrationUserName, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "migration user should have USAGE on schema")

	row = s.conn.QueryRow(`SELECT has_schema_privilege($1, $2, 'CREATE')`, result.MigrationUserName, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "migration user should have CREATE on schema")

	// Verify app user has correct permissions
	row = s.conn.QueryRow(`SELECT has_schema_privilege($1, $2, 'USAGE')`, result.AppUserName, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "app user should have USAGE on schema")
}

func (s *SchemaStoreSuite) Test_CreateSchema_InvalidNames() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	invalidNames := []string{
		"123invalid",   // starts with number
		"invalid-name", // contains dash
		"invalid name", // contains space
		"invalid.name", // contains dot
		"",             // empty
	}

	for _, name := range invalidNames {
		result, err := store.CreateSchema(ctx, name)
		s.Error(err, "should reject invalid schema name: %s", name)
		s.Nil(result, "should not return credentials for invalid schema name: %s", name)
	}
}

func (s *SchemaStoreSuite) Test_CreateSchema_LengthLimit() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	longName := "a"

	for i := 0; i < 47; i++ {
		longName += "a"
	}

	result, err := store.CreateSchema(ctx, longName)
	s.Error(err, "should reject schema name longer than 47 characters")
	s.Nil(result, "should not return credentials for name exceeding length limit")
}

func (s *SchemaStoreSuite) Test_DropSchema_Success() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	// Create schema first
	result, err := store.CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	// Verify it exists
	var exists bool
	row := s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "schema should exist before dropping")

	// Drop schema
	err = store.DropSchema(ctx, s.schemaName)
	s.NoError(err)

	// Verify schema was dropped
	row = s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.False(exists, "schema should not exist after dropping")

	// Verify migration user was dropped
	row = s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, result.MigrationUserName)
	s.NoError(row.Scan(&exists))
	s.False(exists, "migration user should be dropped")

	// Verify app user was dropped
	row = s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, result.AppUserName)
	s.NoError(row.Scan(&exists))
	s.False(exists, "app user should be dropped")
}

func (s *SchemaStoreSuite) Test_DropSchema_NonExistent() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	// Should not error when dropping non-existent schema
	s.NoError(store.DropSchema(ctx, "non_existent_schema"), "should not error when dropping non-existent schema")
}

func (s *SchemaStoreSuite) Test_DropSchema_WithTables() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	// Create schema
	result, err := store.CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	// Create a table in the schema
	_, err = s.conn.Exec(`CREATE TABLE ` + s.schemaName + `.test_table (id SERIAL PRIMARY KEY, name TEXT)`)
	s.NoError(err)

	// Verify table exists
	var exists bool
	row := s.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = $1 AND table_name = 'test_table'
		)
	`, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.True(exists, "table should exist")

	// Drop schema (should cascade and drop table)
	err = store.DropSchema(ctx, s.schemaName)
	s.NoError(err)

	// Verify schema was dropped
	row = s.conn.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, s.schemaName)
	s.NoError(row.Scan(&exists))
	s.False(exists, "schema should be dropped along with tables")
}

func (s *SchemaStoreSuite) Test_DropSchema_InvalidNames() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	invalidNames := []string{
		"123invalid",
		"invalid-name",
	}

	for _, name := range invalidNames {
		err := store.DropSchema(ctx, name)
		s.Error(err, "should reject invalid schema name: %s", name)
	}
}

func (s *SchemaStoreSuite) Test_GetSchema_Success() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	// Create schema
	result, err := store.CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	// Create some tables
	_, err = s.conn.Exec(`CREATE TABLE ` + s.schemaName + `.users (id SERIAL PRIMARY KEY, name TEXT)`)
	s.NoError(err)
	_, err = s.conn.Exec(`CREATE TABLE ` + s.schemaName + `.posts (id SERIAL PRIMARY KEY, title TEXT)`)
	s.NoError(err)

	// Get schema info
	schema, err := store.GetSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(schema)
	s.Equal(s.schemaName, schema.Name)
	s.Len(schema.Tables, 2, "should return 2 tables")

	// Verify table names are present
	tableNames := make(map[string]bool)

	for _, table := range schema.Tables {
		tableNames[table.Name] = true
	}

	s.True(tableNames["users"], "users table should be in results")
	s.True(tableNames["posts"], "posts table should be in results")
}

func (s *SchemaStoreSuite) Test_GetSchema_NonExistent() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	schema, err := store.GetSchema(ctx, "non_existent_schema")
	s.NoError(err)
	s.Nil(schema, "should return nil for non-existent schema")
}

func (s *SchemaStoreSuite) Test_GetSchema_EmptySchema() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	// Create schema
	result, err := store.CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	// Get schema info
	schema, err := store.GetSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(schema)
	s.Equal(s.schemaName, schema.Name)
	s.Empty(schema.Tables, "should have no tables")
}

func (s *SchemaStoreSuite) Test_CreateSchema_ReturnsCredentials() {
	store := buildconf.SchemaStore()
	ctx := context.Background()

	result, err := store.CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	s.Equal(s.schemaName, result.SchemaName)
	s.Equal(database.Config.DBName, result.DBName)
	s.Equal(database.Config.Host, result.Host)
	s.Equal(database.Config.Port, result.Port)

	// Verify result contains all required fields
	s.NotEmpty(result.AppUserName)
	s.NotEmpty(result.AppPassword)
	s.NotEmpty(result.MigrationUserName)
	s.NotEmpty(result.MigrationPassword)

	// Verify credentials are distinct
	s.NotEqual(result.AppPassword, result.MigrationPassword, "passwords should be different")
	s.NotEqual(result.AppUserName, result.MigrationUserName, "usernames should be different")
}

func (s *SchemaStoreSuite) Test_Migrations_Success() {
	ctx := context.Background()

	// Create schema first
	result, err := buildconf.SchemaStore().CreateSchema(ctx, s.schemaName)
	s.NoError(err)
	s.NotNil(result)

	result.MigrationPassword = database.Config.Password
	result.MigrationUserName = database.Config.User
	result.DriverName = "txdb"

	// Create a store with schema configuration
	store, err := buildconf.SchemaStoreFor(result, buildconf.SchemaAccessTypeMigrations)
	s.NoError(err)
	s.NotNil(store)
	defer store.Close()

	// Ensure migrations table
	s.NoError(store.EnsureMigrationsTable())
	s.NoError(store.EnsureMigrationsTable(), "should be idempotent")

	// Initially should be empty
	migrations, err := store.Migrations(ctx)
	s.NoError(err)
	s.Empty(migrations, "should have no migrations initially")

	// Apply multiple migrations
	migrations1 := []struct {
		name    string
		content string
		hash    string
	}{
		{"001_init", "CREATE TABLE test1 (id INT);", "hash1"},
		{"002_add_users", "CREATE TABLE test2 (id INT);", "hash2"},
		{"003_add_posts", "CREATE TABLE test3 (id INT);", "hash3"},
	}

	for _, m := range migrations1 {
		result, err := store.ApplyMigration(ctx, buildconf.ApplyMigrationArgs{
			MigrationName: m.name,
			Content:       []byte(m.content),
			SHA:           m.hash,
			DeploymentID:  1,
		})

		s.NoError(err)
		s.NotNil(result)
		s.Empty(result.Error, "migration should apply without error")
	}

	// Retrieve migrations
	migrations, err = store.Migrations(ctx)
	s.NoError(err)
	s.Len(migrations, 3, "should have three recorded migrations")

	// Verify migration data
	s.Equal("001_init", migrations[0].Name)
	s.Equal("hash1", migrations[0].ContentHash)
	s.Equal("002_add_users", migrations[1].Name)
	s.Equal("hash2", migrations[1].ContentHash)
	s.Equal("003_add_posts", migrations[2].Name)
	s.Equal("hash3", migrations[2].ContentHash)
}

func (s *SchemaStoreSuite) Test_SchemaConf_String() {
	conf := &buildconf.SchemaConf{
		AppUserName:       "app_user",
		AppPassword:       "app_password",
		MigrationUserName: "migration_user",
		MigrationPassword: "migration_password",
		DBName:            "test_db",
		SchemaName:        "test_schema",
		Port:              "5432",
		Host:              "localhost",
	}

	expected := "host=localhost port=5432 dbname=test_db user=app_user password=app_password search_path=test_schema sslmode=disable"
	s.Equal(expected, conf.String())
}

func (s *SchemaStoreSuite) Test_SchemaConf_URL() {
	conf := &buildconf.SchemaConf{
		AppUserName:       "app_user",
		AppPassword:       "app_password",
		MigrationUserName: "migration_user",
		MigrationPassword: "migration_password",
		DBName:            "test_db",
		SchemaName:        "test_schema",
		Port:              "5432",
		Host:              "localhost",
		SSLMode:           "require",
	}

	expected := "postgresql://app_user:app_password@localhost:5432/test_db?search_path=test_schema&sslmode=require"
	s.Equal(expected, conf.URL())
}

func TestSchemaStore(t *testing.T) {
	suite.Run(t, &SchemaStoreSuite{})
}
