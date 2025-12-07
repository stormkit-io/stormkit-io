package buildconf

import (
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type SchemaTable struct {
	Name string
	Size int64 // in bytes
	Rows int64 // estimated number of rows
}

type Schema struct {
	Name   string
	Tables []SchemaTable
}

// SchemaName returns the schema name for the given app and environment IDs.
func SchemaName(appID, envID types.ID) string {
	return fmt.Sprintf("a%de%d", appID, envID)
}

// Map returns the map representation of the schema.
func (s *Schema) Map() map[string]any {
	tables := make([]map[string]any, 0, len(s.Tables))

	for _, table := range s.Tables {
		tables = append(tables, map[string]any{
			"name": table.Name,
			"size": table.Size,
			"rows": table.Rows,
		})
	}

	return map[string]any{
		"name":   s.Name,
		"tables": tables,
	}
}
