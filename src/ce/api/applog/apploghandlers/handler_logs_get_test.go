package apploghandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/ce/api/applog/apploghandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerLogsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	user       *factory.MockUser
	app        *factory.MockApp
	env        *factory.MockEnv
	deployment *factory.MockDeployment
	logs       []*applog.Log
}

func (s *HandlerLogsGetSuite) SetupSuite() {
	s.conn = databasetest.InitTx("handler_logs_get_suite")
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
			Timestamp:     time.Date(2022, 10, 10, 5, 30, 45, 0, &time.Location{}).Unix(),
		},
		{
			AppID:         s.app.ID,
			DeploymentID:  s.deployment.ID,
			EnvironmentID: s.env.ID,
			Label:         "info",
			Data:          "Consectetur adipiscing elit, sed do eiusmod tempor incididunt.",
			RequestID:     "x2387a-21186-ba1ce42ef",
			Timestamp:     time.Date(2023, 10, 10, 6, 30, 45, 0, &time.Location{}).Unix(),
		},
	}

	s.NoError(applog.NewStore().InsertLogs(context.Background(), s.logs))
}

func (s *HandlerLogsGetSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *HandlerLogsGetSuite) Test_FetchingLogs() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apploghandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/app/%s/logs?deploymentId=%s",
			s.app.ID.String(),
			s.deployment.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"logs": [
			{
				"id": "1",
				"appId": "%s",
				"deploymentId": "%s",
				"data": "Lorem ipsum dolor sit amet.",
				"timestamp": "%d"
			},
			{
				"id": "2",
				"appId": "%s",
				"deploymentId": "%s",
				"data": "Consectetur adipiscing elit, sed do eiusmod tempor incididunt.",
				"timestamp": "%d"
			}
		],
		"hasNextPage": false
	}`,
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[0].Timestamp,
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[1].Timestamp,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerLogsGetSuite) Test_FetchingLogs_Sort() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apploghandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/app/%s/logs?deploymentId=%s&sort=desc",
			s.app.ID.String(),
			s.deployment.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"logs": [
			{
				"id": "2",
				"appId": "%s",
				"deploymentId": "%s",
				"data": "Consectetur adipiscing elit, sed do eiusmod tempor incididunt.",
				"timestamp": "%d"
			},
			{
				"id": "1",
				"appId": "%s",
				"deploymentId": "%s",
				"data": "Lorem ipsum dolor sit amet.",
				"timestamp": "%d"
			}
		],
		"hasNextPage": false
	}`,
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[1].Timestamp,
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[0].Timestamp,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerLogsGetSuite) Test_FetchingLogs_Sort_Invalid() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apploghandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/app/%s/logs?deploymentId=%s&sort=something-else",
			s.app.ID.String(),
			s.deployment.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := `{
		"error": "sort must be either 'asc' or 'desc'."
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerLogsGetSuite) TestFetchingLogs_HasNextPage() {
	originalLogLimit := apploghandlers.LogsLimit
	apploghandlers.LogsLimit = 1

	defer func() {
		apploghandlers.LogsLimit = originalLogLimit
	}()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apploghandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/app/%s/logs?deploymentId=%s",
			s.app.ID.String(),
			s.deployment.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"logs": [
			{
				"id": "1",
				"appId": "%s",
				"deploymentId": "%s",
				"data": "Lorem ipsum dolor sit amet.",
				"timestamp": "%d"
			}
		],
		"hasNextPage": true
	}`,
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[0].Timestamp,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerLogsGetSuite) TestFetchingLogs_Pagination() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apploghandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/app/%s/logs?deploymentId=%s&beforeId=%s",
			s.app.ID.String(),
			s.deployment.ID.String(),
			s.logs[0].ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"logs": [
			{
				"id": "2",
				"appId": "%s",
				"deploymentId": "%s",
				"data": "Consectetur adipiscing elit, sed do eiusmod tempor incididunt.",
				"timestamp": "%d"
			}
		],
		"hasNextPage": false
	}`,
		s.app.ID.String(),
		s.deployment.ID.String(),
		s.logs[1].Timestamp,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerLogsGet(t *testing.T) {
	suite.Run(t, &HandlerLogsGetSuite{})
}
