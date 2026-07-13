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
)

type HandlerDeploymentDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeploymentDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentDeleteSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerDeploymentDeleteSuite) delete(keyValue string, deplID fmt.Stringer) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/v1/deployments/%s", deplID),
		nil,
		map[string]string{"Authorization": keyValue},
	)
}

// Test_Success verifies that a valid env-scoped key can delete a deployment.
func (s *HandlerDeploymentDeleteSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)
	key := s.MockAPIKey(appl, env)

	response := s.delete(key.Value, depl.ID)

	s.Equal(http.StatusOK, response.Code)
	s.Equal(true, response.Map()["ok"])

	// Verify the deployment is actually deleted.
	response = s.delete(key.Value, depl.ID)
	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_UnknownID verifies that a non-existent deployment returns 404.
func (s *HandlerDeploymentDeleteSuite) Test_NotFound_UnknownID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := s.delete(key.Value, types.ID(999999999))

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_NotFound_WrongEnv verifies that a deployment belonging to a different env returns 404.
func (s *HandlerDeploymentDeleteSuite) Test_NotFound_WrongEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})
	depl := s.MockDeployment(env1)
	key := s.MockAPIKey(appl, env2)

	response := s.delete(key.Value, depl.ID)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_NoAPIKey verifies that requests without an API key are rejected.
func (s *HandlerDeploymentDeleteSuite) Test_Forbidden_NoAPIKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/v1/deployments/%s", depl.ID),
		nil,
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key with no membership is rejected.
func (s *HandlerDeploymentDeleteSuite) Test_Forbidden_UserNotMember() {
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
		shttp.MethodDelete,
		fmt.Sprintf("/v1/deployments/%s?envId=%d", depl.ID, env.ID),
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerDeploymentDelete(t *testing.T) {
	suite.Run(t, &HandlerDeploymentDeleteSuite{})
}
