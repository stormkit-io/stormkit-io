package adminhandlers_test

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
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
	response := shttptest.Request(s.handler(), shttp.MethodGet, "/admin/git/github/callback?state=test_state", nil)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Missing code parameter"}`, response.String())
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_MissingState() {
	response := shttptest.Request(s.handler(), shttp.MethodGet, "/admin/git/github/callback?code=test_code", nil)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Missing state parameter"}`, response.String())
}

func (s *HandlerGitHubManifestCallbackSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler()
}

// startFlow starts the manifest flow as the given admin and returns the state
// that GitHub will send back to the callback.
func (s *HandlerGitHubManifestCallbackSuite) startFlow(adminUser *factory.MockUser) string {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/admin/git/github/manifest",
		map[string]any{"appName": "test-app"},
		map[string]string{"Authorization": usertest.Authorization(adminUser.ID)},
	)

	s.Require().Equal(http.StatusOK, response.Code)

	state := regexp.MustCompile(`state=([^"]+)`).FindStringSubmatch(response.String())
	s.Require().Len(state, 2)

	return state[1]
}

func (s *HandlerGitHubManifestCallbackSuite) callback(state string) shttptest.Response {
	return shttptest.Request(s.handler(), shttp.MethodGet, "/admin/git/github/callback?code=test_code&state="+state, nil)
}

// mockExchange makes GitHub return app credentials for the manifest code once.
func (s *HandlerGitHubManifestCallbackSuite) mockExchange() {
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
				"webhook_secret": "test-webhook-secret",
				"pem": "test-pem"
			}`)),
		},
	}, nil).Once()
}

// assertRejected calls the callback with the given state and asserts that the
// admin is sent back with an error, without exchanging the code with GitHub or
// changing the configuration.
func (s *HandlerGitHubManifestCallbackSuite) assertRejected(state string) {
	response := s.callback(state)

	s.Equal(http.StatusFound, response.Code)
	s.Contains(response.Header().Get("Location"), "/admin/git?error=github_app_state_invalid")
	s.mockRequest.AssertNotCalled(s.T(), "URL", mock.Anything, mock.Anything)

	config, err := admin.Store().Config(context.Background())
	s.Require().NoError(err)
	s.Empty(config.AuthConfig.Github.ClientID)
}

// Test_ManifestCallback_SignedTokens verifies that no token signed by the
// instance can complete the flow: not the states issued to anonymous visitors
// by the OAuth login routes, and not an admin's own session token.
func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_SignedTokens() {
	anonymous, err := user.JWT(user.JWTParams{Purpose: user.PurposeOAuthState, Claims: jwt.MapClaims{"provider": "github"}})
	s.Require().NoError(err)

	session, err := user.JWT(user.JWTParams{Purpose: user.PurposeSession, Claims: jwt.MapClaims{"uid": s.MockUser(map[string]any{"IsAdmin": true}).ID.String()}})
	s.Require().NoError(err)

	s.assertRejected(anonymous)
	s.assertRejected(session)
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_InvalidState() {
	s.assertRejected("invalid_state_token")
}

// Test_ManifestCallback_NoLongerAdmin verifies that a state is rejected when
// its admin lost admin rights before the flow completed.
func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_NoLongerAdmin() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})
	state := s.startFlow(adminUser)

	_, err := s.conn.Exec("UPDATE users SET is_admin = FALSE WHERE user_id = $1", adminUser.ID)
	s.Require().NoError(err)

	s.assertRejected(state)
}

func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_ValidState() {
	state := s.startFlow(s.MockUser(map[string]any{"IsAdmin": true}))
	s.mockExchange()

	response := s.callback(state)

	s.Equal(http.StatusFound, response.Code)
	s.Contains(response.Header().Get("Location"), "/admin/git?success=github_app_created")

	config, err := admin.Store().Config(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(config.AuthConfig)
	s.Equal("test-client-id", config.AuthConfig.Github.ClientID)
	s.Equal("test-client-secret", config.AuthConfig.Github.ClientSecret)
	s.Equal("test-pem", config.AuthConfig.Github.PrivateKey)
	s.Equal("test-webhook-secret", config.AuthConfig.Github.WebhookSecret)
}

// Test_ManifestCallback_Replay verifies that a state can only be used once.
func (s *HandlerGitHubManifestCallbackSuite) Test_ManifestCallback_Replay() {
	state := s.startFlow(s.MockUser(map[string]any{"IsAdmin": true}))
	s.mockExchange()

	s.Require().Equal(http.StatusFound, s.callback(state).Code)

	response := s.callback(state)

	s.Equal(http.StatusFound, response.Code)
	s.Contains(response.Header().Get("Location"), "/admin/git?error=github_app_state_invalid")
	s.mockRequest.AssertNumberOfCalls(s.T(), "Do", 1)
}

func TestHandlerGitHubManifestCallbackSuite(t *testing.T) {
	suite.Run(t, &HandlerGitHubManifestCallbackSuite{})
}
