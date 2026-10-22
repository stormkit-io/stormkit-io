package authwall

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"unicode"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const StatusAll = "all"
const StatusDev = "dev"
const StatusDisabled = ""

// MetadataMaxLength is the maximum length of a login's metadata, in bytes.
const MetadataMaxLength = 1024

// AuthWall is a login of a team's Auth Wall. It opens the walls of the apps in
// AppIDs, or of every app in the team when AllApps is set.
type AuthWall struct {
	LoginID       types.ID
	TeamID        types.ID
	LoginEmail    string
	LoginPassword string
	// Metadata is passed to the app in the X-Auth-Wall-Metadata header after
	// the visitor logs in.
	Metadata  string
	AllApps   bool
	AppIDs    []types.ID
	LastLogin utils.Unix
	CreatedAt utils.Unix
}

// TeamEnv is an environment of a team's app, with the mode of its wall.
type TeamEnv struct {
	AppID   types.ID
	AppName string
	EnvID   types.ID
	EnvName string
	Status  string
}

// ValidateMetadata reports why metadata cannot be passed to an app in a
// request header, or returns an empty string when it can.
func ValidateMetadata(metadata string) string {
	if len(metadata) > MetadataMaxLength {
		return fmt.Sprintf("Metadata must be at most %d bytes.", MetadataMaxLength)
	}

	for _, r := range metadata {
		if unicode.IsControl(r) {
			return "Metadata cannot contain line breaks or control characters."
		}
	}

	return ""
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
