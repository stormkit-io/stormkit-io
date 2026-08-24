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
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerDeploymentPrioritizeSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentPrioritizeSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	insp := tasks.Inspector()
	insp.DeleteQueue(tasks.QueueDeployService, true)
	insp.DeleteQueue(tasks.QueueDeployServicePriority, true)
}

func (s *HandlerDeploymentPrioritizeSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()

	insp := tasks.Inspector()
	insp.DeleteQueue(tasks.QueueDeployService, true)
	insp.DeleteQueue(tasks.QueueDeployServicePriority, true)
}

func (s *HandlerDeploymentPrioritizeSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerDeploymentPrioritizeSuite) prioritize(keyValue string, deplID fmt.Stringer) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/prioritize", deplID),
		nil,
		map[string]string{"Authorization": keyValue},
	)
}

// enqueueTask places a fake task for the given deployment ID in the regular deploy queue.
func (s *HandlerDeploymentPrioritizeSuite) enqueueTask(deplID types.ID) {
	_, err := tasks.Enqueue(s.T().Context(), tasks.DeploymentStart, "encrypted-payload", &tasks.EnqueueOptions{
		MaxRetry:  10,
		QueueName: tasks.QueueDeployService,
		TaskID:    fmt.Sprintf("deployment-%s", deplID.String()),
	})
	s.Require().NoError(err)
}

// Test_Success verifies that a queued deployment is moved to the priority queue.
func (s *HandlerDeploymentPrioritizeSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	s.enqueueTask(depl.ID)

	response := s.prioritize(key.Value, depl.ID)

	s.Equal(http.StatusOK, response.Code)
	s.Equal(true, response.Map()["ok"])

	insp := tasks.Inspector()

	_, err := insp.GetTaskInfo(tasks.QueueDeployServicePriority, fmt.Sprintf("deployment-%s", depl.ID.String()))
	s.NoError(err, "task should be in the priority queue")
}

// Test_BadRequest_NotQueued verifies that a deployment with no Asynq task returns 400.
func (s *HandlerDeploymentPrioritizeSuite) Test_BadRequest_NotQueued() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	// No task enqueued — inspector.GetTaskInfo returns nil info.
	response := s.prioritize(key.Value, depl.ID)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal([]any{"Deployment is not queued or has already started"}, response.Map()["errors"])
}

// Test_BadRequest_NotRunning verifies that a non-running (finished) deployment returns 400.
func (s *HandlerDeploymentPrioritizeSuite) Test_BadRequest_NotRunning() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.MockAPIKey(appl, env)

	response := s.prioritize(key.Value, depl.ID)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal([]any{"Only queued deployments can be prioritized"}, response.Map()["errors"])
}

// Test_NotFound_UnknownID verifies that a non-existent deployment returns 404.
func (s *HandlerDeploymentPrioritizeSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := s.prioritize(key.Value, types.ID(999999999))

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment from another env returns 404.
func (s *HandlerDeploymentPrioritizeSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})
	depl := s.MockDeployment(env1)
	key := s.MockAPIKey(appl, env2)

	response := s.prioritize(key.Value, depl.ID)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_NoAPIKey verifies that requests without an API key are rejected.
func (s *HandlerDeploymentPrioritizeSuite) Test_Forbidden_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		fmt.Sprintf("/v1/deployments/%s/prioritize", depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key with no membership is rejected.
func (s *HandlerDeploymentPrioritizeSuite) Test_Forbidden_UserNotMember() {
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
		fmt.Sprintf("/v1/deployments/%s/prioritize", depl.ID),
		map[string]any{"envId": env.ID.String()},
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerDeploymentPrioritize(t *testing.T) {
	suite.Run(t, &HandlerDeploymentPrioritizeSuite{})
}
