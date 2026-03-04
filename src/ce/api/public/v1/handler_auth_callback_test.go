package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type HandlerAuthCallbackSuite struct {
	suite.Suite
	*factory.Factory
	conn             databasetest.TestDB
	app              *factory.MockApp
	mockClient       *mocks.Client
	defaultProviders []string
}

func (s *HandlerAuthCallbackSuite) BeforeTest(suiteName, _ string) {
	s.mockClient = &mocks.Client{}
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser(nil), nil)
	s.defaultProviders = skauth.Providers
	skauth.DefaultClient = s.mockClient
	config.SetIsSelfHosted(true)
}

func (s *HandlerAuthCallbackSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	skauth.DefaultClient = nil
	skauth.Providers = s.defaultProviders
	config.SetIsSelfHosted(false)
}

func (s *HandlerAuthCallbackSuite) generateStateToken(envID types.ID, provider string) string {
	token, err := user.JWT(jwt.MapClaims{
		"eid": fmt.Sprintf("%d", envID),
		"prv": provider,
		"ref": "http://localhost:3000/login",
	})
	s.NoError(err)
	return token
}

func (s *HandlerAuthCallbackSuite) Test_Success() {
	secret := "test-secret-key-for-jwt"
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: secret,
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
		},
	})

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)
	s.NoError(err)
	s.NoError(store.CreateAuthTable(context.Background()))

	ctx := context.Background()
	tkn := &oauth2.Token{}
	prv := &skauth.Provider{
		Name: skauth.ProviderGoogle,
		Data: skauth.ProviderData{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
		Status: true,
	}

	err = skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:    env.ID,
		AppID:    s.app.ID,
		Provider: prv,
	})

	s.NoError(err)

	s.mockClient.On("Exchange", mock.Anything, mock.MatchedBy(func(req *shttp.RequestContext) bool {
		return req.FormValue("code") == "test-code"
	})).Return(tkn, nil).Once()

	s.mockClient.On("UserInfo", mock.Anything, tkn).Return(&skauth.UserInfo{
		AccountID: "test-account-id",
		Email:     "test@stormkit.io",
		FirstName: "Jane",
		LastName:  "Doe",
		Avatar:    "link-to-avatar",
	}, nil)

	state := s.generateStateToken(env.ID, skauth.ProviderGoogle)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth/callback?state=%s&code=test-code", state),
		nil,
		nil,
	)

	s.Equal(http.StatusFound, response.Code)
	s.Contains(response.Header().Get("Location"), "http://localhost:3000/_stormkit/auth?code=")
}

func (s *HandlerAuthCallbackSuite) Test_AuthNotEnabled() {
	env := s.MockEnv(s.app, nil) // Default is not enabled
	state := s.generateStateToken(env.ID, skauth.ProviderGoogle)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth/callback?state=%s&code=test-code", state),
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Stormkit Auth is not enabled for this environment"}`, response.String())
}

func (s *HandlerAuthCallbackSuite) Test_Provider_EmptyConfig() {
	skauth.DefaultClient = nil
	skauth.Providers = []string{"invalid-provider"} // This should pass the first check, but fail when trying to get the provider configuration
	secret := "test-secret-key-for-jwt"
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.SKAuthConf{
			Secret: secret,
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
		},
	})

	ctx := context.Background()
	prv := &skauth.Provider{
		Name:   "invalid-provider",
		Data:   skauth.ProviderData{},
		Status: true,
	}

	err := skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:    env.ID,
		AppID:    s.app.ID,
		Provider: prv,
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth/callback?state=%s&code=test-code", s.generateStateToken(env.ID, "invalid-provider")),
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Provider is not an OAuth2 provider"}`, response.String())
}

func (s *HandlerAuthCallbackSuite) Test_InvalidStateToken() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/auth/callback?state=invalid-jwt-token&code=test-code",
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"invalid state parameter"}`, response.String())
}

func (s *HandlerAuthCallbackSuite) Test_EnvironmentNotFound() {
	// Generate state token with non-existent environment
	state := s.generateStateToken(1, skauth.ProviderGoogle)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/auth/callback?state=%s&code=test-code", state),
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAuthCallback(t *testing.T) {
	suite.Run(t, &HandlerAuthCallbackSuite{})
}
