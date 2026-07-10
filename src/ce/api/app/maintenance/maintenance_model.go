package maintenance

import (
	"database/sql/driver"
	"encoding/json"
)

const StatusOn = "on"
const StatusDisabled = ""

// Config is the per-environment maintenance mode configuration. It is stored
// as jsonb so future iterations (custom redirect, custom HTML) can extend it
// without a schema change.
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
