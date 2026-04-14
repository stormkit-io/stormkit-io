package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthUsersListSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
}

func (s *HandlerAuthUsersListSuite) BeforeTest(suiteName, _ string) {
	// Auth table writes bypass the test transaction; truncate before each test.
	truncateAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	config.SetIsSelfHosted(true)
}

func (s *HandlerAuthUsersListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
}

func (s *HandlerAuthUsersListSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

// insertAuthUser seeds the schema store with a test auth user and returns the created user.
func (s *HandlerAuthUsersListSuite) insertAuthUser(schemaConf *buildconf.SchemaConf, email string) *skauth.User {
	store, err := schemaConf.Store(buildconf.SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	oauth := &skauth.OAuth{
		AccountID:    email,
		AccessToken:  "hash",
		TokenType:    "password",
		ProviderName: skauth.ProviderEmail,
	}
	usr := &skauth.User{Email: email}

	err = store.InsertAuthUser(context.Background(), oauth, usr)
	s.Require().NoError(err)

	return usr
}

// get sends a GET /v1/auth/users request with the given Authorization header and optional from query param.
func (s *HandlerAuthUsersListSuite) get(keyValue string, from int) shttptest.Response {
	path := "/v1/auth/users"

	if from > 0 {
		path = fmt.Sprintf("%s?from=%d", path, from)
	}

	return shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, path, nil,
		map[string]string{"Authorization": keyValue})
}

// setupEnv creates an environment with SkAuth and SchemaConf configured, creates the auth table,
// and returns both the env and a schema store for seeding test data.
func (s *HandlerAuthUsersListSuite) setupEnv() (*factory.MockEnv, *buildconf.SchemaConf) {
	schemaConf := &buildconf.SchemaConf{
		SchemaName:        s.conn.Cfg.Schema,
		DBName:            s.conn.Cfg.DBName,
		Port:              s.conn.Cfg.Port,
		Host:              s.conn.Cfg.Host,
		MigrationUserName: s.conn.Cfg.User,
		MigrationPassword: s.conn.Cfg.Password,
		AppUserName:       s.conn.Cfg.User,
		AppPassword:       s.conn.Cfg.Password,
		DriverName:        s.conn.Cfg.DriverName,
	}

	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: "test-secret",
			Status: true,
		},
		"SchemaConf": schemaConf,
	})

	store, err := schemaConf.Store(buildconf.SchemaAccessTypeMigrations)
	s.Require().NoError(err)
	s.Require().NoError(store.CreateAuthTable(context.Background()))

	return env, schemaConf
}

// envKey creates a SCOPE_ENV API key for the given environment.
func (s *HandlerAuthUsersListSuite) envKey(envID types.ID) *factory.MockAPIKey {
	return s.MockAPIKey(nil, nil, map[string]any{
		"EnvID": envID,
		"Scope": apikey.SCOPE_ENV,
	})
}

func (s *HandlerAuthUsersListSuite) Test_Success_Empty() {
	env, _ := s.setupEnv()
	key := s.envKey(env.ID)

	response := s.get(key.Value, 0)

	s.Require().Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Equal([]any{}, body["results"])
	s.Equal(false, body["hasNextPage"])
}

func (s *HandlerAuthUsersListSuite) Test_Success_WithUsers() {
	env, schemaConf := s.setupEnv()
	key := s.envKey(env.ID)

	for i := 0; i < 3; i++ {
		s.insertAuthUser(schemaConf, fmt.Sprintf("user%d@example.com", i))
	}

	response := s.get(key.Value, 0)

	s.Require().Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Len(body["results"], 3)
	s.Equal(false, body["hasNextPage"])
}

// Test_Forbidden verifies that a missing or invalid Authorization header returns 403.
func (s *HandlerAuthUsersListSuite) Test_Forbidden() {
	response := s.get("", 0)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_NotFound_NoAuthConf verifies that the endpoint returns 404 when SkAuth is not configured.
func (s *HandlerAuthUsersListSuite) Test_NotFound_NoAuthConf() {
	env := s.MockEnv(s.app, nil)
	key := s.envKey(env.ID)

	response := s.get(key.Value, 0)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_BadRequest_InvalidFrom verifies that a non-numeric "from" param returns 400.
func (s *HandlerAuthUsersListSuite) Test_BadRequest_InvalidFrom() {
	env, _ := s.setupEnv()
	key := s.envKey(env.ID)

	response := shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, "/v1/auth/users?from=abc", nil,
		map[string]string{"Authorization": key.Value})

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerAuthUsersListSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthUsersListSuite{})
}
