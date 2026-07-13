package adminhandlers_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlerGitHubManifestCallbackSuite struct {
	suite.Suite
	*factory.Factory

	conn        databasetest.TestDB
	mockRequest *mocks.RequestInterface
}

func (s *HandlerGitHubManifestCallbackSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest

	// Reset configuration before each test
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))
}

func (s *HandlerGitHubManifestCallbackSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	shttp.DefaultRequest = nil
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_MissingCode() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/github/callback?state=test_state",
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Missing code parameter"}`, response.String())
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_MissingState() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/github/callback?code=test_code",
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Missing state parameter"}`, response.String())
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_InvalidState() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/github/callback?code=test_code&state=invalid_state_token",
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_ValidState() {
	// Generate a valid state token
	stateToken, err := user.JWT(nil)
	s.NoError(err)

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")

	s.mockRequest.On("URL", "https://api.github.com/app-manifests/test_code/conversions", mock.Anything).Return(s.mockRequest).Once()
	s.mockRequest.On("Method", http.MethodPost).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", headers).Return(s.mockRequest).Once()
	s.mockRequest.On("Do", mock.Anything).Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"id": 12345,
				"name": "test-app",
				"client_id": "test-client-id",
				"client_secret": "test-client-secret",
				"pem": "test-pem"
			}`)),
		},
	}, nil).Once()

	response := shttptest.Request(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/github/callback?code=test_code&state="+stateToken,
		nil,
	)

	s.Equal(http.StatusFound, response.Code)
	s.Contains(response.Header().Get("Location"), "/admin/git?success=github_app_created")

	// Verify the configuration was set correctly
	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(config.AuthConfig)
	s.Equal("test-client-id", config.AuthConfig.Github.ClientID)
	s.Equal("test-client-secret", config.AuthConfig.Github.ClientSecret)
	s.Equal("test-pem", config.AuthConfig.Github.PrivateKey)
}

func TestHandlerGitHubManifestCallbackSuite(t *testing.T) {
	suite.Run(t, &HandlerGitHubManifestCallbackSuite{})
}
