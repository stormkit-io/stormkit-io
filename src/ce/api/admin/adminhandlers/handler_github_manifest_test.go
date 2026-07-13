package adminhandlers_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerGitHubManifestSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerGitHubManifestSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerGitHubManifestSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerGitHubManifestSuite) Test_GenerateManifest_Success() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/github/manifest",
		map[string]any{
			"appName":      "test-app",
			"organization": "test-org",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	expected := `{
		"manifest": {
			"default_events": ["push", "pull_request"],
			"default_permissions": {
				"administration": "write",
				"checks": "write",
				"contents": "read",
				"emails": "read",
				"pull_requests": "write",
				"repository_hooks": "read",
				"statuses": "write"
			},
			"hook_attributes": {
				"active": true,
				"url": "http://api.stormkit:8888/app/webhooks/github/deploy"
			},
			"name": "test-app",
			"public": false,
			"callback_urls": ["http://api.stormkit:8888/auth/github/callback"],
			"redirect_url": "http://api.stormkit:8888/admin/git/github/callback",
			"setup_url": "http://api.stormkit:8888/auth/github/installation",
			"url": "http://api.stormkit:8888"
		},
		"url": "https://github.com/organizations/test-org/settings/apps/new?state=some-token"
	}`

	s.JSONEq(expected, regexp.MustCompile(`state=[^"]+`).ReplaceAllString(response.String(), `state=some-token`))
	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerGitHubManifestSuite) Test_GenerateManifest_PersonalApp() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/github/manifest",
		map[string]any{
			"appName": "test-app",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	expected := `{
		"manifest": {
			"default_events": ["push", "pull_request"],
			"default_permissions": {
				"administration": "write",
				"checks": "write",
				"contents": "read",
				"emails": "read",
				"pull_requests": "write",
				"repository_hooks": "read",
				"statuses": "write"
			},
			"hook_attributes": {
				"active": true,
				"url": "http://api.stormkit:8888/app/webhooks/github/deploy"
			},
			"name": "test-app",
			"public": false,
			"callback_urls": ["http://api.stormkit:8888/auth/github/callback"],
			"redirect_url": "http://api.stormkit:8888/admin/git/github/callback",
			"setup_url": "http://api.stormkit:8888/auth/github/installation",
			"url": "http://api.stormkit:8888"
		},
		"url": "https://github.com/settings/apps/new?state=some-token"
	}`

	s.JSONEq(expected, regexp.MustCompile(`state=[^"]+`).ReplaceAllString(response.String(), `state=some-token`))
	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerGitHubManifestSuite) Test_GenerateManifest_MissingAppName() {
	adminUser := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/github/manifest",
		map[string]any{
			"organization": "test-org",
		},
		map[string]string{
			"Authorization": usertest.Authorization(adminUser.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"App name is required"}`, response.String())
}

func (s *HandlerGitHubManifestSuite) Test_NonAdmin() {
	nonAdmin := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/git/github/manifest",
		map[string]any{
			"appName": "test-app",
		},
		map[string]string{
			"Authorization": usertest.Authorization(nonAdmin.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerGitHubManifestSuite(t *testing.T) {
	suite.Run(t, &HandlerGitHubManifestSuite{})
}
