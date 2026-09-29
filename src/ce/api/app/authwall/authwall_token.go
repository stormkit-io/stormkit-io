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

// Form returns the token embedded in the login form. It carries the URL of
// the protected page, the only place the visitor is sent back to after
// submitting the form.
func (t Token) Form(returnTo string) (string, error) {
	return user.JWT(jwt.MapClaims{
		"purpose":  formPurpose,
		"envId":    t.EnvID.String(),
		"returnTo": returnTo,
	})
}

// Session returns the token issued after a successful login.
func (t Token) Session() (string, error) {
	return t.issue(sessionPurpose)
}

// FormReturnTo returns the page URL of this environment's login form token,
// and false when the token is not a valid form token of this environment.
func (t Token) FormReturnTo(token string) (string, bool) {
	claims := t.verify(verifyParams{token: token, purpose: formPurpose, maxMins: formMaxMins})

	if claims == nil {
		return "", false
	}

	returnTo, _ := claims["returnTo"].(string)

	return returnTo, returnTo != ""
}

// IsValidSession reports whether token is a session of this environment.
func (t Token) IsValidSession(token string) bool {
	return t.verify(verifyParams{token: token, purpose: sessionPurpose, maxMins: sessionMaxMins}) != nil
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

// verify returns the token's claims when it is valid for the given purpose and
// this environment, and nil otherwise.
func (t Token) verify(p verifyParams) jwt.MapClaims {
	if p.token == "" || t.EnvID == 0 {
		return nil
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: p.token, MaxMins: p.maxMins})

	if claims == nil || claims["purpose"] != p.purpose || claims["envId"] != t.EnvID.String() {
		return nil
	}

	return claims
}
