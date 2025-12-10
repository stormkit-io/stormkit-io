package buildconf

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var schemaStmt = struct {
	selectSchema string
	selectTables string
}{
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
				ALTER ROLE "{{.MigrationUserName}}" CONNECTION LIMIT 10;

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

				-- Revoke public schema access
				REVOKE ALL ON SCHEMA public FROM "{{.AppUserName}}";
				REVOKE ALL ON DATABASE postgres FROM "{{.AppUserName}}";

				-- Grant DML permissions (data operations only)
				GRANT USAGE ON SCHEMA "{{.SchemaName}}" TO "{{.AppUserName}}";
				GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA "{{.SchemaName}}" TO "{{.AppUserName}}";
				GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA "{{.SchemaName}}" TO "{{.AppUserName}}";
				ALTER DEFAULT PRIVILEGES IN SCHEMA "{{.SchemaName}}" GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO "{{.AppUserName}}";
				ALTER DEFAULT PRIVILEGES IN SCHEMA "{{.SchemaName}}" GRANT USAGE, SELECT ON SEQUENCES TO "{{.AppUserName}}";
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
}

// SchemaStore returns a store instance.
func SchemaStore() *schemaStore {
	return &schemaStore{
		Store: database.NewStore(),
	}
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
		return nil, errors.New("schema already exists")
	}

	// Validate schema name to prevent SQL injection
	if !isSQLSafe(schemaName) {
		return nil, fmt.Errorf("invalid schema name: %s", schemaName)
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
		"SchemaName":      schemaName,
		"AppUserName":     appUserName,
		"AppUserPassword": appPassword,
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

// See https://github.com/stormkit-io/stormkit-io/pull/56#discussion_r2603452242
const MAX_SCHEMA_NAME_LENGTH = 47

// isSQLSafe validates that the name argument contains only safe characters
func isSQLSafe(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched && len(name) <= MAX_SCHEMA_NAME_LENGTH
}
