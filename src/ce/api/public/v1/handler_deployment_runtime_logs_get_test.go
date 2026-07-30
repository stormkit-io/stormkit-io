package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerDeploymentRuntimeLogsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	user       *factory.MockUser
	app        *factory.MockApp
	env        *factory.MockEnv
	deployment *factory.MockDeployment
	logs       []*applog.Log
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.user = s.MockUser()
	s.app = s.MockApp(s.user)
	s.env = s.MockEnv(s.app)
	s.deployment = s.MockDeployment(s.env)

	s.logs = []*applog.Log{
		{
			AppID:         s.app.ID,
			DeploymentID:  s.deployment.ID,
			EnvironmentID: s.env.ID,
			Label:         "info",
			Data:          "Lorem ipsum dolor sit amet.",
			RequestID:     "x5818c-47818-ca2de192e",
			Timestamp:     time.Date(2022, 10, 10, 5, 30, 45, 0, time.UTC).Unix(),
		},
		{
			AppID:         s.app.ID,
			DeploymentID:  s.deployment.ID,
			EnvironmentID: s.env.ID,
			Label:         "info",
			Data:          "Consectetur adipiscing elit, sed do eiusmod tempor incididunt.",
			RequestID:     "x2387a-21186-ba1ce42ef",
			Timestamp:     time.Date(2023, 10, 10, 6, 30, 45, 0, time.UTC).Unix(),
		},
	}

	s.Require().NoError(applog.NewStore().InsertLogs(context.Background(), s.logs))
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) request(query string, auth string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%s/runtime-logs?envId=%s%s",
			s.deployment.ID.String(), s.env.ID.String(), query),
		nil,
		map[string]string{"Authorization": auth},
	)
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) logJSON(i int) string {
	return fmt.Sprintf(`{
		"id": "%s",
		"appId": "%s",
		"deploymentId": "%s",
		"data": "%s",
		"timestamp": "%d"
	}`,
		s.logs[i].ID.String(),
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[i].Data,
		s.logs[i].Timestamp,
	)
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_Success() {
	response := s.request("", usertest.Authorization(s.user.ID))

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(fmt.Sprintf(`{"logs": [%s, %s], "hasNextPage": false}`,
		s.logJSON(0), s.logJSON(1)), response.String())
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_Success_SortDesc() {
	response := s.request("&sort=desc", usertest.Authorization(s.user.ID))

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(fmt.Sprintf(`{"logs": [%s, %s], "hasNextPage": false}`,
		s.logJSON(1), s.logJSON(0)), response.String())
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_BadRequest_InvalidSort() {
	response := s.request("&sort=something-else", usertest.Authorization(s.user.ID))

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"errors": ["sort must be either 'asc' or 'desc'."]}`, response.String())
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_Success_HasNextPage() {
	original := publicapiv1.RuntimeLogsLimit
	publicapiv1.RuntimeLogsLimit = 1

	defer func() {
		publicapiv1.RuntimeLogsLimit = original
	}()

	response := s.request("", usertest.Authorization(s.user.ID))

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(fmt.Sprintf(`{"logs": [%s], "hasNextPage": true}`, s.logJSON(0)), response.String())
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_Success_Pagination() {
	response := s.request(
		fmt.Sprintf("&beforeId=%s", s.logs[0].ID.String()),
		usertest.Authorization(s.user.ID),
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(fmt.Sprintf(`{"logs": [%s], "hasNextPage": false}`, s.logJSON(1)), response.String())
}

// The deployment must belong to the environment the caller authorized against,
// otherwise logs could be read across environments of the same app.
func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_NotFound_DeploymentInAnotherEnv() {
	otherEnv := s.MockEnv(s.app, map[string]any{"Name": "staging"})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/deployments/%s/runtime-logs?envId=%s",
			s.deployment.ID.String(), otherEnv.ID.String()),
		nil,
		map[string]string{"Authorization": usertest.Authorization(s.user.ID)},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerDeploymentRuntimeLogsGetSuite) Test_Forbidden_NotMember() {
	other := s.MockUser()

	response := s.request("", usertest.Authorization(other.ID))

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerDeploymentRuntimeLogsGet(t *testing.T) {
	suite.Run(t, &HandlerDeploymentRuntimeLogsGetSuite{})
}
