package authwallhandlers_test

import (
	"context"
	"net/http"
	"net/url"
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
	"github.com/stretchr/testify/suite"
)

type HandlerAuthSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	aw   *authwall.AuthWall
}

func (s *HandlerAuthSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	s.aw = &authwall.AuthWall{
		LoginEmail:    "email@example.org",
		LoginPassword: "123pass",
		EnvID:         env.ID,
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), s.aw))
}

func (s *HandlerAuthSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

const (
	protectedPage = "https://site.example.org/page?a=b"
	attackerPage  = "https://attacker.example.com/form"
)

type loginParams struct {
	Password string
	Token    string
}

// login posts the Auth Wall login form from another site, whose Referer must
// never be used as the redirect target.
func (s *HandlerAuthSuite) login(p loginParams) shttptest.Response {
	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"email":    []byte(s.aw.LoginEmail),
		"password": []byte(p.Password),
		"envId":    []byte(s.aw.EnvID.String()),
		"token":    []byte(p.Token),
	}, nil)

	s.Require().NoError(err)

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/login",
		requestBody,
		map[string]string{
			"Content-Type": contentType,
			"Referer":      attackerPage,
		},
	)
}

func (s *HandlerAuthSuite) formToken() string {
	token, err := authwall.Token{EnvID: s.aw.EnvID}.Form(protectedPage)
	s.Require().NoError(err)

	return token
}

// Test_Auth_RejectsOtherTokens verifies that the login form only accepts the
// form token of its own environment.
func (s *HandlerAuthSuite) Test_Auth_RejectsOtherTokens() {
	anonymous, err := user.JWT(jwt.MapClaims{})
	s.Require().NoError(err)

	otherEnv, err := authwall.Token{EnvID: s.aw.EnvID + 1}.Form(protectedPage)
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
	response := s.login(loginParams{Password: s.aw.LoginPassword, Token: s.formToken()})

	s.Equal(http.StatusFound, response.Code)

	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)
	s.Equal("site.example.org", location.Host)
	s.Equal("/page", location.Path)
	s.Equal("b", location.Query().Get("a"))

	session := location.Query().Get("stormkit_success")
	s.True(authwall.Token{EnvID: s.aw.EnvID}.IsValidSession(session))

	_, isForm := authwall.Token{EnvID: s.aw.EnvID}.FormReturnTo(session)
	s.False(isForm)

	logins, err := authwall.Store().Logins(context.Background(), s.aw.EnvID)
	s.NoError(err)
	s.Len(logins, 1)
	s.Equal(s.aw.LoginEmail, logins[0].LoginEmail)
	s.GreaterOrEqual(now, logins[0].LastLogin.Unix())
}

func (s *HandlerAuthSuite) Test_Auth_FailPassword() {
	response := s.login(loginParams{Password: "some-password", Token: s.formToken()})

	s.Equal(http.StatusFound, response.Code)
	s.Equal(protectedPage+"&stormkit_error=invalid_credentials", response.Header().Get("Location"))

	logins, err := authwall.Store().Logins(context.Background(), s.aw.EnvID)
	s.NoError(err)
	s.Len(logins, 1)
	s.Equal(s.aw.LoginEmail, logins[0].LoginEmail)
	s.False(logins[0].LastLogin.Valid)
}

func TestHandlerAuthSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthSuite{})
}
