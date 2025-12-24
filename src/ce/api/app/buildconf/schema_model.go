package buildconf

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sync"

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
	MigrationsEnabled bool   `json:"migrationsEnabled"`
	MigrationsFolder  string `json:"migrationsFolder"` // path in the application for migrations
	DBName            string `json:"dbName"`
	SchemaName        string `json:"schemaName"`
	Port              string `json:"port"`
	Host              string `json:"host"`
	SSLMode           string `json:"sslMode"`
	DriverName        string `json:"-"` // Used in tests to specify the driver name

	cachedStores    map[string]*schemaStore `json:"-"`
	cachedStoresMux sync.Mutex              `json:"-"`
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

const SchemaAccessTypeMigrations = "migrations"
const SchemaAccessTypeAppUser = "app"

func (sc *SchemaConf) Store(accessType string) (*schemaStore, error) {
	sc.cachedStoresMux.Lock()
	defer sc.cachedStoresMux.Unlock()

	if sc.cachedStores == nil {
		sc.cachedStores = make(map[string]*schemaStore)
	}

	cacheKey := fmt.Sprintf("%s:%s", accessType, sc.DBName)

	if db, exists := sc.cachedStores[cacheKey]; exists {
		return db, nil
	}

	store, err := SchemaStoreFor(sc, accessType)

	if err != nil {
		return nil, err
	}

	sc.cachedStores[cacheKey] = store
	return store, err
}

// URL returns the psql connection URL.
func (sc *SchemaConf) URL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?search_path=%s&sslmode=%s",
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
