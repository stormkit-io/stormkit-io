package publicapiv1_test

import (
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
	"gopkg.in/guregu/null.v3"
)

type HandlerDeploymentGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeploymentGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentGetSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

// Test_Success verifies that a valid env-scoped key can retrieve a deployment by ID.
func (s *HandlerDeploymentGetSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d", depl.ID),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()

	got, ok := body["deployment"].(map[string]any)
	s.True(ok)
	s.Equal(depl.ID.String(), got["id"])
	s.Equal(depl.AppID.String(), got["appId"])
	s.Equal(depl.EnvID.String(), got["envId"])
	s.Equal(depl.Branch, got["branch"])
}

// Test_NotFound_UnknownID verifies that a non-existent deployment ID returns 404.
func (s *HandlerDeploymentGetSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/deployments/999999999",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment belonging to a different environment
// is not returned when the API key is scoped to a different env.
func (s *HandlerDeploymentGetSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})

	// deployment belongs to env1, but the key is scoped to env2
	depl := s.MockDeployment(env1)
	key := s.MockAPIKey(appl, env2)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d", depl.ID),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_NoAPIKey verifies that requests without an API key are rejected with 403.
func (s *HandlerDeploymentGetSuite) Test_Forbidden_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d", depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key whose owner is not a member
// of the environment's team is rejected with 403.
func (s *HandlerDeploymentGetSuite) Test_Forbidden_UserNotMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

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
		fmt.Sprintf("/v1/deployments/%d?envId=%d", depl.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_WithLogs_IncludesLogsAndStatusChecks verifies that ?logs=true causes the stored
// log data to be fetched from the DB and returned in the response.
func (s *HandlerDeploymentGetSuite) Test_WithLogs_IncludesLogsAndStatusChecks() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{
		"Logs": null.NewString(`[{"title":"Install dependencies","message":"npm ci output","status":true}]`, true),
	})
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d?logs=true", depl.ID),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()

	got := body["deployment"].(map[string]any)

	logs, ok := got["logs"].([]any)
	s.True(ok, "logs should be an array when ?logs=true is passed")
	s.Len(logs, 1)

	logEntry := logs[0].(map[string]any)
	s.Equal("Install dependencies", logEntry["title"])
	s.Equal("npm ci output", logEntry["message"])
}

func TestHandlerDeploymentGet(t *testing.T) {
	suite.Run(t, &HandlerDeploymentGetSuite{})
}
