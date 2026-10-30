package authwall

import (
	"encoding/json"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	// formMaxMins is how long a visitor has to submit the login form.
	formMaxMins = 5

	// formReturnMaxMins is how long an expired form still sends the visitor
	// back to their page, with an error, instead of failing outright.
	formReturnMaxMins = 24 * 60

	// sessionMaxMins is how long a visitor stays logged in.
	sessionMaxMins = 24 * 60

	// triggerMaxMins is how long a trigger's token opens the wall. A token is
	// issued for every run, so it only has to outlive the request.
	triggerMaxMins = 5
)

// HeaderTriggerToken carries the token that lets a periodic trigger's request
// through the wall of the trigger's environment.
const HeaderTriggerToken = "X-Stormkit-Trigger-Token"

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
	return user.JWT(user.JWTParams{
		Purpose: user.PurposeAuthWallForm,
		Claims: jwt.MapClaims{
			"envId":    t.EnvID.String(),
			"returnTo": returnTo,
		},
	})
}

// Session is a verified session of an environment's Auth Wall.
type Session struct {
	// Email is the email of the login the visitor logged in with.
	Email string `json:"email,omitempty"`

	// Metadata is the login's metadata at the time the visitor logged in.
	Metadata string `json:"metadata,omitempty"`
}

// Session returns the token issued after a successful login. The login's email
// and metadata ride the token, so they reach the app without a lookup on every
// request; a change to the metadata reaches it on the next login. They are
// encrypted, since the token is first handed over in the page's URL.
func (t Token) Session(s Session) (string, error) {
	login, err := json.Marshal(s)

	if err != nil {
		return "", err
	}

	return user.JWT(user.JWTParams{
		Purpose: user.PurposeAuthWallSession,
		Claims: jwt.MapClaims{
			"envId": t.EnvID.String(),
			"login": utils.EncryptToString(string(login)),
		},
	})
}

// Trigger returns the token a run of the given trigger sends to pass the wall.
func (t Token) Trigger(triggerID types.ID) (string, error) {
	return user.JWT(user.JWTParams{
		Purpose: user.PurposeAuthWallTrigger,
		Claims: jwt.MapClaims{
			"envId":     t.EnvID.String(),
			"triggerId": triggerID.String(),
		},
	})
}

// ParseTrigger verifies a trigger token of this environment and returns the
// trigger it was issued for, or 0 when the token is not one.
func (t Token) ParseTrigger(token string) types.ID {
	claims := t.verify(verifyParams{token: token, purpose: user.PurposeAuthWallTrigger, maxMins: triggerMaxMins})

	if claims == nil {
		return 0
	}

	triggerID, _ := claims["triggerId"].(string)

	return utils.StringToID(triggerID)
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
	claims := t.verify(verifyParams{token: token, purpose: user.PurposeAuthWallForm, maxMins: formReturnMaxMins})

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

// ParseSession verifies a session token of this environment. It returns nil
// when the token is not one.
func (t Token) ParseSession(token string) *Session {
	claims := t.verify(verifyParams{token: token, purpose: user.PurposeAuthWallSession, maxMins: sessionMaxMins})

	if claims == nil {
		return nil
	}

	session := &Session{}

	if login, _ := claims["login"].(string); login != "" {
		_ = json.Unmarshal([]byte(utils.DecryptToString(login)), session)
	}

	return session
}

type verifyParams struct {
	token   string
	purpose user.Purpose
	maxMins int
}

// verify returns the token's claims when it is valid for the given purpose and
// this environment, and nil otherwise.
func (t Token) verify(p verifyParams) jwt.MapClaims {
	if p.token == "" || t.EnvID == 0 {
		return nil
	}

	claims := user.ParseJWT(&user.ParseJWTArgs{Bearer: p.token, MaxMins: p.maxMins, Purposes: []user.Purpose{p.purpose}})

	if claims == nil || claims["envId"] != t.EnvID.String() {
		return nil
	}

	return claims
}
