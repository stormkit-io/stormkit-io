package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthEmailLoginSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
}

func (s *HandlerAuthEmailLoginSuite) BeforeTest(suiteName, _ string) {
	s.truncateAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	config.SetIsSelfHosted(true)
}

func (s *HandlerAuthEmailLoginSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
}

func (s *HandlerAuthEmailLoginSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAuthEmailLoginSuite) truncateAuthTables() {
	truncateAuthTables()
}

func (s *HandlerAuthEmailLoginSuite) post(fields map[string]string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/login",
		fields,
		nil,
	)
}

func (s *HandlerAuthEmailLoginSuite) setupEnv() (*factory.MockEnv, error) {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: "test-secret",
			Status: true,
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
			DriverName:        s.conn.Cfg.DriverName,
		},
	})

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)

	if err != nil {
		return nil, err
	}

	if err := store.CreateAuthTable(context.Background()); err != nil {
		return nil, err
	}

	err = skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID: env.ID,
		AppID: s.app.ID,
		Provider: &skauth.Provider{
			Name:   skauth.ProviderEmail,
			Status: true,
		},
	})

	return env, err
}

func (s *HandlerAuthEmailLoginSuite) register(envID any, email, password string) {
	resp := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/register",
		map[string]string{
			"envId":    fmt.Sprintf("%v", envID),
			"email":    email,
			"password": password,
		},
		nil,
	)

	s.Require().Equal(http.StatusCreated, resp.Code)
}

func (s *HandlerAuthEmailLoginSuite) Test_Success() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	s.register(env.ID, "login@example.com", "supersecret123")

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "login@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	token, ok := body["token"].(string)
	s.True(ok)
	s.NotEmpty(token)
	s.Equal("login@example.com", body["email"])
	s.NotEmpty(body["userId"])
}

func (s *HandlerAuthEmailLoginSuite) Test_WrongPassword() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	s.register(env.ID, "login2@example.com", "supersecret123")

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "login2@example.com",
		"password": "wrongpassword",
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("invalid email or password", response.Map()["errors"].([]any)[0])
}

func (s *HandlerAuthEmailLoginSuite) Test_UnknownEmail() {
	env, err := s.setupEnv()
	s.Require().NoError(err)

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "nobody@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("invalid email or password", response.Map()["errors"].([]any)[0])
}

func (s *HandlerAuthEmailLoginSuite) Test_InvalidEmail() {
	response := s.post(map[string]string{
		"envId":    "1",
		"email":    "not-an-email",
		"password": "supersecret123",
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("email is invalid", response.Map()["errors"].([]any)[0])
}

func (s *HandlerAuthEmailLoginSuite) Test_ShortPassword() {
	response := s.post(map[string]string{
		"envId":    "1",
		"email":    "login3@example.com",
		"password": "short",
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("password must be at least 8 characters", response.Map()["errors"].([]any)[0])
}

func (s *HandlerAuthEmailLoginSuite) Test_AuthNotEnabled() {
	env := s.MockEnv(s.app, nil)

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "login4@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthEmailLoginSuite) Test_EmailProviderNotEnabled() {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: "test-secret",
			Status: true,
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
			DriverName:        s.conn.Cfg.DriverName,
		},
	})

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "login5@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAuthEmailLoginSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthEmailLoginSuite{})
}
