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

type HandlerAuthUpsertSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	usr        *factory.MockUser
	app        *factory.MockApp
	env        *factory.MockEnv
	schemaName string
}

func (s *HandlerAuthUpsertSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user, app, and environment
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.env = s.MockEnv(s.app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})
}

func (s *HandlerAuthUpsertSuite) AfterTest(_, _ string) {
	// Clean up schema
	if s.schemaName != "" {
		_ = buildconf.SchemaStore().DropSchema(context.Background(), s.schemaName)
	}

	s.conn.CloseTx()
}

func (s *HandlerAuthUpsertSuite) Test_Success() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/skauth",
		map[string]any{
			"envId":        s.env.ID,
			"providerName": "google",
			"clientId":     "test-client-id",
			"clientSecret": "test",
			"status":       true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	provider, err := skauth.NewStore().Provider(context.Background(), s.env.ID, skauth.ProviderGoogle)
	s.NoError(err)
	s.NotNil(provider, "Provider should be saved")
	s.True(provider.Status)

	s.Equal(skauth.ProviderData{
		ClientID:     "test-client-id",
		ClientSecret: "test",
		RedirectURL:  "http://api.stormkit:8888/auth/v1/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}, provider.Data)
}

func (s *HandlerAuthUpsertSuite) Test_Update() {
	err := skauth.NewStore().SaveProvider(context.Background(), skauth.SaveProviderArgs{
		EnvID:  s.env.ID,
		AppID:  s.app.ID,
		Status: true,
		Client: skauth.NewGoogleClient("my-client-id", "my-client-secret"),
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/skauth",
		map[string]any{
			"envId":        s.env.ID,
			"providerName": "google",
			"clientId":     "test-client-id",
			"status":       true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	provider, err := skauth.NewStore().Provider(context.Background(), s.env.ID, skauth.ProviderGoogle)
	s.NoError(err)
	s.NotNil(provider, "Provider should be saved")
	s.True(provider.Status)
	s.Equal("google", provider.Name)
	s.Equal(skauth.ProviderData{
		ClientID:     "test-client-id",
		ClientSecret: "my-client-secret",
		RedirectURL:  "http://api.stormkit:8888/auth/v1/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}, provider.Data)
}

func (s *HandlerAuthUpsertSuite) Test_InvalidRequests() {
	type payload = map[string]any

	envWithoutSchema := s.MockEnv(s.app, map[string]any{
		"Name": "env_without_schema",
	})

	payloads := map[string]payload{
		"Schema configuration is not set for this environment. Please configure it first.": {
			"envId": envWithoutSchema.ID,
		},
		"Client ID is required": {
			"envId":        s.env.ID,
			"providerName": "google",
			"clientSecret": "test",
			"status":       true,
		},
		"Client Secret is required": {
			"envId":        s.env.ID,
			"providerName": "google",
			"clientId":     "test-client-id",
			"status":       true,
		},
		"Invalid provider": {
			"envId":        s.env.ID,
			"providerName": "invalid-provider",
			"clientId":     "test-client-id",
			"clientSecret": "test",
		},
	}

	for msg, payload := range payloads {
		{
			response := shttptest.RequestWithHeaders(
				shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
				shttp.MethodPost,
				"/skauth",
				payload,
				map[string]string{
					"Authorization": usertest.Authorization(s.usr.ID),
				},
			)

			s.Equal(http.StatusBadRequest, response.Code)
			s.JSONEq(fmt.Sprintf(`{ "error": "%s" }`, msg), response.String())
		}
	}
}

func (s *HandlerAuthUpsertSuite) Test_Idempotent() {
	for range 2 {
		response := shttptest.RequestWithHeaders(
			shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
			shttp.MethodPost,
			"/skauth",
			map[string]any{
				"envId":        s.env.ID,
				"providerName": "google",
				"clientId":     "my-client-id",
				"clientSecret": "my-secret",
				"status":       true,
			},
			map[string]string{
				"Authorization": usertest.Authorization(s.usr.ID),
			},
		)

		s.Equal(http.StatusOK, response.Code)
	}

	provider, err := skauth.NewStore().Provider(context.Background(), s.env.ID, skauth.ProviderGoogle)
	s.NoError(err)
	s.NotNil(provider, "Provider should be saved")
	s.True(provider.Status)
	s.Equal("google", provider.Name)
	s.Equal(skauth.ProviderData{
		ClientID:     "my-client-id",
		ClientSecret: "my-secret",
		RedirectURL:  "http://api.stormkit:8888/auth/v1/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}, provider.Data)
}

func TestHandlerUpsertSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthUpsertSuite{})
}
