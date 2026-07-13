package apphandlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	null "gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

var gitlabMergeExample = func(status string) string {
	return fmt.Sprintf(`{
		"object_kind": "merge_request",
		"project": {
		  "id": 1,
		  "path_with_namespace":"stormkit-test-acc/test-repo",
		  "default_branch":"master"
		},
		"object_attributes": {
		  "id": 99,
		  "iid": 41,
		  "target_branch": "master",
		  "source_branch": "ms-viewport",
		  "milestone_id": null,
		  "state": "%s",
		  "source": {
			"path_with_namespace":"stormkit-test-acc/test-repo",
			"default_branch":"ms-viewport"
		  },
		  "target": {
			"path_with_namespace":"stormkit-test-acc/test-repo",
			"default_branch":"master"
		  },
		  "last_commit": {
			"id": "123abc456FGH",
			"message": "fixed readme"
		  },
		  "action": "open"
		}
	}`, status)
}

const gitlabPushExample = `{
	"object_kind": "push",
	"ref": "refs/heads/main",
	"project_id": 15,
	"project":{
	  "id": 15,
	  "name":"Diaspora",
	  "description":"",
	  "path_with_namespace":"stormkit-test-acc/test-repo",
	  "default_branch":"main"
	},
	"commits": [
	  {
		"message": "Update Catalan translation to e38cb41.\n\nSee https://gitlab.com/gitlab-org/gitlab for more information"
	  }
	]
}`

type InboundGitlabSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *InboundGitlabSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *InboundGitlabSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	deployservice.MockDeployer = nil
}

func (s *InboundGitlabSuite) app(autoDeploy bool) *factory.MockApp {
	app := s.MockApp(nil, map[string]any{
		"Repo": "gitlab/stormkit-test-acc/test-repo",
	})

	s.MockEnv(app, map[string]any{
		"AutoDeploy": autoDeploy,
	})

	return app
}

func (s *InboundGitlabSuite) Test_NoAutoDeploy() {
	s.app(false)

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(gitlabPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/webhooks/gitlab",
		payload,
		map[string]string{
			"X-Gitlab-Event": "Push Hook",
		},
	)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGitlabSuite) TestPushEventSuccess() {
	a := assert.New(s.T())
	appl := s.app(true)

	payload := map[string]interface{}{}
	a.NoError(json.Unmarshal([]byte(gitlabPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Gitlab-Event": "Push Hook",
		},
	)

	a.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return a.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return a.Equal(_depl.CheckoutRepo, "gitlab/stormkit-test-acc/test-repo") &&
				a.Equal(true, _depl.IsAutoDeploy) &&
				a.Equal(int64(0), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
}

func (s *InboundGitlabSuite) TestMergeRequestOpened() {
	a := assert.New(s.T())
	appl := s.app(true)

	payload := map[string]interface{}{}
	a.NoError(json.Unmarshal([]byte(gitlabMergeExample("opened")), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Gitlab-Event": "Merge Request Hook",
		},
	)

	a.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return a.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return a.Equal(_depl.CheckoutRepo, "gitlab/stormkit-test-acc/test-repo") &&
				a.Equal(true, _depl.IsAutoDeploy) &&
				a.Equal("ms-viewport", _depl.Branch) &&
				a.Equal(int64(41), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
}

func (s *InboundGitlabSuite) Test_MergeRequestMerged() {
	appl := s.app(true)

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(gitlabMergeExample("merged")), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Gitlab-Event": "Merge Request Hook",
		},
	)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGitlabSuite) Test_DoNotRebuildSameCommits() {
	appl := s.app(true)

	s.MockDeployment(s.GetEnv(), map[string]any{
		"Branch": "my-private-branch",
		"Commit": deploy.CommitInfo{
			ID: null.NewString("123abc456FGH", true),
		},
	})

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(gitlabMergeExample("opened")), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Gitlab-Event": "Merge Request Hook",
		},
	)

	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
	s.Equal(http.StatusAlreadyReported, response.Code)
}

func TestInboundGitlab(t *testing.T) {
	suite.Run(t, &InboundGitlabSuite{})
}
