package buildconf

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
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

type SchemaConf struct {
	AppUserName       string `json:"appUserName"`
	AppPassword       string `json:"appPassword"`
	MigrationUserName string `json:"migrationUserName"`
	MigrationPassword string `json:"migrationPassword"`
	DBName            string `json:"dbName"`
	SchemaName        string `json:"schemaName"`
	Port              string `json:"port"`
	Host              string `json:"host"`
}

// Scan implements the Scanner interface.
func (sc *SchemaConf) Scan(value any) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)

	if !ok {
		return fmt.Errorf("failed to scan SchemaConf: invalid type %T", value)
	}

	decrypted, err := utils.Decrypt(b)

	if err != nil {
		return err
	}

	return json.Unmarshal(decrypted, sc)
}

// Value implements the Sql Driver interface.
func (sc *SchemaConf) Value() (driver.Value, error) {
	if sc == nil {
		return nil, nil
	}

	js, err := json.Marshal(sc)

	if err != nil {
		return nil, err
	}

	return utils.Encrypt(js)
}

// String returns the psql connection string.
func (sc *SchemaConf) String() string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s search_path=%s sslmode=disable",
		sc.Host,
		sc.Port,
		sc.DBName,
		sc.AppUserName,
		sc.AppPassword,
		sc.SchemaName,
	)
}
