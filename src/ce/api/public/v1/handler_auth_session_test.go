package publicapiv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerSessionSuite struct {
	suite.Suite
	*factory.Factory
	conn     databasetest.TestDB
	usr      *factory.MockUser
	app      *factory.MockApp
	env      *factory.MockEnv
	secret   string
	authUser func(ctx context.Context, env *buildconf.Env, userID types.ID) (*skauth.User, error)
}

func (s *HandlerSessionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.authUser = skauthhandlers.AuthUser

	// Create test user, app, and environment with AuthConf
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.secret = "test-secret-key-for-jwt"

	s.env = s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.AuthConf{
			Secret: s.secret,
		},
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName:        s.conn.Cfg.Schema,
			DBName:            s.conn.Cfg.DBName,
			Port:              s.conn.Cfg.Port,
			Host:              s.conn.Cfg.Host,
			MigrationUserName: s.conn.Cfg.User,
			MigrationPassword: s.conn.Cfg.Password,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
		},
	})

	config.SetIsSelfHosted(true)
}

func (s *HandlerSessionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	skauthhandlers.AuthUser = s.authUser
	config.SetIsSelfHosted(false)
}

func (s *HandlerSessionSuite) generateJWT(userID, envID types.ID, secret string) string {
	sessionToken, err := user.JWT(jwt.MapClaims{
		"uid": userID,
	}, secret)

	s.NoError(err)
	return fmt.Sprintf("%s:%s", envID, sessionToken)
}

func (s *HandlerSessionSuite) Test_Success() {
	user := &skauth.User{
		ID:        1,
		FirstName: "Test",
		LastName:  "User",
		Email:     "my-email@test.com",
	}

	skauthhandlers.AuthUser = func(ctx context.Context, env *buildconf.Env, authID types.ID) (*skauth.User, error) {
		return user, nil
	}

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", s.generateJWT(user.ID, s.env.ID, s.secret)),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	res := struct {
		User *skauth.User `json:"user"`
	}{}

	s.NoError(json.Unmarshal(response.Byte(), &res))
	s.Equal(user.ID, res.User.ID)
	s.Equal(user.FirstName, res.User.FirstName)
	s.Equal(user.LastName, res.User.LastName)
	s.Equal(user.Email, res.User.Email)
}

func (s *HandlerSessionSuite) Test_InvalidBearerFormat() {
	// Missing colon separator
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{
			"Authorization": "Bearer invalid-token-without-colon",
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Invalid Bearer token"}`, response.String())
}

func (s *HandlerSessionSuite) Test_EnvironmentNotFound() {
	// Generate JWT for non-existent environment
	bearer := s.generateJWT(s.usr.ID, 9999999, s.secret)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", bearer),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerSessionSuite) Test_EnvironmentWithoutAuthConf() {
	// Create environment without AuthConf
	env := s.MockEnv(s.app, map[string]any{"Name": "no-auth-env"})
	tkn := s.generateJWT(s.usr.ID, env.ID, s.secret)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", tkn),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerSessionSuite) Test_EnvironmentWithoutSchemaConf() {
	// Create environment without SchemaConf
	env := s.MockEnv(s.app, map[string]any{"Name": "no-schema-env", "AuthConf": s.env.AuthConf})
	tkn := s.generateJWT(s.usr.ID, env.ID, s.secret)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", tkn),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerSessionSuite) Test_InvalidJWTSignature() {
	// Generate JWT with wrong secret
	bearer := s.generateJWT(s.usr.ID, s.env.ID, "wrong-secret-key")

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", bearer),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerSessionSuite) Test_MissingAuthorizationHeader() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/session",
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerSession(t *testing.T) {
	suite.Run(t, &HandlerSessionSuite{})
}
