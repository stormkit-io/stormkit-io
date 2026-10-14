package apphandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type InboundForgejoSuite struct {
	suite.Suite
	*factory.Factory

	conn          databasetest.TestDB
	mockDeployer  *mocks.Deployer
	restoreRunner func()
}

func (s *InboundForgejoSuite) BeforeTest(suiteName, _ string) {
	s.restoreRunner = apphandlers.RunWebhookDeploysSync()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *InboundForgejoSuite) AfterTest(_, _ string) {
	s.restoreRunner()
	s.conn.CloseTx()
	deployservice.MockDeployer = nil
}

func (s *InboundForgejoSuite) app() *factory.MockApp {
	appl := s.MockApp(nil, map[string]any{"Repo": "forgejo/stormkit-test-acc/test-repo"})
	s.MockEnv(appl, map[string]any{"AutoDeploy": true})
	return appl
}

// post sends a Forgejo webhook with the given event to the app's hook URL.
func (s *InboundForgejoSuite) post(appl *factory.MockApp, event string, payload map[string]any) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/forgejo/%s", appl.Secret()),
		payload,
		map[string]string{"X-Forgejo-Event": event},
	)
}

func (s *InboundForgejoSuite) pushPayload(branch string) map[string]any {
	return map[string]any{
		"ref":           "refs/heads/" + branch,
		"total_commits": 1,
		"commits": []map[string]any{
			{"message": "Update readme\n\nLonger description", "modified": []string{"README.md"}},
		},
		"repository": map[string]any{
			"full_name":      "stormkit-test-acc/test-repo",
			"default_branch": "main",
		},
	}
}

func (s *InboundForgejoSuite) pullRequestPayload(action, headRepo string) map[string]any {
	return map[string]any{
		"action": action,
		"number": 7,
		"pull_request": map[string]any{
			"title": "Add landing page",
			"head": map[string]any{
				"ref":  "feat/landing",
				"sha":  "abc123",
				"repo": map[string]any{"full_name": headRepo},
			},
		},
		"repository": map[string]any{
			"full_name":      "stormkit-test-acc/test-repo",
			"default_branch": "main",
		},
	}
}

func (s *InboundForgejoSuite) Test_PushEvent_DefaultBranch() {
	appl := s.app()

	response := s.post(appl, "push", s.pushPayload("main"))

	s.Equal(http.StatusOK, response.Code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything,
		mock.MatchedBy(func(a *app.App) bool { return a.ID == appl.ID }),
		mock.MatchedBy(func(d *deploy.Deployment) bool {
			return d.CheckoutRepo == "forgejo/stormkit-test-acc/test-repo" &&
				d.Branch == "main" &&
				d.IsAutoDeploy &&
				d.PullRequestNumber.ValueOrZero() == 0
		}),
	)
}

// Test_PushEvent_OtherBranch verifies that pushes outside the default branch
// are left to their pull requests.
func (s *InboundForgejoSuite) Test_PushEvent_OtherBranch() {
	response := s.post(s.app(), "push", s.pushPayload("feat/landing"))

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundForgejoSuite) Test_PushEvent_GiteaHeader() {
	appl := s.app()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/forgejo/%s", appl.Secret()),
		s.pushPayload("main"),
		map[string]string{"X-Gitea-Event": "push"},
	)

	s.Equal(http.StatusOK, response.Code)
	s.mockDeployer.AssertNumberOfCalls(s.T(), "Deploy", 1)
}

func (s *InboundForgejoSuite) Test_PullRequestOpened() {
	appl := s.app()

	response := s.post(appl, "pull_request", s.pullRequestPayload("opened", "stormkit-test-acc/test-repo"))

	s.Equal(http.StatusOK, response.Code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything,
		mock.Anything,
		mock.MatchedBy(func(d *deploy.Deployment) bool {
			return d.Branch == "feat/landing" &&
				d.PullRequestNumber.ValueOrZero() == 7
		}),
	)
}

func (s *InboundForgejoSuite) Test_PullRequestClosed() {
	response := s.post(s.app(), "pull_request", s.pullRequestPayload("closed", "stormkit-test-acc/test-repo"))

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_PullRequest_Fork verifies that pull requests from forks are not built automatically.
func (s *InboundForgejoSuite) Test_PullRequest_Fork() {
	response := s.post(s.app(), "pull_request", s.pullRequestPayload("synchronized", "attacker/test-repo"))

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Rejected_SecretOfAnotherRepo verifies that a valid secret cannot be
// used to trigger deployments for a repository its app is not connected to.
func (s *InboundForgejoSuite) Test_Rejected_SecretOfAnotherRepo() {
	s.app()
	other := s.MockApp(nil, map[string]any{"Repo": "forgejo/attacker/other-repo"})

	response := s.post(other, "push", s.pushPayload("main"))

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundForgejoSuite) Test_Rejected_InvalidSecret() {
	s.app()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/webhooks/forgejo/not-a-secret",
		s.pushPayload("main"),
		map[string]string{"X-Forgejo-Event": "push"},
	)

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundForgejoSuite) Test_UnsupportedEvent() {
	response := s.post(s.app(), "issues", s.pushPayload("main"))

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func TestInboundForgejoSuite(t *testing.T) {
	suite.Run(t, new(InboundForgejoSuite))
}
