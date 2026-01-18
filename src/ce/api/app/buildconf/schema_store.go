package buildconf

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"text/template"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

var schemaStmt = struct {
	selectSchema          string
	selectTables          string
	createAuthTable       string
	createMigrationsTable string
	selectMigrations      string
	selectAuthUser        string
	insertOAuth           string
	insertAuthUser        string
}{
	createAuthTable: `
		CREATE TABLE IF NOT EXISTS stormkit_auth_users (
			user_id SERIAL PRIMARY KEY NOT NULL,
			email TEXT NOT NULL UNIQUE,
			first_name TEXT,
			last_name TEXT,
			avatar TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_login_at TIMESTAMPTZ
		);

		CREATE TABLE IF NOT EXISTS stormkit_auth_providers (
			auth_id SERIAL PRIMARY KEY NOT NULL,
			user_id BIGINT NOT NULL REFERENCES stormkit_auth_users(user_id) ON DELETE CASCADE,
			account_id TEXT,
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			token_type TEXT NOT NULL,
			provider_name TEXT NOT NULL,
			expiry TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, provider_name)
		);
	`,

	createMigrationsTable: `
		CREATE TABLE IF NOT EXISTS stormkit_schema_migrations (
			migration_id SERIAL PRIMARY KEY,
			migration_name TEXT NOT NULL UNIQUE,
			migration_duration_ms BIGINT NOT NULL DEFAULT 0,
			deployment_id BIGINT NOT NULL,
			content_hash TEXT NOT NULL,
			error_message TEXT,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`,

	selectMigrations: `
		SELECT
			migration_id, migration_name, content_hash, error_message
		FROM
			stormkit_schema_migrations
		ORDER BY
			migration_id ASC;
	`,

	selectSchema: `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.schemata where schema_name = $1
		);
	`,

	selectTables: `
		SELECT
			t.table_name,
			pg_relation_size(quote_ident(t.table_schema)||'.'||quote_ident(t.table_name)) AS size_bytes,
			coalesce(s.n_live_tup, 0) AS estimated_rows
		FROM information_schema.tables t
		LEFT JOIN pg_stat_user_tables s
			ON s.schemaname = t.table_schema
			AND s.relname = t.table_name
		WHERE
			t.table_schema = $1
			AND t.table_type = 'BASE TABLE'
		ORDER BY 3;
	`,

	selectAuthUser: `
		SELECT
			user_id,
			first_name,
			last_name,
			email,
			avatar,
			created_at,
			last_login_at
		FROM
			stormkit_auth_users
		WHERE
			user_id = $1;
	`,

	insertOAuth: `
		INSERT INTO stormkit_auth_providers (
			user_id, account_id, access_token, refresh_token, token_type, provider_name, expiry
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		);
	`,

	insertAuthUser: `
		INSERT INTO stormkit_auth_users (
			email, first_name, last_name, avatar
		) VALUES (
			$1, $2, $3, $4
		) RETURNING
			user_id;
	`,
}

