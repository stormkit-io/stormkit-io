package publicapiv1_test

import (
	"encoding/json"
	"fmt"
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

type HandlerEnvListSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvListSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerEnvListSuite) get(keyValue string) shttptest.Response {
	return shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, "/v1/envs", nil,
		map[string]string{"Authorization": keyValue})
}

// Test_Success verifies that an app-scoped key returns all environments for the app.
func (s *HandlerEnvListSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	s.MockEnv(appl, map[string]any{"Name": "production"})
	s.MockEnv(appl, map[string]any{"Name": "staging"})
	key := s.MockAPIKey(appl, nil)

	response := s.get(key.Value)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.Require().NoError(json.Unmarshal([]byte(response.String()), &body))

	envs, ok := body["environments"].([]any)
	s.True(ok)
	s.Len(envs, 2)
}

// Test_IsolatedByApp verifies that environments from other apps are not returned.
func (s *HandlerEnvListSuite) Test_IsolatedByApp() {
	usr := s.MockUser()
	appl1 := s.MockApp(usr)
	appl2 := s.MockApp(usr)
	s.MockEnv(appl2, map[string]any{"Name": "other-app-env"})
	key := s.MockAPIKey(appl1, nil)

	response := s.get(key.Value)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.Require().NoError(json.Unmarshal([]byte(response.String()), &body))

	envs, ok := body["environments"].([]any)
	s.Require().True(ok)
	s.Empty(envs)
}

// Test_Forbidden_NoAPIKey verifies that requests without a key are rejected.
func (s *HandlerEnvListSuite) Test_Forbidden_NoAPIKey() {
	response := shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, "/v1/envs", nil, map[string]string{})
	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key with no team membership is rejected.
func (s *HandlerEnvListSuite) Test_Forbidden_UserNotMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)

	usr2 := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr2.ID,
		"Scope":  apikey.SCOPE_USER,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
	})

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/envs?appId=%s", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	// The app can be resolved from the appId query parameter; this is forbidden
	// because the user-scoped key belongs to a user who is not a member of the app's team.
	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerEnvList(t *testing.T) {
	suite.Run(t, &HandlerEnvListSuite{})
}
