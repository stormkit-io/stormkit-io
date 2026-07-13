package publicapiv1_test

import (
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

type HandlerDeploymentPublishSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentPublishSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeploymentPublishSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentPublishSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerDeploymentPublishSuite) publishURL(id fmt.Stringer) string {
	return fmt.Sprintf("/v1/deployments/%s/publish", id)
}

// Test_Success verifies that a valid env-scoped key can publish a deployment at 100%.
func (s *HandlerDeploymentPublishSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		s.publishURL(depl.ID),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	s.Equal(true, body["ok"])
}

// Test_PublishedStateReflected verifies that after publishing, fetching the deployment
// returns non-empty published info.
func (s *HandlerDeploymentPublishSuite) Test_PublishedStateReflected() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		s.publishURL(depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	// Fetch the deployment and verify published is populated.
	getResponse := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%s", depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, getResponse.Code)

	body := getResponse.Map()

	got := body["deployment"].(map[string]any)
	published := got["published"].([]any)
	s.Len(published, 1)

	p := published[0].(map[string]any)
	s.Equal(env.ID.String(), p["envId"])
	s.Equal(float64(100), p["percentage"])
}

// Test_NotFound_UnknownID verifies that a non-existent deployment ID returns 404.
func (s *HandlerDeploymentPublishSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/deployments/999999999/publish",
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment in a different environment returns 404.
func (s *HandlerDeploymentPublishSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})

	depl := s.MockDeployment(env1)
	key := s.MockAPIKey(appl, env2)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		s.publishURL(depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_NoAPIKey verifies that requests without an API key are rejected with 403.
func (s *HandlerDeploymentPublishSuite) Test_Forbidden_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		s.publishURL(depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key whose owner is not a member
// of the environment team is rejected with 403.
func (s *HandlerDeploymentPublishSuite) Test_Forbidden_UserNotMember() {
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

	// For POST endpoints, envId must be in the body (not query string) so the
	// middleware can resolve the environment before checking team membership.
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/publish", depl.ID),
		map[string]any{"envId": env.ID.String()},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_BadRequest_FailedDeployment verifies that attempting to publish a deployment
// with a non-zero exit code returns 400 with an appropriate error message.
func (s *HandlerDeploymentPublishSuite) Test_BadRequest_FailedDeployment() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		s.publishURL(depl.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusBadRequest, response.Code)

	body := response.Map()
	s.Equal([]any{"Deployment must have a successful build before it can be published"}, body["errors"])
}

func TestHandlerDeploymentPublish(t *testing.T) {
	suite.Run(t, &HandlerDeploymentPublishSuite{})
}
