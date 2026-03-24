package publicapiv1_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerDeploymentPollSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentPollSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeploymentPollSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentPollSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

// Test_Running verifies that a deployment with no exit code yet returns status "running".
func (s *HandlerDeploymentPollSuite) Test_Running() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env) // ExitCode is null → running
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d/poll", depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)
	body := response.Map()
	s.Equal("running", body["status"])
}

// Test_Complete_Success verifies that a deployment with exit code 0 returns status "success".
func (s *HandlerDeploymentPollSuite) Test_Complete_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	// Use a config snapshot without status checks so ExitCode 0 maps directly to "success".
	conf, err := json.Marshal(deploy.ConfigSnapshot{
		BuildConfig: &buildconf.BuildConf{BuildCmd: "npm run build", DistFolder: "build"},
	})

	s.Require().NoError(err)

	depl := s.MockDeployment(env, map[string]any{
		"ExitCode":   null.IntFrom(0),
		"ConfigCopy": conf,
	})
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d/poll", depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)
	body := response.Map()
	s.Equal("success", body["status"])
}

// Test_Complete_Failed verifies that a deployment with a non-zero exit code returns status "failed".
func (s *HandlerDeploymentPollSuite) Test_Complete_Failed() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d/poll", depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)
	body := response.Map()
	s.Equal("failed", body["status"])
}

// Test_NotFound_UnknownID verifies that a non-existent deployment ID returns 404.
func (s *HandlerDeploymentPollSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/deployments/999999999/poll",
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment belonging to a different env is not accessible.
func (s *HandlerDeploymentPollSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})
	depl := s.MockDeployment(env1)
	key := s.MockAPIKey(appl, env2) // key scoped to env2, deployment belongs to env1

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d/poll", depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Unauthorized_NoAPIKey verifies that a request with no API key returns 403.
func (s *HandlerDeploymentPollSuite) Test_Unauthorized_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%d/poll", depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerDeploymentPoll(t *testing.T) {
	suite.Run(t, &HandlerDeploymentPollSuite{})
}
