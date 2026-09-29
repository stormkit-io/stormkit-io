package authwall

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

const (
	formPurpose    = "auth-wall-form"
	sessionPurpose = "auth-wall-session"

	// formMaxMins is how long a visitor has to submit the login form.
	formMaxMins = 5

	// formReturnMaxMins is how long an expired form still sends the visitor
	// back to their page, with an error, instead of failing outright.
	formReturnMaxMins = 24 * 60

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

// Form is a verified login form token of an environment.
type Form struct {
	// ReturnTo is the URL of the protected page the form was shown on.
	ReturnTo string

	// Expired is true when the form was not submitted in time.
	Expired bool
}

// ParseForm verifies a login form token of this environment. It returns nil
// when the token is not one, or is too old to trust its return URL.
func (t Token) ParseForm(token string) *Form {
	claims := t.verify(verifyParams{token: token, purpose: formPurpose, maxMins: formReturnMaxMins})

	if claims == nil {
		return nil
	}

	returnTo, _ := claims["returnTo"].(string)

	if returnTo == "" {
		return nil
	}

	issued, _ := claims["issued"].(float64)
	age := time.Since(time.Unix(int64(issued), 0))

	return &Form{
		ReturnTo: returnTo,
		Expired:  age > formMaxMins*time.Minute,
	}
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
