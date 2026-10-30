package authwall

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const StatusAll = "all"
const StatusDev = "dev"
const StatusDisabled = ""

// MetadataMaxLength is the maximum length of a login's metadata, in bytes.
const MetadataMaxLength = 1024

// PublicPathsMax is the maximum number of public paths of an environment.
const PublicPathsMax = 50

// PublicPathMaxLength is the maximum length of a public path, in bytes.
const PublicPathMaxLength = 256

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

	// PublicPaths skip the environment's wall.
	PublicPaths []string
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

// ValidatePublicPaths returns the paths trimmed and without duplicates, or the
// reason they cannot be used.
//
// A path is either exact, such as /robots.txt, or ends with /* to match
// everything below it, such as /api/webhooks/*. Nothing else is a pattern, so
// a path cannot open more than it says.
func ValidatePublicPaths(paths []string) ([]string, string) {
	clean := []string{}
	seen := map[string]bool{}

	for _, p := range paths {
		p = strings.TrimSpace(p)

		if p == "" || seen[p] {
			continue
		}

		if len(p) > PublicPathMaxLength {
			return nil, fmt.Sprintf("Public paths must be at most %d characters.", PublicPathMaxLength)
		}

		if !strings.HasPrefix(p, "/") {
			return nil, fmt.Sprintf("Public path %q must start with a slash.", p)
		}

		if strings.ContainsAny(p, "?#") || strings.IndexFunc(p, unicode.IsSpace) != -1 || strings.IndexFunc(p, unicode.IsControl) != -1 {
			return nil, fmt.Sprintf("Public path %q cannot contain spaces, query strings or fragments.", p)
		}

		if i := strings.Index(p, "*"); i != -1 && (i != len(p)-1 || !strings.HasSuffix(p, "/*")) {
			return nil, fmt.Sprintf("Public path %q can only use * at the end, after a slash, such as /api/webhooks/*.", p)
		}

		if p == "/*" {
			return nil, "Public path /* opens the whole environment. Turn its Auth Wall off instead."
		}

		if hasDotSegment(strings.TrimSuffix(p, "*")) {
			return nil, fmt.Sprintf("Public path %q cannot contain . or .. segments.", p)
		}

		seen[p] = true
		clean = append(clean, p)
	}

	if len(clean) > PublicPathsMax {
		return nil, fmt.Sprintf("An environment can have at most %d public paths.", PublicPathsMax)
	}

	return clean, ""
}

// IsPublicPath reports whether the request path is one of the public paths,
// which skip the wall.
//
// A path with . or .. segments is never public: the app may resolve it to a
// path outside the public one, and the wall would have let it through.
func IsPublicPath(publicPaths []string, path string) bool {
	if len(publicPaths) == 0 || hasDotSegment(path) {
		return false
	}

	for _, p := range publicPaths {
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if path == p {
			return true
		}
	}

	return false
}

func hasDotSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return true
		}
	}

	return false
}

type Config struct {
	Status string `json:"status"`

	// PublicPaths skip the wall. See IsPublicPath.
	PublicPaths []string `json:"publicPaths,omitempty"`
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
