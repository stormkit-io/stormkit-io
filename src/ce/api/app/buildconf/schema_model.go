package buildconf

import (
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// schemaStoreCache is a package-level cache of open schema store connections,
// keyed by access type and connection coordinates. It survives across requests,
// unlike the per-request SchemaConf structs that are deserialized fresh from
// the database each time.
var schemaStoreCache sync.Map

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
	MigrationsEnabled bool   `json:"migrationsEnabled"`
	InjectEnvVars     bool   `json:"injectEnvVars"`
	MigrationsFolder  string `json:"migrationsFolder"` // path in the application for migrations
	DBName            string `json:"dbName"`
	SchemaName        string `json:"schemaName"`
	Port              string `json:"port"`
	Host              string `json:"host"`
	SSLMode           string `json:"sslMode"`
	DriverName        string `json:"-"` // Used in tests to specify the driver name

}

// Value implements the Sql Driver interface.
func (sc *SchemaConf) Value() (driver.Value, error) {
	return utils.ByteaValue(sc)
}

const SchemaAccessTypeMigrations = "migrations"
const SchemaAccessTypeAppUser = "app"

func (sc *SchemaConf) storeKey(accessType string) string {
	return fmt.Sprintf("%s:%s@%s:%s/%s/%s", accessType, sc.AppUserName, sc.Host, sc.Port, sc.DBName, sc.SchemaName)
}

func (sc *SchemaConf) Store(accessType string) (*schemaStore, error) {
	cacheKey := sc.storeKey(accessType)

	if cached, ok := schemaStoreCache.Load(cacheKey); ok {
		return cached.(*schemaStore), nil
	}

	store, err := SchemaStoreFor(sc, accessType)

	if err != nil {
		return nil, err
	}

	schemaStoreCache.Store(cacheKey, store)
	return store, nil
}

// URL returns the psql connection URL.
func (sc *SchemaConf) URL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?options=-csearch_path=%s&sslmode=%s",
		sc.AppUserName,
		sc.AppPassword,
		sc.Host,
		sc.Port,
		sc.DBName,
		sc.SchemaName,
		utils.GetString(sc.SSLMode, "disable"),
	)
}

// String returns the psql connection string.
func (sc *SchemaConf) String() string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s search_path=%s sslmode=%s",
		sc.Host,
		sc.Port,
		sc.DBName,
		sc.AppUserName,
		sc.AppPassword,
		sc.SchemaName,
		utils.GetString(sc.SSLMode, "disable"),
	)
}
