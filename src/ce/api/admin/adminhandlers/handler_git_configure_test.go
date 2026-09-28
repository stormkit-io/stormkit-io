package adminhandlers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerGitConfigureSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerGitConfigureSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Reset
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))
}

func (s *HandlerGitConfigureSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerGitConfigureSuite) Test_ConfigureGithub_Success() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"appId":        "12345",
			"provider":     "github",
			"account":      "github-org",
			"clientId":     "my-new-client-id",
			"clientSecret": "my-new-secret",
			"privateKey":   "my-new-pem",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify the configuration was set correctly
	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(config.AuthConfig)
	s.Equal("github-org", config.AuthConfig.Github.Account)
	s.Equal("my-new-pem", config.AuthConfig.Github.PrivateKey)
	s.Equal("my-new-secret", config.AuthConfig.Github.ClientSecret)
	s.Equal("my-new-client-id", config.AuthConfig.Github.ClientID)
	s.Equal(int(12345), config.AuthConfig.Github.AppID)
}

// configureGithub posts a manual GitHub configuration. Fields in overrides
// replace the defaults, and a nil value removes the field from the request.
func (s *HandlerGitConfigureSuite) configureGithub(overrides map[string]any) shttptest.Response {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	payload := map[string]any{
		"appId":        "12345",
		"provider":     "github",
		"account":      "github-org",
		"clientId":     "my-new-client-id",
		"clientSecret": "my-new-secret",
		"privateKey":   "my-new-pem",
	}

	for key, value := range overrides {
		if value == nil {
			delete(payload, key)
		} else {
			payload[key] = value
		}
	}

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		payload,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)
}

// githubConfig returns the stored GitHub configuration.
func (s *HandlerGitConfigureSuite) githubConfig() admin.GithubConfig {
	config, err := admin.Store().Config(context.Background())
	s.Require().NoError(err)
	s.Require().NotNil(config.AuthConfig)

	return config.AuthConfig.Github
}

func (s *HandlerGitConfigureSuite) Test_ConfigureGithub_WebhookSecret() {
	s.Equal(http.StatusOK, s.configureGithub(map[string]any{"webhookSecret": "my-webhook-secret"}).Code)
	s.Equal("my-webhook-secret", s.githubConfig().WebhookSecret)
}

// Test_ConfigureGithub_KeepsWebhookSecret verifies that saving the form without
// a webhook secret does not erase the stored one.
func (s *HandlerGitConfigureSuite) Test_ConfigureGithub_KeepsWebhookSecret() {
	s.Equal(http.StatusOK, s.configureGithub(map[string]any{"webhookSecret": "my-webhook-secret"}).Code)
	s.Equal(http.StatusOK, s.configureGithub(nil).Code)
	s.Equal("my-webhook-secret", s.githubConfig().WebhookSecret)
}

// Test_ConfigureGithub_KeepsCredentials verifies that an existing GitHub App can
// be updated, for instance to add a webhook secret, without re-entering its
// client secret and private key. Whitespace-only values count as blank.
func (s *HandlerGitConfigureSuite) Test_ConfigureGithub_KeepsCredentials() {
	s.Require().Equal(http.StatusOK, s.configureGithub(nil).Code)

	response := s.configureGithub(map[string]any{
		"clientSecret":  "  ",
		"privateKey":    "\n",
		"webhookSecret": "my-webhook-secret",
	})

	s.Equal(http.StatusOK, response.Code)

	github := s.githubConfig()
	s.Equal("my-new-secret", github.ClientSecret)
	s.Equal("my-new-pem", github.PrivateKey)
	s.Equal("my-webhook-secret", github.WebhookSecret)
}