var sqlTemplates = struct {
	createSchema        *template.Template
	createMigrationUser *template.Template
	createAppUser       *template.Template
	dropSchema          *template.Template
}{
	createSchema: template.Must(template.New("createSchema").Parse(`CREATE SCHEMA IF NOT EXISTS {{.SchemaName}}`)),

	createMigrationUser: template.Must(template.New("createMigrationUser").Parse(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '{{.MigrationUserName}}') THEN
				CREATE ROLE "{{.MigrationUserName}}" WITH LOGIN PASSWORD '{{.MigrationUserPassword}}';

				-- Set resource limits for safety
				ALTER ROLE "{{.MigrationUserName}}" SET statement_timeout = '30s';
				ALTER ROLE "{{.MigrationUserName}}" SET lock_timeout = '10s';
				ALTER ROLE "{{.MigrationUserName}}" SET idle_in_transaction_session_timeout = '60s';
				ALTER ROLE "{{.MigrationUserName}}" SET temp_file_limit = '100MB';
				ALTER ROLE "{{.MigrationUserName}}" SET work_mem = '4MB';
				ALTER ROLE "{{.MigrationUserName}}" CONNECTION LIMIT 1;

				-- Prevent superuser privileges
				ALTER ROLE "{{.MigrationUserName}}" WITH NOSUPERUSER NOCREATEDB NOCREATEROLE;

				-- Revoke public schema access
				REVOKE ALL ON SCHEMA public FROM "{{.MigrationUserName}}";
				REVOKE ALL ON DATABASE postgres FROM "{{.MigrationUserName}}";

				-- Grant DDL permissions (schema changes only)
				GRANT USAGE ON SCHEMA "{{.SchemaName}}" TO "{{.MigrationUserName}}";
				GRANT CREATE ON SCHEMA "{{.SchemaName}}" TO "{{.MigrationUserName}}";
				GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA "{{.SchemaName}}" TO "{{.MigrationUserName}}";
				ALTER DEFAULT PRIVILEGES IN SCHEMA "{{.SchemaName}}" GRANT USAGE, SELECT ON SEQUENCES TO "{{.MigrationUserName}}";
			END IF;
		END
		$$;
	`)),

	createAppUser: template.Must(template.New("createAppUser").Parse(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '{{.AppUserName}}') THEN
				CREATE ROLE "{{.AppUserName}}" WITH LOGIN PASSWORD '{{.AppUserPassword}}';

				-- Prevent superuser privileges
				ALTER ROLE "{{.AppUserName}}" WITH NOSUPERUSER NOCREATEDB NOCREATEROLE;
				ALTER ROLE "{{.AppUserName}}" SET statement_timeout = '15s';
				ALTER ROLE "{{.AppUserName}}" SET lock_timeout = '5s';
				ALTER ROLE "{{.AppUserName}}" SET idle_in_transaction_session_timeout = '60s';
				ALTER ROLE "{{.AppUserName}}" SET temp_file_limit = '100MB';
				ALTER ROLE "{{.AppUserName}}" SET work_mem = '8MB';
				ALTER ROLE "{{.AppUserName}}" CONNECTION LIMIT 10;

				-- Revoke public schema access
				REVOKE ALL ON SCHEMA public FROM "{{.AppUserName}}";
				REVOKE ALL ON DATABASE postgres FROM "{{.AppUserName}}";

				-- Grant DML permissions (data operations only)
				GRANT USAGE ON SCHEMA "{{.SchemaName}}" TO "{{.AppUserName}}";
				GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA "{{.SchemaName}}" TO "{{.AppUserName}}";
				GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA "{{.SchemaName}}" TO "{{.AppUserName}}";

				-- Grant permissions on future objects created by migration user
				ALTER DEFAULT PRIVILEGES FOR ROLE "{{.MigrationUserName}}" IN SCHEMA "{{.SchemaName}}" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "{{.AppUserName}}";
				ALTER DEFAULT PRIVILEGES FOR ROLE "{{.MigrationUserName}}" IN SCHEMA "{{.SchemaName}}" GRANT USAGE, SELECT ON SEQUENCES TO "{{.AppUserName}}";
			END IF;
		END
		$$;
	`)),

	dropSchema: template.Must(template.New("dropSchema").Parse(`
		DO $$
		DECLARE
			db_name TEXT := current_database();
		BEGIN
			-- Terminate any active connections using the schema
			PERFORM pg_terminate_backend(pid)
			FROM pg_stat_activity
			WHERE datname = db_name
			  AND (usename = '{{.MigrationUserName}}' OR usename = '{{.AppUserName}}');

			-- Drop schema with cascade (drops all contained objects)
			DROP SCHEMA IF EXISTS "{{.SchemaName}}" CASCADE;

			-- Revoke all role memberships and drop roles
			BEGIN
				EXECUTE format('REVOKE ALL ON DATABASE %I FROM %I', db_name, '{{.MigrationUserName}}');
			EXCEPTION WHEN OTHERS THEN
				NULL; -- Ignore if user has no grants
			END;

			BEGIN
				EXECUTE format('REVOKE ALL ON DATABASE %I FROM %I', db_name, '{{.AppUserName}}');
			EXCEPTION WHEN OTHERS THEN
				NULL; -- Ignore if user has no grants
			END;

			-- Drop the users/roles (only if they exist)
			DROP ROLE IF EXISTS "{{.MigrationUserName}}";
			DROP ROLE IF EXISTS "{{.AppUserName}}";
		END
		$$;
	`)),
}

