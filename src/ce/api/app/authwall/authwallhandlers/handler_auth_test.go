package authwallhandlers_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"
)

type HandlerAuthSuite struct {
	suite.Suite
	*factory.Factory
	conn  databasetest.TestDB
	usr   *factory.MockUser
	app   *factory.MockApp
	envID types.ID
	aw    *authwall.AuthWall
}

func (s *HandlerAuthSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.usr = s.MockUser()
	s.app = s.MockApp(s.usr)
	s.envID = s.MockEnv(s.app).ID
	s.aw = &authwall.AuthWall{
		TeamID:        s.app.TeamID,
		LoginEmail:    "email@example.org",
		LoginPassword: "123pass",
		Metadata:      `{"role":"client"}`,
		AppIDs:        []types.ID{s.app.ID},
	}

	s.Require().NoError(authwall.Store().CreateLogin(context.Background(), s.aw))
}

func (s *HandlerAuthSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

const (
	protectedPage = "https://site.example.org/page?a=b"
	attackerPage  = "https://attacker.example.com/form"
)

type loginParams struct {
	Email    string
	Password string
	Token    string

	// EnvID defaults to the environment of the suite's app.
	EnvID types.ID

	// Referer defaults to a page on another site.
	Referer string
}

// login posts the Auth Wall login form from another site, whose Referer must
// never be used as the redirect target.
func (s *HandlerAuthSuite) login(p loginParams) shttptest.Response {
	if p.Email == "" {
		p.Email = s.aw.LoginEmail
	}

	if p.EnvID == 0 {
		p.EnvID = s.envID
	}

	if p.Token == "" {
		token, err := authwall.Token{EnvID: p.EnvID}.Form(protectedPage)
		s.Require().NoError(err)
		p.Token = token
	}

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"email":    []byte(p.Email),
		"password": []byte(p.Password),
		"envId":    []byte(p.EnvID.String()),
		"token":    []byte(p.Token),
	}, nil)

	s.Require().NoError(err)

	referer := p.Referer

	if referer == "" {
		referer = attackerPage
	}

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/login",
		requestBody,
		map[string]string{
			"Content-Type": contentType,
			"Referer":      referer,
		},
	)
}

// session returns the session a login response hands back, or nil when the
// login failed.
func (s *HandlerAuthSuite) session(response shttptest.Response, envID types.ID) *authwall.Session {
	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)

	return authwall.Token{EnvID: envID}.ParseSession(location.Query().Get("stormkit_success"))
}

// Test_Auth_RejectsOtherTokens verifies that the login form only accepts the
// form token of its own environment.
func (s *HandlerAuthSuite) Test_Auth_RejectsOtherTokens() {
	anonymous, err := user.JWT(user.JWTParams{Purpose: user.PurposeOAuthState, Claims: jwt.MapClaims{}})
	s.Require().NoError(err)

	otherEnv, err := authwall.Token{EnvID: s.envID + 1}.Form(protectedPage)
	s.Require().NoError(err)

	for _, token := range []string{anonymous, otherEnv} {
		response := s.login(loginParams{Password: s.aw.LoginPassword, Token: token})

		s.Equal(http.StatusBadRequest, response.Code)
		s.Empty(response.Header().Get("Location"))
	}
}

// Test_Auth_Success verifies that a successful login returns the visitor, with
// the session, to the protected page carried by the form token, not to the
// Referer.
func (s *HandlerAuthSuite) Test_Auth_Success() {
	now := time.Now().UTC().Unix()
	response := s.login(loginParams{Password: s.aw.LoginPassword})

	s.Equal(http.StatusFound, response.Code)

	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)
	s.Equal("site.example.org", location.Host)
	s.Equal("/page", location.Path)
	s.Equal("b", location.Query().Get("a"))

	session := location.Query().Get("stormkit_success")
	s.Equal(&authwall.Session{
		Email:    s.aw.LoginEmail,
		Metadata: s.aw.Metadata,
	}, authwall.Token{EnvID: s.envID}.ParseSession(session))

	s.Nil(authwall.Token{EnvID: s.envID}.ParseForm(session))

	login, err := authwall.Store().TeamLogin(context.Background(), s.app.TeamID, s.aw.LoginID)
	s.Require().NoError(err)
	s.Require().NotNil(login)
	s.True(login.LastLogin.Valid, "the login time is recorded")
	s.GreaterOrEqual(login.LastLogin.Unix(), now)
}

// Test_Auth_HTTPSBehindProxy verifies that the return URL is upgraded to https
// when the browser was on https, as behind a proxy that terminates TLS.
func (s *HandlerAuthSuite) Test_Auth_HTTPSBehindProxy() {
	token, err := authwall.Token{EnvID: s.envID}.Form("http://site.example.org/page")
	s.Require().NoError(err)

	response := s.login(loginParams{Password: s.aw.LoginPassword, Token: token, Referer: "https://site.example.org/"})

	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)
	s.Equal("https", location.Scheme)
	s.Equal("site.example.org", location.Host)
}

