package publicapiv1_test

import (
	"fmt"
	"net/http"
	"testing"

	null "gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlerDeploymentRestartSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *HandlerDeploymentRestartSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *HandlerDeploymentRestartSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	deployservice.MockDeployer = nil
}

func (s *HandlerDeploymentRestartSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerDeploymentRestartSuite) restart(keyValue string, deplID fmt.Stringer) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/restart", deplID),
		nil,
		map[string]string{"Authorization": keyValue},
	)
}

// Test_BadRequest_NotFailed verifies that restarting a non-failed deployment returns 400.
func (s *HandlerDeploymentRestartSuite) Test_BadRequest_NotFailed() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	// ExitCode = 0 means success, not failed
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(0)})
	key := s.MockAPIKey(appl, env)

	response := s.restart(key.Value, depl.ID)

	s.Equal(http.StatusBadRequest, response.Code)

	body := response.Map()
	s.Equal([]any{"Only failed deployments can be restarted"}, body["errors"])
}

// Test_BadRequest_RunningDeployment verifies that a running (no exit code) deployment returns 400.
func (s *HandlerDeploymentRestartSuite) Test_BadRequest_RunningDeployment() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	// No ExitCode = still running
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	response := s.restart(key.Value, depl.ID)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_NotFound_UnknownID verifies that a non-existent deployment returns 404.
func (s *HandlerDeploymentRestartSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := s.restart(key.Value, types.ID(999999999))

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment belonging to a different env returns 404.
func (s *HandlerDeploymentRestartSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})
	depl := s.MockDeployment(env1, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.MockAPIKey(appl, env2)

	response := s.restart(key.Value, depl.ID)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_NoAPIKey verifies that requests without an API key are rejected.
func (s *HandlerDeploymentRestartSuite) Test_Forbidden_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(1)})

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/restart", depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key with no membership is rejected.
func (s *HandlerDeploymentRestartSuite) Test_Forbidden_UserNotMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(1)})

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
		fmt.Sprintf("/v1/deployments/%s/restart", depl.ID),
		map[string]any{"envId": env.ID.String()},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerDeploymentRestartSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.MockAPIKey(appl, env)

	response := s.restart(key.Value, depl.ID)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Equal(true, body["ok"])
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandlerDeploymentRestart(t *testing.T) {
	suite.Run(t, &HandlerDeploymentRestartSuite{})
}
