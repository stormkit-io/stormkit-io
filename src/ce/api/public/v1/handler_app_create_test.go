package publicapiv1_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerAppCreateSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAppCreateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAppCreateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAppCreateSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAppCreateSuite) teamKey() (string, *factory.MockUser) {
	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"TeamID": usr.DefaultTeamID,
		"Scope":  apikey.SCOPE_TEAM,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"UserID": usr.ID,
	})
	return key.Value, usr
}

// Test_Success verifies that a team-scoped key can create an app and returns the
// new app in the response.
func (s *HandlerAppCreateSuite) Test_Success() {
	keyValue, usr := s.teamKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/app",
		map[string]any{
			"repo":     "owner/my-repo",
			"provider": "github",
		},
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	appData, ok := body["app"].(map[string]any)
	s.True(ok)
	s.Equal("github/owner/my-repo", appData["repo"])
	s.Equal(usr.DefaultTeamID.String(), appData["teamId"])
	s.NotEmpty(appData["id"])
	s.NotEmpty(appData["displayName"])
}

// Test_Success_WithDisplayName verifies that a custom displayName is persisted.
func (s *HandlerAppCreateSuite) Test_Success_WithDisplayName() {
	keyValue, _ := s.teamKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/app",
		map[string]any{
			"repo":        "owner/my-repo",
			"provider":    "gitlab",
			"displayName": "my-custom-app",
		},
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	appData := body["app"].(map[string]any)
	s.Equal("my-custom-app", appData["displayName"])
}

// Test_Success_BareApp verifies that an app can be created without a repo.
func (s *HandlerAppCreateSuite) Test_Success_BareApp() {
	keyValue, _ := s.teamKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/app",
		map[string]any{},
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	appData, ok := body["app"].(map[string]any)
	s.True(ok)
	s.Empty(appData["repo"])
	s.NotEmpty(appData["id"])
}

// Test_InvalidBody_InvalidProvider verifies that an unknown provider is rejected.
func (s *HandlerAppCreateSuite) Test_InvalidBody_InvalidProvider() {
	keyValue, _ := s.teamKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/app",
		map[string]any{
			"repo":     "owner/my-repo",
			"provider": "unknown",
		},
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_Unauthorized_NoAPIKey verifies that requests without a key are rejected with 403.
func (s *HandlerAppCreateSuite) Test_Unauthorized_NoAPIKey() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/app",
		map[string]any{
			"repo":     "owner/my-repo",
			"provider": "github",
		},
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_LowScopeKey verifies that an env-scoped key (below SCOPE_TEAM) is rejected.
func (s *HandlerAppCreateSuite) Test_Forbidden_LowScopeKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/app",
		map[string]any{
			"repo":     "owner/my-repo",
			"provider": "github",
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerAppCreate(t *testing.T) {
	suite.Run(t, &HandlerAppCreateSuite{})
}