// Test_ConfigureGithub_OtherAppNeedsCredentials verifies that the secrets of
// one GitHub App are not kept when switching to another app.
func (s *HandlerGitConfigureSuite) Test_ConfigureGithub_OtherAppNeedsCredentials() {
	s.Require().Equal(http.StatusOK, s.configureGithub(map[string]any{"webhookSecret": "my-webhook-secret"}).Code)

	response := s.configureGithub(map[string]any{
		"appId":        "67890",
		"clientId":     "other-client-id",
		"clientSecret": nil,
		"privateKey":   nil,
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal("my-new-secret", s.githubConfig().ClientSecret)

	response = s.configureGithub(map[string]any{
		"appId":        "67890",
		"clientId":     "other-client-id",
		"clientSecret": "other-secret",
		"privateKey":   "other-pem",
	})

	s.Equal(http.StatusOK, response.Code)

	github := s.githubConfig()
	s.Equal("other-secret", github.ClientSecret)
	s.Equal("other-pem", github.PrivateKey)
	s.Empty(github.WebhookSecret)
}

// Test_ConfigureGithub_MissingCredentials verifies that a first-time setup
// still requires the client secret and private key.
func (s *HandlerGitConfigureSuite) Test_ConfigureGithub_MissingCredentials() {
	response := s.configureGithub(map[string]any{"clientSecret": nil, "privateKey": nil})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "GitHub App is not properly configured")

	config, err := admin.Store().Config(context.Background())
	s.Require().NoError(err)
	s.Empty(config.AuthConfig.Github.ClientID)
}

func (s *HandlerGitConfigureSuite) Test_ConfigureGitlab_Success() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "gitlab",
			"clientId":     "gitlab-client-id",
			"clientSecret": "gitlab-client-secret",
			"redirectUrl":  "https://myapp.com/auth/gitlab/callback",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify the configuration was set correctly
	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(config.AuthConfig)
	s.Equal("gitlab-client-id", config.AuthConfig.Gitlab.ClientID)
	s.Equal("gitlab-client-secret", config.AuthConfig.Gitlab.ClientSecret)
	s.Equal("https://myapp.com/auth/gitlab/callback", config.AuthConfig.Gitlab.RedirectURL)
}

func (s *HandlerGitConfigureSuite) Test_ConfigureBitbucket_Success() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "bitbucket",
			"clientId":     "bitbucket-client-id",
			"clientSecret": "bitbucket-client-secret",
			"deployKey":    "bitbucket-deploy-key",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify the configuration was set correctly
	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(config.AuthConfig)
	s.Equal("bitbucket-client-id", config.AuthConfig.Bitbucket.ClientID)
	s.Equal("bitbucket-client-secret", config.AuthConfig.Bitbucket.ClientSecret)
	s.Equal("bitbucket-deploy-key", config.AuthConfig.Bitbucket.DeployKey)
}

func (s *HandlerGitConfigureSuite) Test_UpdateExistingConfiguration() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Test updating GitLab configuration (since GitHub is TODO)
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "gitlab",
			"clientId":     "new-gitlab-client-id",
			"clientSecret": "new-gitlab-client-secret",
			"redirectUrl":  "https://updated.com/auth/gitlab/callback",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify the configuration was set
	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(config.AuthConfig)
	s.Equal("new-gitlab-client-id", config.AuthConfig.Gitlab.ClientID)
	s.Equal("new-gitlab-client-secret", config.AuthConfig.Gitlab.ClientSecret)
	s.Equal("https://updated.com/auth/gitlab/callback", config.AuthConfig.Gitlab.RedirectURL)
}

func (s *HandlerGitConfigureSuite) Test_MissingProvider() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"clientId":     "some-client-id",
			"clientSecret": "some-client-secret",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Provider is required"}`, response.String())
}

func (s *HandlerGitConfigureSuite) Test_InvalidProvider() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "invalid-provider",
			"clientId":     "some-client-id",
			"clientSecret": "some-client-secret",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Invalid provider. Must be one of: github, gitlab, bitbucket"}`, response.String())
}

func (s *HandlerGitConfigureSuite) Test_NonAdmin() {
	nonAdmin := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "github",
			"clientId":     "some-client-id",
			"clientSecret": "some-client-secret",
		},
		map[string]string{
			"Authorization": usertest.Authorization(nonAdmin.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerGitConfigureSuite) Test_InvalidJSON() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		strings.NewReader("invalid json"),
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerGitConfigureSuite) Test_ConfigureMultipleProviders() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Configure GitHub first (TODO implementation)
	response1 := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "bitbucket",
			"clientId":     "my-bitbucket-client-id",
			"clientSecret": "my-bitbucket-client-secret",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response1.Code)

	// Configure GitLab second
	response2 := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/configure",
		map[string]any{
			"provider":     "gitlab",
			"clientId":     "gitlab-client-id",
			"clientSecret": "gitlab-client-secret",
			"redirectUrl":  "https://myapp.com/auth/gitlab/callback",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response2.Code)

	config, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(config.AuthConfig)

	s.Equal("gitlab-client-id", config.AuthConfig.Gitlab.ClientID)
	s.Equal("gitlab-client-secret", config.AuthConfig.Gitlab.ClientSecret)
	s.Equal("https://myapp.com/auth/gitlab/callback", config.AuthConfig.Gitlab.RedirectURL)
	s.Equal("my-bitbucket-client-id", config.AuthConfig.Bitbucket.ClientID)
	s.False(config.IsGithubEnabled())
}

func TestHandlerGitConfigureSuite(t *testing.T) {
	suite.Run(t, &HandlerGitConfigureSuite{})
}
