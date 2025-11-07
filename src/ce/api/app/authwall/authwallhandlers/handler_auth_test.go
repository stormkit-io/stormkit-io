package authwallhandlers_test

import (
	"context"
	"fmt"
	"net/http"
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

func (s *HandlerAuthSuite) Test_Auth_Success() {
	now := time.Now().UTC().Unix()
	token, err := user.JWT(jwt.MapClaims{})

	s.NoError(err)

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"email":    []byte(s.aw.LoginEmail),
		"password": []byte(s.aw.LoginPassword),
		"envId":    []byte(s.aw.EnvID.String()),
		"token":    []byte(token),
	}, nil)

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/login",
		requestBody,
		map[string]string{
			"Content-Type": contentType,
			"Referer":      "http://example.org",
		},
	)

	expectedReferer := fmt.Sprintf("http://example.org?stormkit_success=%s", token)

	s.Equal(http.StatusFound, response.Code)
	s.Equal(expectedReferer, response.Header().Get("Location"))

	logins, err := authwall.Store().Logins(context.Background(), s.aw.EnvID)
	s.NoError(err)
	s.Len(logins, 1)
	s.Equal(s.aw.LoginEmail, logins[0].LoginEmail)
	s.GreaterOrEqual(now, logins[0].LastLogin.Unix())
}

func (s *HandlerAuthSuite) Test_Auth_FailPassword() {
	token, err := user.JWT(jwt.MapClaims{})

	s.NoError(err)

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"email":    []byte(s.aw.LoginEmail),
		"password": []byte("some-password"),
		"envId":    []byte(s.aw.EnvID.String()),
		"token":    []byte(token),
	}, nil)

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth-wall/login",
		requestBody,
		map[string]string{
			"Content-Type": contentType,
			"Referer":      "http://example.org",
		},
	)

	s.Equal(http.StatusFound, response.Code)
	s.Equal("http://example.org?stormkit_error=invalid_credentials", response.Header().Get("Location"))

	logins, err := authwall.Store().Logins(context.Background(), s.aw.EnvID)
	s.NoError(err)
	s.Len(logins, 1)
	s.Equal(s.aw.LoginEmail, logins[0].LoginEmail)
	s.False(logins[0].LastLogin.Valid)
}

func TestHandlerAuthSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthSuite{})
}