type schemaStore struct {
	*database.Store

	conf       *SchemaConf
	accessType string
}

// SchemaStore returns a store instance.
func SchemaStore() *schemaStore {
	return &schemaStore{
		Store: database.NewStore(),
	}
}

// SchemaStoreFor returns a schema store for the given configuration and credentials.
func SchemaStoreFor(conf *SchemaConf, accessType string) (*schemaStore, error) {
	var username, password string

	switch accessType {
	case SchemaAccessTypeMigrations:
		username = conf.MigrationUserName
		password = conf.MigrationPassword
	case SchemaAccessTypeAppUser:
		username = conf.AppUserName
		password = conf.AppPassword
	default:
		return nil, fmt.Errorf("unknown schema access type: %s", accessType)
	}

	conn, err := database.NewConnectionWithConfig(database.DBConf{
		Host:         conf.Host,
		Port:         conf.Port,
		User:         username,
		Password:     password,
		DBName:       conf.DBName,
		Schema:       conf.SchemaName,
		SSLMode:      conf.SSLMode,
		DriverName:   conf.DriverName,
		MaxLifetime:  database.Config.MaxLifetime,
		MaxOpenConns: database.Config.MaxOpenConns,
		MaxIdleConns: database.Config.MaxIdleConns,
	})

	if err != nil {
		return nil, err
	}

	return &schemaStore{
		conf:       conf,
		accessType: accessType,
		Store: &database.Store{
			Conn: conn,
		},
	}, nil
}

