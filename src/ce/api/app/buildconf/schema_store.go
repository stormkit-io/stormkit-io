package buildconf

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
)

var schemaStmt = struct {
	selectTables string
}{
	selectTables: `SELECT
		t.table_schema,
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
	rows, err := s.Query(ctx, schemaStmt.selectTables, schemaName)

	if err != nil {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	schema := Schema{
		Name:   schemaName,
		Tables: []SchemaTable{},
	}

	for rows.Next() {
		table := SchemaTable{}

		if err := rows.Scan(&schemaName, &table.Name, &table.Size, &table.Rows); err != nil {
			return nil, err
		}

		schema.Tables = append(schema.Tables, table)
	}

	return &schema, nil
}
