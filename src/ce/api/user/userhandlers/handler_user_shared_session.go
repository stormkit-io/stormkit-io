package userhandlers

import (
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type userSharedSessionArgs struct {
	Hours int `json:"hours"`
}

// handlerUserSharedSession generates a short-lived session token for the
// authenticated user. The token can be shared over a private channel to allow
// admin access to the instance without creating a new user account.
//
// An optional `hours` field (1–24) controls the token lifetime; it defaults
// to 1. Non-Enterprise users are clamped to 1h regardless of the requested value.
func handlerUserSharedSession(req *user.RequestContext) *shttp.Response {
	body := userSharedSessionArgs{}

	if err := req.Post(&body); err != nil {
		return shttp.Error(err)
	}

	hours := body.Hours

	if hours < 1 || hours > 24 {
		hours = 1
	}

	// Limit the token lifetime to 1 hour for non-Enterprise users, regardless of the requested duration.
	if hours > 1 && !req.License().IsEnterprise() {
		hours = 1
	}

	token, err := user.JWT(jwt.MapClaims{
		"uid": req.User.ID,
		"exp": time.Now().Add(time.Duration(hours) * time.Hour).Unix(),
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{"token": token},
	}
}
