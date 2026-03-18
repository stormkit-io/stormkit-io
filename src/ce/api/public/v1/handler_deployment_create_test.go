package publicapiv1_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type HandlerDeploymentCreateSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *HandlerDeploymentCreateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *HandlerDeploymentCreateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	deployservice.MockDeployer = nil
}

func (s *HandlerDeploymentCreateSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{
			"envId": env.ID,
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusCreated, response.Code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerDeploymentCreateSuite) Test_Success_WithDefaults() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusCreated, response.Code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.MatchedBy(func(depl *deploy.Deployment) bool {
		return depl.Branch == env.Branch && depl.ShouldPublish == false
	}))
}

func (s *HandlerDeploymentCreateSuite) Test_Success_WithBranchAndPublish() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{
			"branch":  "feature/my-branch",
			"publish": true,
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusCreated, response.Code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.MatchedBy(func(depl *deploy.Deployment) bool {
		return depl.Branch == "feature/my-branch" && depl.ShouldPublish == true
	}))
}

func (s *HandlerDeploymentCreateSuite) Test_Unauthorized_NoAPIKey() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{"branch": "main"},
		map[string]string{},
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *HandlerDeploymentCreateSuite) Test_NotFound_Env() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env)

	_, err := buildconf.NewStore().MarkAsDeleted(context.Background(), env.ID)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Forbidden_TeamTokenNotOwner verifies that a team-scoped token whose TeamID does not
// match the application's TeamID is rejected with 403.
func (s *HandlerDeploymentCreateSuite) Test_Forbidden_TeamTokenNotOwner() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)
	env := s.MockEnv(appl)

	// usr2 gets its own default team; its team does NOT own the application.
	usr2 := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"TeamID": usr2.DefaultTeamID,
		"Scope":  apikey.SCOPE_TEAM,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"UserID": types.ID(0),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{
			"envId": env.ID,
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Forbidden_UserNotTeamMember verifies that a user-scoped token whose owner is not a
// member of the application's team is rejected with 403.
func (s *HandlerDeploymentCreateSuite) Test_Forbidden_UserNotTeamMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)
	env := s.MockEnv(appl)

	// usr2 is a distinct user who has never been added to usr1's team.
	usr2 := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr2.ID,
		"Scope":  apikey.SCOPE_USER,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{
			"envId": env.ID,
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Forbidden_AppTokenNotOwner verifies that an app-scoped token whose AppID does not
// match the application derived from envId is rejected with 403.
func (s *HandlerDeploymentCreateSuite) Test_Forbidden_AppTokenNotOwner() {
	usr := s.MockUser()
	appl1 := s.MockApp(usr)
	appl2 := s.MockApp(usr)
	env2 := s.MockEnv(appl2)

	// Key is scoped to appl1 but the request targets appl2's environment.
	key := s.MockAPIKey(appl1, nil, map[string]any{
		"EnvID": types.ID(0),
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/deploy",
		map[string]any{
			"envId": env2.ID,
		},
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func TestHandlerDeploymentCreate(t *testing.T) {
	suite.Run(t, &HandlerDeploymentCreateSuite{})
}
