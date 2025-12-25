package deployhandlers_test

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"

	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	null "gopkg.in/guregu/null.v3"
)

type DeployStopSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *DeployStopSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *DeployStopSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *DeployStopSuite) Test_Stop_Success() {
	depl := s.Factory.MockDeployment(nil, map[string]any{
		"ExitCode": null.NewInt(0, false),
	})

	app := s.Factory.GetApp()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/stop",
		map[string]any{
			"appId":        app.ID.String(),
			"deploymentId": depl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	d, err := deploy.NewStore().MyDeployment(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
	})

	s.NoError(err)
	s.Equal(response.Code, http.StatusOK)
	s.Equal(deploy.ExitCodeStopped, d.ExitCode.ValueOrZero())
}

func (s *DeployStopSuite) Test_Stop_StatusChecks_Success() {
	depl := s.Factory.MockDeployment(nil, map[string]any{
		"ExitCode": null.IntFrom(0),
	})

	app := s.Factory.GetApp()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/stop",
		map[string]any{
			"appId":        app.ID.String(),
			"deploymentId": depl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	d, err := deploy.NewStore().MyDeployment(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
	})

	s.NoError(err)
	s.Equal(response.Code, http.StatusOK)
	s.Equal(deploy.ExitCodeSuccess, d.ExitCode.ValueOrZero())
	s.True(d.StatusChecksPassed.Valid)
	s.False(d.StatusChecksPassed.ValueOrZero())
}

func (s *DeployStopSuite) Test_Fail_AlreadyStopped() {
	depl := s.Factory.MockDeployment(nil, map[string]any{
		"ExitCode": null.NewInt(deploy.ExitCodeStopped, true),
	})

	app := s.Factory.GetApp()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/stop",
		map[string]any{
			"appId":        app.ID.String(),
			"deploymentId": depl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	d, err := deploy.NewStore().MyDeployment(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
	})

	s.Equal(http.StatusOK, response.Code)
	s.NoError(err)
	s.Equal(deploy.ExitCodeStopped, d.ExitCode.ValueOrZero())
}

func (s *DeployStopSuite) Test_Fail_AlreadyCompleted() {
	depl := s.Factory.MockDeployment(nil, map[string]any{
		"ExitCode":    null.IntFrom(deploy.ExitCodeSuccess),
		"IsImmutable": null.BoolFrom(true),
	})

	app := s.Factory.GetApp()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/stop",
		map[string]any{
			"appId":        app.ID.String(),
			"deploymentId": depl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	d, err := deploy.NewStore().MyDeployment(context.Background(), &deploy.DeploymentsQueryFilters{
		DeploymentID: depl.ID,
	})

	s.Equal(http.StatusOK, response.Code)
	s.NoError(err)
	s.Equal(deploy.ExitCodeSuccess, d.ExitCode.ValueOrZero())
}

func (s *DeployStopSuite) Test_Fail_DeploymentNotFound() {
	app := s.Factory.MockApp(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy/stop",
		map[string]any{
			"appId":        app.ID.String(),
			"deploymentId": "5918",
		},
		map[string]string{
			"Authorization": usertest.Authorization(app.UserID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerDeployStop(t *testing.T) {
	suite.Run(t, &DeployStopSuite{})
}
