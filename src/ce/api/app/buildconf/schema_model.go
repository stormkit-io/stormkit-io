package buildconf

import (
	"context"
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

// authSchemaEnsured tracks, per process, which schemas have had their auth
// tables reconciled (CreateAuthTable, including idempotent column ALTERs) since
// boot, so EnsureAuthSchema runs the DDL at most once per schema. Keyed like
// schemaStoreCache.
var authSchemaEnsured sync.Map

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
	// Migration stores are never cached: migration roles have CONNECTION LIMIT 1,
	// so a pooled idle connection would permanently occupy the role's only slot.
	// Callers open them for short-lived DDL and must Close them when done.
	if accessType == SchemaAccessTypeMigrations {
		return SchemaStoreFor(sc, accessType)
	}

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

// EnsureAuthSchema guarantees the auth tables in this schema match the current
// DDL, including columns (e.g. metadata) added after the tenant first enabled
// auth. It runs the idempotent CreateAuthTable through the migration role once
// per schema per process; a failure is not cached, so the next call retries.
func (sc *SchemaConf) EnsureAuthSchema(ctx context.Context) error {
	key := sc.storeKey(SchemaAccessTypeMigrations)

	if _, ok := authSchemaEnsured.Load(key); ok {
		return nil
	}

	store, err := sc.Store(SchemaAccessTypeMigrations)

	if err != nil {
		return err
	}

	defer store.Close()

	if err := store.CreateAuthTable(ctx); err != nil {
		return err
	}

	authSchemaEnsured.Store(key, struct{}{})
	return nil
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
