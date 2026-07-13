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
