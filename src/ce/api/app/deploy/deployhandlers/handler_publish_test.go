package deployhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerPublishDeploymentSuite struct {
	suite.Suite
	*factory.Factory
	conn           databasetest.TestDB
	originalFunc   func(ctx context.Context, settings []*deploy.PublishSettings) error
	calledSettings []*deploy.PublishSettings
}

func (s *HandlerPublishDeploymentSuite) SetupSuite() {
	s.originalFunc = deploy.Publish
}

func (s *HandlerPublishDeploymentSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.calledSettings = nil

	deployhandlers.Publish = func(ctx context.Context, settings []*deploy.PublishSettings) error {
		s.calledSettings = settings
		return nil
	}
}

func (s *HandlerPublishDeploymentSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	deployhandlers.Publish = nil
	shttp.DefaultRequest = nil
}

func (s *HandlerPublishDeploymentSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	dpl := s.MockDeployment(env, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deployments/publish",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"publish": []map[string]any{
				{"percentage": 100, "deploymentId": dpl.ID.String()},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expectedResponse := fmt.Sprintf(`{
		"appId": "%s",
		"config": [
			{ "percentage":100, "deploymentId": "%s" }
		],
		"envId": "%s"}`,
		app.ID.String(),
		dpl.ID.String(),
		env.ID.String())

	expectedSettings := []*deploy.PublishSettings{
		{
			DeploymentID: dpl.ID,
			EnvID:        env.ID,
			Percentage:   100,
		},
	}

	a := assert.New(s.T())
	a.Equal(http.StatusOK, response.Code)
	a.Equal(expectedSettings, s.calledSettings)
	a.JSONEq(expectedResponse, response.String())
}

func (s *HandlerPublishDeploymentSuite) Test_Success_Multiple() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	dpls := s.MockDeployments(2, env, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deployments/publish",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"publish": []map[string]any{
				{"percentage": 30, "deploymentId": dpls[0].ID.String()},
				{"percentage": 70, "deploymentId": dpls[1].ID.String()},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expectedResponse := fmt.Sprintf(`{
		"appId": "%s",
		"config": [
			{ "percentage": 30, "deploymentId": "%s" },
			{ "percentage": 70, "deploymentId": "%s" }
		],
		"envId": "%s"}`,
		app.ID.String(),
		dpls[0].ID.String(),
		dpls[1].ID.String(),
		env.ID.String())

	expectedSettings := []*deploy.PublishSettings{
		{
			DeploymentID: dpls[0].ID,
			EnvID:        env.ID,
			Percentage:   30,
		},
		{
			DeploymentID: dpls[1].ID,
			EnvID:        env.ID,
			Percentage:   70,
		},
	}

	a := assert.New(s.T())
	a.Equal(http.StatusOK, response.Code)
	a.Equal(expectedSettings, s.calledSettings)
	a.JSONEq(expectedResponse, response.String())
}

func (s *HandlerPublishDeploymentSuite) Test_BadRequest() {
	usr := s.MockUser()
	app := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deployments/publish",
		map[string]any{
			"appId": app.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expectedResponse := `{"errors":{"envId":"Environment ID is a required field","percentage":"The sum of percentages should be 100 in order to publish."},"ok":false}`

	a := assert.New(s.T())
	a.Equal(http.StatusBadRequest, response.Code)
	a.Equal(expectedResponse, response.String())
	a.Nil(s.calledSettings)
}

func (s *HandlerPublishDeploymentSuite) Test_BadRequest_PercentageNot100() {
	usr := s.MockUser()
	app := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deployments/publish",
		map[string]any{
			"appId": app.ID.String(),
			"envId": app.ID.String(),
			"publish": []map[string]any{
				{"percentage": 70, "deploymentId": app.ID.String()},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expectedResponse := `{"errors":{"percentage":"The sum of percentages should be 100 in order to publish."},"ok":false}`

	a := assert.New(s.T())
	a.Equal(http.StatusBadRequest, response.Code)
	a.Equal(expectedResponse, response.String())
	a.Nil(s.calledSettings)
}

func TestHandlerPublishDeployment(t *testing.T) {
	suite.Run(t, &HandlerPublishDeploymentSuite{})
}
