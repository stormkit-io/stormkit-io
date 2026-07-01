package skauthhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthUsersSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
}

func (s *HandlerAuthUsersSuite) BeforeTest(suiteName, _ string) {
	// Auth table writes bypass the test transaction; truncate before each test.
	truncateAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
}

func (s *HandlerAuthUsersSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthUsersSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler()
}

// insertAuthUser seeds the schema store with a test auth user and returns the created user.
func (s *HandlerAuthUsersSuite) insertAuthUser(schemaConf *buildconf.SchemaConf, email string) *skauth.User {
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

// setupEnv creates an environment with SkAuth and SchemaConf configured, creates the auth table,
// and returns both the env and its schema config for seeding test data.
func (s *HandlerAuthUsersSuite) setupEnv() (*factory.MockEnv, *buildconf.SchemaConf) {
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

func (s *HandlerAuthUsersSuite) auth() map[string]string {
	return map[string]string{"Authorization": usertest.Authorization(s.usr.ID)}
}

func (s *HandlerAuthUsersSuite) request(method, target string, body any, headers map[string]string) shttptest.Response {
	return shttptest.RequestWithHeaders(s.handler(), method, target, body, headers)
}

// --- List ---

func (s *HandlerAuthUsersSuite) Test_List_Empty() {
	env, _ := s.setupEnv()

	response := s.request(shttp.MethodGet, fmt.Sprintf("/skauth/users?envId=%d", env.ID), nil, s.auth())

	s.Require().Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Equal([]any{}, body["results"])
	s.Equal(false, body["hasNextPage"])
}

func (s *HandlerAuthUsersSuite) Test_List_WithUsers() {
	env, schemaConf := s.setupEnv()

	for i := 0; i < 3; i++ {
		s.insertAuthUser(schemaConf, fmt.Sprintf("user%d@example.com", i))
	}

	response := s.request(shttp.MethodGet, fmt.Sprintf("/skauth/users?envId=%d", env.ID), nil, s.auth())

	s.Require().Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Len(body["results"], 3)
	s.Equal(false, body["hasNextPage"])
}

func (s *HandlerAuthUsersSuite) Test_List_Unauthorized() {
	env, _ := s.setupEnv()

	response := s.request(shttp.MethodGet, fmt.Sprintf("/skauth/users?envId=%d", env.ID), nil, nil)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_List_NotFound_NoAuthConf() {
	env := s.MockEnv(s.app, nil)

	response := s.request(shttp.MethodGet, fmt.Sprintf("/skauth/users?envId=%d", env.ID), nil, s.auth())

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_List_BadRequest_InvalidFrom() {
	env, _ := s.setupEnv()

	response := s.request(shttp.MethodGet, fmt.Sprintf("/skauth/users?envId=%d&from=abc", env.ID), nil, s.auth())

	s.Equal(http.StatusBadRequest, response.Code)
}

// --- Update ---

func (s *HandlerAuthUsersSuite) Test_Update_Success() {
	env, schemaConf := s.setupEnv()
	usr := s.insertAuthUser(schemaConf, "old@example.com")

	response := s.request(shttp.MethodPut, "/skauth/users/"+usr.UUID, map[string]any{
		"envId":     env.ID,
		"email":     "new@example.com",
		"firstName": "Jane",
		"lastName":  "Doe",
	}, s.auth())

	s.Require().Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Equal("new@example.com", body["email"])
	s.Equal("Jane", body["firstName"])
	s.Equal("Doe", body["lastName"])
	s.Equal(usr.UUID, body["id"])
}

func (s *HandlerAuthUsersSuite) Test_Update_InvalidEmail() {
	env, schemaConf := s.setupEnv()
	usr := s.insertAuthUser(schemaConf, "old@example.com")

	response := s.request(shttp.MethodPut, "/skauth/users/"+usr.UUID, map[string]any{
		"envId": env.ID,
		"email": "not-an-email",
	}, s.auth())

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_Update_DuplicateEmail() {
	env, schemaConf := s.setupEnv()
	s.insertAuthUser(schemaConf, "taken@example.com")
	usr := s.insertAuthUser(schemaConf, "mine@example.com")

	response := s.request(shttp.MethodPut, "/skauth/users/"+usr.UUID, map[string]any{
		"envId": env.ID,
		"email": "taken@example.com",
	}, s.auth())

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_Update_NotFound() {
	env, _ := s.setupEnv()

	response := s.request(shttp.MethodPut, "/skauth/users/00000000-0000-0000-0000-000000000000",
		map[string]any{"envId": env.ID, "email": "new@example.com"}, s.auth())

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_Update_Unauthorized() {
	env, schemaConf := s.setupEnv()
	usr := s.insertAuthUser(schemaConf, "old@example.com")

	response := s.request(shttp.MethodPut, "/skauth/users/"+usr.UUID,
		map[string]any{"envId": env.ID, "email": "new@example.com"}, nil)

	s.Equal(http.StatusUnauthorized, response.Code)
}

// --- Delete ---

func (s *HandlerAuthUsersSuite) Test_Delete_Success() {
	env, schemaConf := s.setupEnv()
	usr := s.insertAuthUser(schemaConf, "gone@example.com")

	response := s.request(shttp.MethodDelete,
		fmt.Sprintf("/skauth/users/%s?envId=%d", usr.UUID, env.ID), nil, s.auth())

	s.Require().Equal(http.StatusOK, response.Code)

	store, err := schemaConf.Store(buildconf.SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	found, err := store.AuthUser(context.Background(), usr.ID)
	s.Require().NoError(err)
	s.Nil(found)
}

func (s *HandlerAuthUsersSuite) Test_Delete_NotFound() {
	env, _ := s.setupEnv()

	response := s.request(shttp.MethodDelete,
		fmt.Sprintf("/skauth/users/00000000-0000-0000-0000-000000000000?envId=%d", env.ID), nil, s.auth())

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_Delete_Unauthorized() {
	env, schemaConf := s.setupEnv()
	usr := s.insertAuthUser(schemaConf, "gone@example.com")

	response := s.request(shttp.MethodDelete,
		fmt.Sprintf("/skauth/users/%s?envId=%d", usr.UUID, env.ID), nil, nil)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerAuthUsersSuite) Test_UpdateLastLogin_StampsTime() {
	_, schemaConf := s.setupEnv()
	usr := s.insertAuthUser(schemaConf, "login@example.com")

	store, err := schemaConf.Store(buildconf.SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	before, err := store.AuthUser(context.Background(), usr.ID)
	s.Require().NoError(err)
	s.True(before.LastLoginAt.IsZero(), "last login should be unset before first login")

	s.Require().NoError(store.UpdateLastLogin(context.Background(), usr.ID))

	after, err := store.AuthUser(context.Background(), usr.ID)
	s.Require().NoError(err)
	s.False(after.LastLoginAt.IsZero(), "last login should be stamped after UpdateLastLogin")
}

func TestHandlerAuthUsersSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthUsersSuite{})
}
