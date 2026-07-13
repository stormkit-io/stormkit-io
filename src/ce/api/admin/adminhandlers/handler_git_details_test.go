package adminhandlers_test

import (
	"context"
	"net/http"
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

type HandlerGitDetailsSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerGitDetailsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Reset configuration before each test
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))
}

func (s *HandlerGitDetailsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_NoConfiguration() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_EmptyAuthConfig() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// The BeforeTest sets up an empty AuthConfig, which results in an empty response
	// since no providers are enabled

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	// When AuthConfig exists but no providers are enabled, returns empty object
	s.JSONEq(`{}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_GitHubConfigured() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Set up GitHub configuration
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			Account:      "test-account",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			PrivateKey:   "test-private-key",
			AppID:        12345,
			RunnerRepo:   "test-runner-repo",
			RunnerToken:  "test-runner-token",
		},
	}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"github": {
			"account": "test-account",
			"appId": "12345",
			"clientId": "test-client-id",
			"runnerRepo": "test-runner-repo",
			"hasRunnerToken": true,
			"hasPrivateKey": true,
			"hasClientSecret": true
		}
	}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_GitHubPartialConfiguration() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Set up partial GitHub configuration (missing runner token)
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			Account:      "test-account",
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			PrivateKey:   "test-private-key",
			AppID:        12345,
			RunnerRepo:   "test-runner-repo",
			// RunnerToken is empty
		},
	}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"github": {
			"appId": "12345",
			"account": "test-account",
			"clientId": "test-client-id",
			"runnerRepo": "test-runner-repo",
			"hasRunnerToken": false,
			"hasPrivateKey": true,
			"hasClientSecret": true
		}
	}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_GitLabConfigured() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Set up GitLab configuration
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Gitlab: admin.GitlabConfig{
			ClientID:     "gitlab-client-id",
			ClientSecret: "gitlab-client-secret",
			RedirectURL:  "https://example.com/auth/gitlab/callback",
		},
	}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"gitlab": {
			"clientId": "gitlab-client-id",
			"redirectUrl": "https://example.com/auth/gitlab/callback",
			"hasClientSecret": true
		}
	}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_BitbucketConfigured() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Set up Bitbucket configuration
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Bitbucket: admin.BitbucketConfig{
			ClientID:     "bitbucket-client-id",
			ClientSecret: "bitbucket-client-secret",
			DeployKey:    "bitbucket-deploy-key",
		},
	}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"bitbucket": {
			"clientId": "bitbucket-client-id",
			"hasDeployKey": true,
			"hasClientSecret": true
		}
	}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_BitbucketNoDeployKey() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Set up Bitbucket configuration without deploy key
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Bitbucket: admin.BitbucketConfig{
			ClientID:     "bitbucket-client-id",
			ClientSecret: "bitbucket-client-secret",
			// DeployKey is empty
		},
	}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"bitbucket": {
			"clientId": "bitbucket-client-id",
			"hasDeployKey": false,
			"hasClientSecret": true
		}
	}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_AllProvidersConfigured() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Set up all providers
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			Account:      "github-account",
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret",
			PrivateKey:   "github-private-key",
			AppID:        12345,
			RunnerRepo:   "github-runner-repo",
			RunnerToken:  "github-runner-token",
		},
		Gitlab: admin.GitlabConfig{
			ClientID:     "gitlab-client-id",
			ClientSecret: "gitlab-client-secret",
			RedirectURL:  "https://example.com/auth/gitlab/callback",
		},
		Bitbucket: admin.BitbucketConfig{
			ClientID:     "bitbucket-client-id",
			ClientSecret: "bitbucket-client-secret",
			DeployKey:    "bitbucket-deploy-key",
		},
	}
	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"github": {
			"account": "github-account",
			"clientId": "github-client-id",
			"runnerRepo": "github-runner-repo",
			"hasRunnerToken": true,
			"hasPrivateKey": true,
			"hasClientSecret": true,
			"appId": "12345"
		},
		"gitlab": {
			"clientId": "gitlab-client-id",
			"hasClientSecret": true,
			"redirectUrl": "https://example.com/auth/gitlab/callback"
		},
		"bitbucket": {
			"clientId": "bitbucket-client-id",
			"hasDeployKey": true,
			"hasClientSecret": true
		}
	}`, response.String())
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_NonAdmin() {
	nonAdmin := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(nonAdmin.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerGitDetailsSuite) Test_GitDetails_DatabaseError() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	// Close the transaction to simulate a database error
	s.conn.CloseTx()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/git/details",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusInternalServerError, response.Code)
}

func TestHandlerGitDetailsSuite(t *testing.T) {
	suite.Run(t, &HandlerGitDetailsSuite{})
}
