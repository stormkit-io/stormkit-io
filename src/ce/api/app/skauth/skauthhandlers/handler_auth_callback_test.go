package skauthhandlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type HandlerAuthCallbackSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	app        *factory.MockApp
	mockClient *mocks.Client
	exchangeFn func(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error)
}

func (s *HandlerAuthCallbackSuite) BeforeTest(suiteName, _ string) {
	s.exchangeFn = skauthhandlers.Exchange
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockClient = &mocks.Client{}
	skauthhandlers.DefaultClient = s.mockClient

	// Create test user, app, and environment
	s.app = s.MockApp(s.MockUser(nil), nil)
}

func (s *HandlerAuthCallbackSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	skauthhandlers.DefaultClient = nil
	skauthhandlers.Exchange = s.exchangeFn
}

func (s *HandlerAuthCallbackSuite) generateStateToken(envID types.ID, provider string) string {
	token, err := user.JWT(jwt.MapClaims{
		"eid": fmt.Sprintf("%d", envID),
		"prv": provider,
	})
	s.NoError(err)
	return token
}

func (s *HandlerAuthCallbackSuite) Test_Success() {
	secret := "test-secret-key-for-jwt"
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.AuthConf{
			Secret: secret,
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
	err = skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:  env.ID,
		AppID:  s.app.ID,
		Status: true,
		Client: skauth.NewGoogleClient("test-client-id", "test-client-secret"),
	})

	s.NoError(err)

	skauthhandlers.Exchange = func(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error) {
		return tkn, nil
	}

	s.mockClient.On("UserInfo", mock.Anything, tkn).Return(&skauth.UserInfo{
		AccountID: "test-account-id",
		Email:     "test@stormkit.io",
		FirstName: "Jane",
		LastName:  "Doe",
		Avatar:    "link-to-avatar",
	}, nil)

	state := s.generateStateToken(env.ID, skauth.ProviderGoogle)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth/v1/callback?state=%s&code=test-code", state),
		nil,
		nil,
	)

	// Test the response
	body := struct {
		Token string `json:"token"`
	}{}

	s.NoError(json.Unmarshal(response.Byte(), &body))
	s.Equal(http.StatusOK, response.Code)
	s.True(strings.HasPrefix(body.Token, fmt.Sprintf("%s:", env.ID)))
	s.mockClient.AssertExpectations(s.T())

	// Parse the JWT token and grab user id
	pieces := strings.SplitN(body.Token, ":", 2)
	s.Len(pieces, 2)

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: pieces[1],
		Secret: secret,
	})

	userID := utils.StringToID(claims["uid"].(string))
	s.NotZero(userID, "User ID should be set in JWT token")
	s.Equal(skauth.ProviderGoogle, claims["prv"])

	// Verify that the user is created in the database
	usr, err := store.AuthUser(ctx, userID)
	s.NoError(err)
	s.NotNil(usr, "User should be created in the database")
	s.Equal("test@stormkit.io", usr.Email)
	s.Equal("Jane", usr.FirstName)
	s.Equal("Doe", usr.LastName)
	s.Equal("link-to-avatar", usr.Avatar)
}

func (s *HandlerAuthCallbackSuite) Test_Provider_EmptyConfig() {
	secret := "test-secret-key-for-jwt"
	env := s.MockEnv(s.app, map[string]any{
		"AuthConf": &buildconf.AuthConf{
			Secret: secret,
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

	mockClient := &mocks.Client{}
	mockClient.On("Data").Once().Return(skauth.ProviderData{}) // used in SaveProvider method
	mockClient.On("Name").Once().Return(skauth.ProviderGoogle)
	mockClient.On("Config").Once().Return(nil)

	ctx := context.Background()
	err = skauth.NewStore().SaveProvider(ctx, skauth.SaveProviderArgs{
		EnvID:  env.ID,
		AppID:  s.app.ID,
		Status: true,
		Client: mockClient,
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth/v1/callback?state=%s&code=test-code", s.generateStateToken(env.ID, skauth.ProviderGoogle)),
		nil,
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Provider is not an OAuth2 provider"}`, response.String())
}

func (s *HandlerAuthCallbackSuite) Test_InvalidStateToken() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/v1/callback?state=invalid-jwt-token&code=test-code",
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
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth/v1/callback?state=%s&code=test-code", state),
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerAuthCallbackSuite) Test_EnvironmentWithoutSchemaConf() {
	// Create environment without schema configuration
	envWithoutSchema := s.MockEnv(s.app, map[string]any{"Name": "no-schema-env"})
	state := s.generateStateToken(envWithoutSchema.ID, skauth.ProviderGoogle)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(skauthhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth/v1/callback?state=%s&code=test-code", state),
		nil,
		nil,
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAuthCallback(t *testing.T) {
	suite.Run(t, &HandlerAuthCallbackSuite{})
}
