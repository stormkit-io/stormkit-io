package buildconf

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

type CustomCert struct {
	Value string `json:"value"` // The certificate value
	Key   string `json:"key"`   // The private key
}

type DomainModel struct {
	ID         types.ID
	EnvID      types.ID
	AppID      types.ID
	Name       string
	Verified   bool
	Token      null.String
	VerifiedAt utils.Unix
	CreatedAt  utils.Unix
	CustomCert *CustomCert
	LastPing   *PingResult
}

type PingResult struct {
	DomainID   types.ID   `json:"-"`
	Status     int        `json:"status"`
	Error      string     `json:"error,omitempty"`
	LastPingAt utils.Unix `json:"lastPingAt"`
}

// Scan implements the Scanner interface.
func (pr *PingResult) Scan(value any) error {
	if value != nil {
		if b, ok := value.([]byte); ok {
			if err := json.Unmarshal(b, pr); err != nil {
				return err
			}

			// This makes sure the time is in UTC
			pr.LastPingAt = utils.UnixFrom(pr.LastPingAt.UTC())
		}
	}

	return nil
}

// Value implements the Sql Driver interface.
func (pr *PingResult) Value() (driver.Value, error) {
	if pr == nil {
		return nil, nil
	}

	return json.Marshal(pr)
}