// GetSchema retrieves schema information from the database.
func (s *schemaStore) GetSchema(ctx context.Context, schemaName string) (*Schema, error) {
	var exists bool

	row, err := s.QueryRow(ctx, schemaStmt.selectSchema, schemaName)

	if err != nil {
		return nil, err
	}

	if err := row.Scan(&exists); err != sql.ErrNoRows && err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	rows, err := s.Query(ctx, schemaStmt.selectTables, schemaName)

	if err != nil {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	schema := &Schema{
		Name:   schemaName,
		Tables: []SchemaTable{},
	}

	for rows.Next() {
		table := SchemaTable{}

		if err := rows.Scan(&table.Name, &table.Size, &table.Rows); err != nil {
			return nil, err
		}

		schema.Tables = append(schema.Tables, table)
	}

	return schema, nil
}

// CreateSchema creates a new schema in the database if it doesn't exist.
// It also creates schema-specific users and grants permissions.
func (s *schemaStore) CreateSchema(ctx context.Context, schemaName string) (*SchemaConf, error) {
	schema, err := s.GetSchema(ctx, schemaName)

	if err != nil {
		return nil, err
	}

	if schema != nil {
		return nil, ErrSchemaExists
	}

	// Validate schema name to prevent SQL injection
	if !isSQLSafe(schemaName) {
		return nil, ErrInvalidSchemaName
	}

	// Create schema
	buf := bytes.Buffer{}

	err = sqlTemplates.createSchema.Execute(&buf, map[string]string{
		"SchemaName": schemaName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to render create schema template: %w", err)
	}

	if _, err := s.Exec(ctx, buf.String()); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	buf.Reset()

	// Create migration user (with DDL permissions)
	migrationUserName := fmt.Sprintf("%s_migration_user", schemaName)
	migrationPassword := utils.RandomToken(32)

	err = sqlTemplates.createMigrationUser.Execute(&buf, map[string]string{
		"SchemaName":            schemaName,
		"MigrationUserName":     migrationUserName,
		"MigrationUserPassword": migrationPassword,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to render migration user template: %w", err)
	}

	if _, err := s.Exec(ctx, buf.String()); err != nil {
		return nil, fmt.Errorf("failed to create migration user for schema: %w", err)
	}

	buf.Reset()

	// Create app user (with DML permissions)
	appUserName := fmt.Sprintf("%s_user", schemaName)
	appPassword := utils.RandomToken(32)

	err = sqlTemplates.createAppUser.Execute(&buf, map[string]string{
		"SchemaName":        schemaName,
		"AppUserName":       appUserName,
		"AppUserPassword":   appPassword,
		"MigrationUserName": migrationUserName,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to render app user template: %w", err)
	}

	if _, err := s.Exec(ctx, buf.String()); err != nil {
		return nil, fmt.Errorf("failed to create app user for schema: %w", err)
	}

	return &SchemaConf{
		AppUserName:       appUserName,
		AppPassword:       appPassword,
		MigrationUserName: migrationUserName,
		MigrationPassword: migrationPassword,
		SchemaName:        schemaName,
		DBName:            database.Config.DBName,
		Port:              database.Config.Port,
		Host:              database.Config.Host,
	}, nil
}

// DropSchema removes a schema and its associated users.
func (s *schemaStore) DropSchema(ctx context.Context, schemaName string) error {
	// Validate schema name to prevent SQL injection
	if !isSQLSafe(schemaName) {
		return fmt.Errorf("invalid schema name: %s", schemaName)
	}

	migrationUserName := fmt.Sprintf("%s_migration_user", schemaName)
	appUserName := fmt.Sprintf("%s_user", schemaName)

	buf := bytes.Buffer{}

	err := sqlTemplates.dropSchema.Execute(&buf, map[string]string{
		"SchemaName":        schemaName,
		"MigrationUserName": migrationUserName,
		"AppUserName":       appUserName,
	})

	if err != nil {
		return fmt.Errorf("failed to render drop schema template: %w", err)
	}

	if _, err := s.Exec(ctx, buf.String()); err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}

	return nil
}

// EnsureMigrationsTable ensures that the migrations table exists in the schema.
// This table is used to track applied migrations.
func (s *schemaStore) EnsureMigrationsTable() error {
	if s.conf == nil {
		return fmt.Errorf("schema configuration is required to ensure migrations table")
	}

	_, err := s.Exec(context.Background(), schemaStmt.createMigrationsTable)
	return err
}

// CreateAuthTable creates the authentication table in the schema for the given environment.
func (s *schemaStore) CreateAuthTable(ctx context.Context) error {
	if s.conf == nil {
		return fmt.Errorf("schema configuration is required to create auth table")
	}

	_, err := s.Conn.ExecContext(ctx, schemaStmt.createAuthTable)
	return err
}

type Migration struct {
	ID          types.ID
	Name        string
	ContentHash string
	ErrorMsg    null.String
}

// Migrations retrieves the list of last migrations.
func (s *schemaStore) Migrations(ctx context.Context) ([]Migration, error) {
	if s.conf == nil {
		return nil, fmt.Errorf("schema configuration is required to retrieve migrations")
	}

	rows, err := s.Query(ctx, schemaStmt.selectMigrations)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var migrations []Migration

	for rows.Next() {
		migration := Migration{}

		if err := rows.Scan(&migration.ID, &migration.Name, &migration.ContentHash, &migration.ErrorMsg); err != nil {
			return nil, err
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

type MigrationResult struct {
	Duration time.Duration
	FileName string
	Error    string
}

type ApplyMigrationArgs struct {
	MigrationName string
	Content       []byte
	SHA           string
	DeploymentID  types.ID
}

// ApplyMigration applies a migration to the schema if it hasn't been applied yet.
func (s *schemaStore) ApplyMigration(ctx context.Context, args ApplyMigrationArgs) (*MigrationResult, error) {
	if s.conf == nil {
		return nil, fmt.Errorf("schema configuration is required to apply migrations")
	}

	// Apply migration
	now := time.Now()
	_, migrationErr := s.Conn.ExecContext(ctx, string(args.Content))

	result := MigrationResult{
		Duration: time.Since(now),
		FileName: args.MigrationName,
	}

	if migrationErr != nil {
		result.Error = migrationErr.Error()
	}

	// Record applied migration
	_, err := s.Exec(ctx, `
		INSERT INTO
			stormkit_schema_migrations (
				migration_name, migration_duration_ms, deployment_id, content_hash, error_message
			)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (migration_name)
		DO UPDATE SET
			content_hash = EXCLUDED.content_hash,
			migration_duration_ms = EXCLUDED.migration_duration_ms,
			deployment_id = EXCLUDED.deployment_id,
			error_message = EXCLUDED.error_message,
			applied_at = NOW();
	`,
		args.MigrationName,
		result.Duration.Milliseconds(),
		args.DeploymentID,
		args.SHA,
		null.NewString(result.Error, result.Error != ""),
	)

	if err != nil {
		return &result, fmt.Errorf("failed to record applied migration %s: %w", args.MigrationName, err)
	}

	return &result, migrationErr
}

// InsertAuthUser inserts a new authentication user into the database.
func (s *schemaStore) InsertAuthUser(ctx context.Context, oauth *skauth.OAuth, user *skauth.User) error {
	if s.conf == nil {
		return fmt.Errorf("schema configuration is required to insert auth user")
	}

	tx, err := s.Conn.BeginTx(ctx, nil)

	if err != nil {
		fmt.Println("Failed to begin transaction:", err)
		return err
	}

	// Defer rollback - will be a no-op if transaction is committed successfully
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx, schemaStmt.insertAuthUser, user.Email, user.FirstName, user.LastName, user.Avatar).Scan(&user.ID)

	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, schemaStmt.insertOAuth,
		user.ID,
		oauth.AccountID,
		utils.EncryptToString(oauth.AccessToken),
		utils.EncryptToString(oauth.RefreshToken),
		oauth.TokenType,
		oauth.ProviderName,
		oauth.Expiry,
	)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// AuthUser retrieves the authentication user by its ID.
func (s *schemaStore) AuthUser(ctx context.Context, authID types.ID) (*skauth.User, error) {
	if s.conf == nil {
		return nil, fmt.Errorf("schema configuration is required to retrieve auth user")
	}

	row, err := s.QueryRow(ctx, schemaStmt.selectAuthUser, authID)

	if err != nil {
		return nil, err
	}

	authUser := &skauth.User{}

	err = row.Scan(
		&authUser.ID,
		&authUser.FirstName,
		&authUser.LastName,
		&authUser.Email,
		&authUser.Avatar,
		&authUser.CreatedAt,
		&authUser.LastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return authUser, nil
}

// Close closes the schema store and its underlying database connection.
func (s *schemaStore) Close() error {
	s.conf.cachedStoresMux.Lock()
	defer s.conf.cachedStoresMux.Unlock()

	if s.conf != nil {
		delete(s.conf.cachedStores, fmt.Sprintf("%s:%s", s.accessType, s.conf.DBName))
	}

	return s.Conn.Close()
}

// See https://github.com/stormkit-io/stormkit-io/pull/56#discussion_r2603452242
const MaxSchemaNameLength = 47

// isSQLSafe validates that the name argument contains only safe characters
func isSQLSafe(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched && len(name) <= MaxSchemaNameLength
}
