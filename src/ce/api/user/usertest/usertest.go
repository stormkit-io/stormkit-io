package usertest

import (
	"fmt"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// Authorization returns the session token required to login for the given user id.
func Authorization(userID types.ID) string {
	token, _ := user.JWT(jwt.MapClaims{
		"uid": userID.String(),
	})

	return fmt.Sprintf("Bearer %s", token)
}
