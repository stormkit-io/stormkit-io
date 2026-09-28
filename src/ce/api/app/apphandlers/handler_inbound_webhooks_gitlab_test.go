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
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
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

func (s *InboundGitlabSuite) app(autoDeploy bool, envOverwrites ...map[string]any) *factory.MockApp {
	app := s.MockApp(nil, map[string]any{
		"Repo": "gitlab/stormkit-test-acc/test-repo",
	})

	overwrites := []map[string]any{{
		"AutoDeploy": autoDeploy,
	}}
	overwrites = append(overwrites, envOverwrites...)
	s.MockEnv(app, overwrites...)

	return app
}

// post sends a GitLab webhook with the given event to target.
func (s *InboundGitlabSuite) post(target, event string, payload map[string]any) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		target,
		payload,
		map[string]string{"X-Gitlab-Event": event},
	)
}

func (s *InboundGitlabSuite) pushPayload() map[string]any {
	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(gitlabPushExample), &payload))
	return payload
}

// Test_Rejected_NoSecret verifies that webhooks without an app secret never trigger a deployment.
func (s *InboundGitlabSuite) Test_Rejected_NoSecret() {
	s.app(true)

	response := s.post("/app/webhooks/gitlab", "Push Hook", s.pushPayload())

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGitlabSuite) Test_Rejected_InvalidSecret() {
	s.app(true)

	response := s.post("/app/webhooks/gitlab/not-a-secret", "Push Hook", s.pushPayload())

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Rejected_InvalidSecret_BodyNotRead verifies that requests without a
// valid app secret are rejected before their payload is parsed.
func (s *InboundGitlabSuite) Test_Rejected_InvalidSecret_BodyNotRead() {
	code, read := postWebhook("/app/webhooks/gitlab/not-a-secret", map[string]string{"X-Gitlab-Event": "Push Hook"})

	s.Equal(http.StatusForbidden, code)
	s.False(read)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Rejected_SecretOfAnotherRepo verifies that a valid secret cannot be
// used to trigger deployments for a repository its app is not connected to.
func (s *InboundGitlabSuite) Test_Rejected_SecretOfAnotherRepo() {
	s.app(true)

	other := s.MockApp(nil, map[string]any{"Repo": "gitlab/attacker/other-repo"})

	response := s.post(fmt.Sprintf("/app/webhooks/gitlab/%s", other.Secret()), "Push Hook", s.pushPayload())

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_OnlyVerifiedAppDeploys verifies that a webhook deploys only the app
// whose secret it carries, even when other apps use the same repository.
func (s *InboundGitlabSuite) Test_OnlyVerifiedAppDeploys() {
	appl := s.app(true)
	s.app(true)

	response := s.post(fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()), "Push Hook", s.pushPayload())

	s.Equal(http.StatusOK, response.Code)
	s.mockDeployer.AssertNumberOfCalls(s.T(), "Deploy", 1)
	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return _appl.ID == appl.ID
		}),
		mock.Anything,
	)
}

// Test_SameCommitDeploysEveryApp verifies that a commit built for one app is
// still built for another app on the same repository, which has its own hook.
func (s *InboundGitlabSuite) Test_SameCommitDeploysEveryApp() {
	s.app(true)
	firstEnv := s.GetEnv()
	second := s.app(true)

	s.MockDeployment(firstEnv, map[string]any{
		"Commit": deploy.CommitInfo{ID: null.NewString("123abc456FGH", true)},
	})

	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(gitlabMergeExample("opened")), &payload))

	response := s.post(fmt.Sprintf("/app/webhooks/gitlab/%s", second.Secret()), "Merge Request Hook", payload)

	s.Equal(http.StatusOK, response.Code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return _appl.ID == second.ID
		}),
		mock.Anything,
	)
}

// Test_PushEvent_NoCommits verifies that pushes without commits are ignored.
func (s *InboundGitlabSuite) Test_PushEvent_NoCommits() {
	appl := s.app(true)
	payload := s.pushPayload()
	payload["commits"] = []any{}

	response := s.post(fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()), "Push Hook", payload)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_MergeRequestOpened_Fork verifies that merge requests from forks are not built automatically.
func (s *InboundGitlabSuite) Test_MergeRequestOpened_Fork() {
	appl := s.app(true)

	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(gitlabMergeExample("opened")), &payload))

	attrs := payload["object_attributes"].(map[string]any)
	attrs["source"].(map[string]any)["path_with_namespace"] = "attacker/test-repo"

	response := s.post(fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()), "Merge Request Hook", payload)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGitlabSuite) Test_NoAutoDeploy() {
	appl := s.app(false)

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(gitlabPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()),
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

// gitlabPushPayload builds a push payload carrying the given commits.
// totalCommits mirrors GitLab's total_commits_count, which is how a truncated
// commit list is detected.
func (s *InboundGitlabSuite) gitlabPushPayload(commits []map[string]any, totalCommits int) map[string]any {
	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(gitlabPushExample), &payload))
	payload["commits"] = commits
	payload["total_commits_count"] = totalCommits

	return payload
}

func (s *InboundGitlabSuite) gitlabPush(appl *factory.MockApp, payload map[string]any) int {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/gitlab/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Gitlab-Event": "Push Hook",
		},
	)

	return response.Code
}

func (s *InboundGitlabSuite) Test_PushEvent_BuildRootUnchanged() {
	appl := s.app(true, map[string]any{
		"Data": &buildconf.BuildConf{
			WorkDir:                "apps/frontend",
			SkipUnchangedBuildRoot: null.BoolFrom(true),
		},
	})

	code := s.gitlabPush(appl, s.gitlabPushPayload([]map[string]any{
		{
			"message":  "Update infrastructure",
			"modified": []string{"apps/infrastructure/main.tf"},
		},
	}, 1))

	s.Equal(http.StatusNoContent, code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// A change to a watched shared package deploys even though it sits outside the
// build root.
func (s *InboundGitlabSuite) Test_PushEvent_WatchPathChanged() {
	appl := s.app(true, map[string]any{
		"Data": &buildconf.BuildConf{
			WorkDir:                "apps/frontend",
			WatchPaths:             []string{"packages/ui"},
			SkipUnchangedBuildRoot: null.BoolFrom(true),
		},
	})

	code := s.gitlabPush(appl, s.gitlabPushPayload([]map[string]any{
		{
			"message":  "Restyle the button",
			"modified": []string{"packages/ui/button.tsx"},
		},
	}, 1))

	s.Equal(http.StatusOK, code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

// GitLab caps the commit list but reports the real total. A mismatch means the
// change set is incomplete, so the candidate has to be kept.
func (s *InboundGitlabSuite) Test_PushEvent_TruncatedCommitsDeploys() {
	appl := s.app(true, map[string]any{
		"Data": &buildconf.BuildConf{
			WorkDir:                "apps/frontend",
			SkipUnchangedBuildRoot: null.BoolFrom(true),
		},
	})

	code := s.gitlabPush(appl, s.gitlabPushPayload([]map[string]any{
		{
			"message":  "Update infrastructure",
			"modified": []string{"apps/infrastructure/main.tf"},
		},
	}, 25))

	s.Equal(http.StatusOK, code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
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
				a.Equal(false, _depl.IsFork) &&
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
