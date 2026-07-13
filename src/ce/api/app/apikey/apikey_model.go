package apikey

import (
	"fmt"

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
