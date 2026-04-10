package publicapiv1_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	null "gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerDeploymentStopSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentStopSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeploymentStopSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentStopSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerDeploymentStopSuite) stop(keyValue string, deplID fmt.Stringer) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/stop", deplID),
		nil,
		map[string]string{"Authorization": keyValue},
	)
}

// Test_Success verifies that a running deployment can be stopped.
func (s *HandlerDeploymentStopSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	// ExitCode not set = deployment is still running
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	response := s.stop(key.Value, depl.ID)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.Require().NoError(json.Unmarshal([]byte(response.String()), &body))
	s.Equal(true, body["ok"])
}

// Test_Success_AlreadyStopped verifies that stopping an already-finished deployment
// is a no-op and still returns 200.
func (s *HandlerDeploymentStopSuite) Test_Success_AlreadyStopped() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(0)})
	key := s.MockAPIKey(appl, env)

	response := s.stop(key.Value, depl.ID)

	s.Equal(http.StatusOK, response.Code)
}

// Test_NotFound_UnknownID verifies that a non-existent deployment returns 404.
func (s *HandlerDeploymentStopSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := s.stop(key.Value, types.ID(999999999))

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment belonging to a different env returns 404.
func (s *HandlerDeploymentStopSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})
	depl := s.MockDeployment(env1)
	key := s.MockAPIKey(appl, env2)

	response := s.stop(key.Value, depl.ID)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_NoAPIKey verifies that requests without an API key are rejected.
func (s *HandlerDeploymentStopSuite) Test_Forbidden_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/stop", depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key with no membership is rejected.
func (s *HandlerDeploymentStopSuite) Test_Forbidden_UserNotMember() {
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
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/stop", depl.ID),
		map[string]any{"envId": env.ID.String()},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerDeploymentStop(t *testing.T) {
	suite.Run(t, &HandlerDeploymentStopSuite{})
}
