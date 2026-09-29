package authwall

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

const (
	formPurpose    = "auth-wall-form"
	sessionPurpose = "auth-wall-session"

	// formMaxMins is how long a visitor has to submit the login form.
	formMaxMins = 5

	// sessionMaxMins is how long a visitor stays logged in.
	sessionMaxMins = 24 * 60
)

// Token issues and verifies the tokens of an environment's Auth Wall. Every
// token carries its purpose and environment, so the login form's token is not
// a session, and a session of one environment does not open another.
type Token struct {
	EnvID types.ID
}

// Form returns the token embedded in the login form.
func (t Token) Form() (string, error) {
	return t.issue(formPurpose)
}

// Session returns the token issued after a successful login.
func (t Token) Session() (string, error) {
	return t.issue(sessionPurpose)
}

// IsValidForm reports whether token is this environment's login form token.
func (t Token) IsValidForm(token string) bool {
	return t.verify(verifyParams{token: token, purpose: formPurpose, maxMins: formMaxMins})
}

// IsValidSession reports whether token is a session of this environment.
func (t Token) IsValidSession(token string) bool {
	return t.verify(verifyParams{token: token, purpose: sessionPurpose, maxMins: sessionMaxMins})
}

func (t Token) issue(purpose string) (string, error) {
	return user.JWT(jwt.MapClaims{
		"purpose": purpose,
		"envId":   t.EnvID.String(),
	})
}

type verifyParams struct {
	token   string
	purpose string
	maxMins int
}

func (t Token) verify(p verifyParams) bool {
	if p.token == "" || t.EnvID == 0 {
		return false
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: p.token, MaxMins: p.maxMins})

	return claims != nil && claims["purpose"] == p.purpose && claims["envId"] == t.EnvID.String()
}
