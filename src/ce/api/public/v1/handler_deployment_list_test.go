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

type HandlerDeploymentListSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerDeploymentListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeploymentListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentListSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerDeploymentListSuite) get(keyValue, query string) shttptest.Response {
	url := "/v1/deployments"
	if query != "" {
		url += "?" + query
	}

	return shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, url, nil,
		map[string]string{"Authorization": keyValue})
}

// Test_Success verifies that an env-scoped key returns deployments for that environment.
func (s *HandlerDeploymentListSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	s.MockDeployments(3, env)
	key := s.MockAPIKey(appl, env)

	response := s.get(key.Value, "")

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()

	depls, ok := body["deployments"].([]any)
	s.True(ok)
	s.Len(depls, 3)
	s.Equal(false, body["hasNextPage"])
}

// Test_Success_HasNextPage verifies that hasNextPage is true when more results exist.
func (s *HandlerDeploymentListSuite) Test_Success_HasNextPage() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	s.MockDeployments(21, env)
	key := s.MockAPIKey(appl, env)

	response := s.get(key.Value, "")

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()

	depls := body["deployments"].([]any)
	s.Len(depls, 20)
	s.Equal(true, body["hasNextPage"])
}

// Test_Success_Pagination verifies that ?from= offsets the result set to the next page.
func (s *HandlerDeploymentListSuite) Test_Success_Pagination() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	s.MockDeployments(21, env)
	key := s.MockAPIKey(appl, env)

	// First page: 20 results, more available.
	firstPage := s.get(key.Value, "")
	s.Equal(http.StatusOK, firstPage.Code)
	firstBody := firstPage.Map()
	s.Len(firstBody["deployments"].([]any), 20)
	s.Equal(true, firstBody["hasNextPage"])

	// Second page: the remaining 1 result.
	secondPage := s.get(key.Value, "from=20")
	s.Equal(http.StatusOK, secondPage.Code)
	secondBody := secondPage.Map()
	s.Len(secondBody["deployments"].([]any), 1)
	s.Equal(false, secondBody["hasNextPage"])
}

// Test_Success_FilterByBranch verifies that ?branch= filters the results.
func (s *HandlerDeploymentListSuite) Test_Success_FilterByBranch() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	s.MockDeployment(env, map[string]any{"Branch": "main"})
	s.MockDeployment(env, map[string]any{"Branch": "staging"})
	key := s.MockAPIKey(appl, env)

	response := s.get(key.Value, "branch=main")

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()

	depls := body["deployments"].([]any)
	s.Len(depls, 1)

	d := depls[0].(map[string]any)
	s.Equal("main", d["branch"])
}

// Test_IsolatedByEnv verifies that deployments from other environments are not returned.
func (s *HandlerDeploymentListSuite) Test_IsolatedByEnv() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env1 := s.MockEnv(appl)
	env2 := s.MockEnv(appl, map[string]any{"Name": "staging"})

	s.MockDeployment(env2) // belongs to env2
	key := s.MockAPIKey(appl, env1)

	response := s.get(key.Value, "")

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()

	depls := body["deployments"].([]any)
	s.Empty(depls)
}

// Test_Forbidden_NoAPIKey verifies that requests without a key are rejected.
func (s *HandlerDeploymentListSuite) Test_Forbidden_NoAPIKey() {
	response := shttptest.RequestWithHeaders(s.handler(), shttp.MethodGet, "/v1/deployments", nil, map[string]string{})
	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Forbidden_UserNotMember verifies that a user-scoped key with no team membership is rejected.
func (s *HandlerDeploymentListSuite) Test_Forbidden_UserNotMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)
	env := s.MockEnv(appl)

	usr2 := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr2.ID,
		"Scope":  apikey.SCOPE_USER,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
	})

	response := s.get(key.Value, fmt.Sprintf("envId=%d", env.ID))
	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerDeploymentList(t *testing.T) {
	suite.Run(t, &HandlerDeploymentListSuite{})
}
