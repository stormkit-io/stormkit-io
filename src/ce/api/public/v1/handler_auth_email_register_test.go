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

type HandlerAuthEmailRegisterSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	app  *factory.MockApp
}

func (s *HandlerAuthEmailRegisterSuite) BeforeTest(suiteName, _ string) {
	// Auth table operations in the handler open a separate real postgres connection
	// (bypassing the txdb transaction), so their data persists across tests.
	// Truncate those tables before each test to ensure a clean slate.
	s.truncateAuthTables()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	config.SetIsSelfHosted(true)
}

func (s *HandlerAuthEmailRegisterSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsSelfHosted(false)
}

func (s *HandlerAuthEmailRegisterSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAuthEmailRegisterSuite) truncateAuthTables() {
	truncateAuthTables()
}

func (s *HandlerAuthEmailRegisterSuite) post(fields map[string]string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/register",
		fields,
		nil,
	)
}

func (s *HandlerAuthEmailRegisterSuite) setupEnv(successURL string) (*factory.MockEnv, error) {
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret:     "test-secret",
			Status:     true,
			SuccessURL: successURL,
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

func (s *HandlerAuthEmailRegisterSuite) Test_Success() {
	env, err := s.setupEnv("/dashboard")
	s.Require().NoError(err)

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "jane@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusCreated, response.Code)

	token, ok := response.Map()["token"].(string)
	s.True(ok)
	s.NotEmpty(token)
}

// Test_Success_NoSuccessURL verifies that registration works regardless of whether a SuccessURL is set.
func (s *HandlerAuthEmailRegisterSuite) Test_Success_NoSuccessURL() {
	env, err := s.setupEnv("")
	s.Require().NoError(err)

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "jane@example-2.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusCreated, response.Code)

	token, ok := response.Map()["token"].(string)
	s.True(ok)
	s.NotEmpty(token)
}

// Test_DuplicateEmail verifies that registering with an existing email returns a JSON 400 error.
func (s *HandlerAuthEmailRegisterSuite) Test_DuplicateEmail() {
	env, err := s.setupEnv("")
	s.Require().NoError(err)

	fields := map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "jane-5@example.com",
		"password": "supersecret123",
	}

	s.Require().Equal(http.StatusCreated, s.post(fields).Code)

	// Second registration with same email — no Referer, so falls back to JSON error.
	response := s.post(fields)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("an account with this email already exists", response.Map()["errors"].([]any)[0])
}

func (s *HandlerAuthEmailRegisterSuite) Test_InvalidEmail() {
	response := s.post(map[string]string{
		"envId":    "1",
		"email":    "not-an-email",
		"password": "supersecret123",
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("email is invalid", response.Map()["errors"].([]any)[0])
}

func (s *HandlerAuthEmailRegisterSuite) Test_ShortPassword() {
	response := s.post(map[string]string{
		"envId":    "1",
		"email":    "jane-142@example.com",
		"password": "short",
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("password must be at least 8 characters", response.Map()["errors"].([]any)[0])
}

// Test_AuthNotEnabled verifies that registering when SkAuth is disabled returns 404.
func (s *HandlerAuthEmailRegisterSuite) Test_AuthNotEnabled() {
	env := s.MockEnv(s.app, nil)

	response := s.post(map[string]string{
		"envId":    fmt.Sprintf("%d", env.ID),
		"email":    "jane-143@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_EmailProviderNotEnabled verifies that registering when the email provider is disabled returns 404.
func (s *HandlerAuthEmailRegisterSuite) Test_EmailProviderNotEnabled() {
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
		"email":    "jane-144@example.com",
		"password": "supersecret123",
	})

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAuthEmailRegisterSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthEmailRegisterSuite{})
}
