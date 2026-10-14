package apikey

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	SCOPE_ENV   = "env"
	SCOPE_APP   = "app"
	SCOPE_ADMIN = "admin"
	SCOPE_TEAM  = "team"
	SCOPE_USER  = "user"
)

var AllowedScopes = []string{
	SCOPE_TEAM,
	SCOPE_ENV,
	SCOPE_USER,
	SCOPE_APP,
	SCOPE_ADMIN,
}

type Token struct {
	ID     types.ID `json:"id"`
	AppID  types.ID `json:"appId"`
	EnvID  types.ID `json:"envId"`
	UserID types.ID `json:"userId"`
	TeamID types.ID `json:"teamId"`
	Name   string   `json:"name"`
	Value  string   `json:"token"`
	Scope  string   `json:"scope"`

	// ReadOnly restricts the key to requests that do not change state: GET and
	// HEAD on the REST API, and read-only tools on the MCP server.
	ReadOnly bool `json:"readOnly"`
}

// AllowsMethod reports whether the key may make a request with the given HTTP
// method. Read-only keys are limited to methods that do not change state.
func (t *Token) AllowsMethod(method string) bool {
	if !t.ReadOnly {
		return true
	}

	switch method {
	case "", http.MethodGet, http.MethodHead:
		// An empty method means GET, as in net/http.
		return true
	default:
		return false
	}
}

func GenerateTokenValue() string {
	return fmt.Sprintf("SK_%s", utils.RandomToken(62))
}

// IsScopeValid validates the given scope.
func IsScopeValid(scope string) bool {
	found := false

	for _, allowedScope := range AllowedScopes {
		if scope == allowedScope {
			found = true
			break
		}
	}

	return found
}