// Test_Auth_ExpiredForm verifies that a form submitted too late sends the
// visitor back to the page with an error instead of logging them in.
func (s *HandlerAuthSuite) Test_Auth_ExpiredForm() {
	token, err := user.JWT(user.JWTParams{Purpose: user.PurposeAuthWallForm, Claims: jwt.MapClaims{
		"envId":    s.envID.String(),
		"returnTo": protectedPage,
		"issued":   time.Now().Add(-10 * time.Minute).Unix(),
	}})
	s.Require().NoError(err)

	response := s.login(loginParams{Password: s.aw.LoginPassword, Token: token})

	s.Equal(http.StatusFound, response.Code)
	s.Equal(protectedPage+"&stormkit_error=invalid_token", response.Header().Get("Location"))
}

func (s *HandlerAuthSuite) Test_Auth_FailPassword() {
	response := s.login(loginParams{Password: "some-password"})

	s.Equal(http.StatusFound, response.Code)
	s.Equal(protectedPage+"&stormkit_error=invalid_credentials", response.Header().Get("Location"))

	login, err := authwall.Store().TeamLogin(context.Background(), s.app.TeamID, s.aw.LoginID)
	s.Require().NoError(err)
	s.Require().NotNil(login)
	s.False(login.LastLogin.Valid)
}

// Test_Auth_EmailCaseInsensitive verifies that the email matches regardless of
// its case.
func (s *HandlerAuthSuite) Test_Auth_EmailCaseInsensitive() {
	response := s.login(loginParams{Email: "EMAIL@example.org", Password: s.aw.LoginPassword})
	session := s.session(response, s.envID)

	s.Require().NotNil(session)
	s.Equal(s.aw.LoginEmail, session.Email, "the app receives the stored email, not the typed one")
}

// Test_Auth_SessionHidesLogin verifies that the login's email and metadata are
// not readable in the session token, which is handed over in the page's URL.
func (s *HandlerAuthSuite) Test_Auth_SessionHidesLogin() {
	response := s.login(loginParams{Password: s.aw.LoginPassword})
	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)

	parts := strings.Split(location.Query().Get("stormkit_success"), ".")
	s.Require().Len(parts, 3)

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	s.Require().NoError(err)
	s.NotContains(string(payload), s.aw.LoginEmail)
	s.NotContains(string(payload), "client")
}

// Test_Auth_EveryEnvOfApp verifies that a login opens every environment of the
// apps it has access to.
func (s *HandlerAuthSuite) Test_Auth_EveryEnvOfApp() {
	envID := s.MockEnv(s.app, map[string]any{"Name": "staging", "Branch": "staging"}).ID
	response := s.login(loginParams{Password: s.aw.LoginPassword, EnvID: envID})

	s.NotNil(s.session(response, envID))
}

// Test_Auth_OtherAppOfTeam verifies that a login does not open the wall of an
// app of its team it has no access to, unless it has access to all apps.
func (s *HandlerAuthSuite) Test_Auth_OtherAppOfTeam() {
	envID := s.MockEnv(s.MockApp(s.usr)).ID
	response := s.login(loginParams{Password: s.aw.LoginPassword, EnvID: envID})

	s.Nil(s.session(response, envID))
	s.Equal(protectedPage+"&stormkit_error=invalid_credentials", response.Header().Get("Location"))

	_, err := s.conn.Exec("UPDATE auth_wall SET all_apps = TRUE WHERE login_id = $1", s.aw.LoginID)
	s.Require().NoError(err)

	response = s.login(loginParams{Password: s.aw.LoginPassword, EnvID: envID})

	s.NotNil(s.session(response, envID))
}

// Test_Auth_OtherTeam verifies that a login never opens the wall of another
// team's app, even with access to all apps.
func (s *HandlerAuthSuite) Test_Auth_OtherTeam() {
	_, err := s.conn.Exec("UPDATE auth_wall SET all_apps = TRUE WHERE login_id = $1", s.aw.LoginID)
	s.Require().NoError(err)

	envID := s.MockEnv(s.MockApp(s.MockUser())).ID
	response := s.login(loginParams{Password: s.aw.LoginPassword, EnvID: envID})

	s.Nil(s.session(response, envID))
}

// Test_Auth_LegacyPassword verifies that a password stored encrypted, before
// passwords were hashed, still logs in and is hashed on that login.
func (s *HandlerAuthSuite) Test_Auth_LegacyPassword() {
	_, err := s.conn.Exec("UPDATE auth_wall SET login_password = $1 WHERE login_id = $2", utils.EncryptToString("legacy-pass"), s.aw.LoginID)
	s.Require().NoError(err)

	response := s.login(loginParams{Password: "wrong-pass"})
	s.Nil(s.session(response, s.envID))

	response = s.login(loginParams{Password: "legacy-pass"})
	s.NotNil(s.session(response, s.envID))

	var stored string
	s.Require().NoError(s.conn.QueryRow("SELECT login_password FROM auth_wall WHERE login_id = $1", s.aw.LoginID).Scan(&stored))
	s.NoError(bcrypt.CompareHashAndPassword([]byte(stored), []byte("legacy-pass")))

	response = s.login(loginParams{Password: "legacy-pass"})
	s.NotNil(s.session(response, s.envID))
}

func TestHandlerAuthSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthSuite{})
}
