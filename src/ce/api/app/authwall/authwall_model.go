package authwall

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const StatusAll = "all"
const StatusDev = "dev"
const StatusDisabled = ""

type AuthWall struct {
	EnvID         types.ID
	LoginID       types.ID
	LoginEmail    string
	LoginPassword string
	LastLogin     utils.Unix
	CreatedAt     utils.Unix
}

type Config struct {
	Status string `json:"status"`
}

// Scan implements the Scanner interface.
func (cnf *Config) Scan(value any) error {
	if value != nil {
		if b, ok := value.([]byte); ok {
			if err := json.Unmarshal(b, &cnf); err != nil {
				return err
			}
		}
	}

	return nil
}

// Value implements the Sql Driver interface.
func (cnf *Config) Value() (driver.Value, error) {
	if cnf == nil {
		return nil, nil
	}

	return json.Marshal(cnf)
}
