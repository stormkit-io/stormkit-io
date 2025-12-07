package buildconf

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
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
	selectTables: `SELECT
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
	ORDER BY 3;`,
}

type schemaStore struct {
	*database.Store
}

// NewStore returns a store instance.
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
func (s *schemaStore) CreateSchema(ctx context.Context, schemaName string) error {
	// Validate schema name to prevent SQL injection
	if !isValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name: %s", schemaName)
	}

	_, err := s.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, schemaName))
	return err
}

// isValidSchemaName validates that the schema name contains only safe characters
func isValidSchemaName(name string) bool {
	// Schema names should only contain letters, numbers, and underscores
	// and should start with a letter or underscore
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched && len(name) <= 63 // PostgreSQL identifier length limit
}
