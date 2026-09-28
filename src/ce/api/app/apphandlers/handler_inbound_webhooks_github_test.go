package apphandlers_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	null "gopkg.in/guregu/null.v3"
)

type githubMergeParams struct {
	status   string
	merged   bool
	headRepo string
	baseRepo string
}

var githubMergeExample = func(params githubMergeParams) string {
	repo := "stormkit-test-acc/test-repo"

	if params.headRepo == "" {
		params.headRepo = repo
	}

	return fmt.Sprintf(`{
		"action": "%s",
		"pull_request": {
		  "state":"open",
		  "number": 53,
		  "title":"Change readme",
		  "head": {
			"ref": "my-pr-branch",
			"repo": {
				"full_name": "%s"
			}
	      },
		  "base": {
		    "ref": "master",
			"repo": {
				"full_name": "%s"
			}
		   },
		  "merged": %t
		},
		"repository": {
		  "full_name": "%s"
		}
	}`, params.status, params.headRepo, params.baseRepo, params.merged, repo)
}

const githubPushExample = `{
  "ref": "refs/heads/main",
  "head_commit": {
	"message": "Whatever is the message - you pick."
  },
  "repository": {
	"full_name": "stormkit-test-acc/test-repo"
  }
}`

const githubWebhookSecret = "random-token"

// githubSignature computes the X-Hub-Signature-256 header GitHub sends
// for the given payload.
func githubSignature(payload map[string]any) string {
	// Marshal the payload the same way the request body is built so that
	// whitespace differences do not break the signature.
	mac := hmac.New(sha256.New, []byte(githubWebhookSecret))
	body, _ := json.Marshal(payload)
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type InboundGithubSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *InboundGithubSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
	s.setWebhookSecret(githubWebhookSecret)
}

func (s *InboundGithubSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	deployservice.MockDeployer = nil
	admin.ResetCache(context.Background())
}

// setWebhookSecret configures the secret GitHub webhooks are verified with.
func (s *InboundGithubSuite) setWebhookSecret(secret string) {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{Github: admin.GithubConfig{WebhookSecret: secret}}
	admin.SetConfig(&cnf)
}

// push sends a push event to the webhook endpoint with the given headers.
func (s *InboundGithubSuite) push(headers map[string]string) shttptest.Response {
	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(githubPushExample), &payload))

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/webhooks/github/deploy",
		payload,
		headers,
	)
}

// Test_Rejected_MissingSignature verifies that unsigned payloads never trigger a deployment.
func (s *InboundGithubSuite) Test_Rejected_MissingSignature() {
	s.app(nil)

	response := s.push(map[string]string{"X-Github-Event": "push"})

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGithubSuite) Test_Rejected_InvalidSignature() {
	s.app(nil)

	response := s.push(map[string]string{
		"X-Github-Event":      "push",
		"X-Hub-Signature-256": "sha256=" + strings.Repeat("0", 64),
	})

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Rejected_MissingSignature_BodyNotRead verifies that unsigned requests
// are rejected before their body is read.
func (s *InboundGithubSuite) Test_Rejected_MissingSignature_BodyNotRead() {
	code, read := postWebhook(postWebhookParams{
		Target:  "/app/webhooks/github/deploy",
		Headers: map[string]string{"X-Github-Event": "push"},
	})

	s.Equal(http.StatusForbidden, code)
	s.False(read)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// Test_Rejected_NoSecretConfigured verifies that webhooks are rejected, rather
// than accepted unverified, when the instance has no webhook secret.
func (s *InboundGithubSuite) Test_Rejected_NoSecretConfigured() {
	s.app(nil)
	s.setWebhookSecret("")

	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(githubPushExample), &payload))

	response := s.push(map[string]string{
		"X-Github-Event":      "push",
		"X-Hub-Signature-256": githubSignature(payload),
	})

	s.Equal(http.StatusForbidden, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGithubSuite) app(envOverwrite map[string]any) *factory.MockApp {
	if envOverwrite == nil {
		envOverwrite = map[string]any{}
	}

	appl := s.MockApp(nil, map[string]any{
		"Repo": "github/stormkit-test-acc/test-repo",
	})

	s.MockEnv(appl, envOverwrite)

	return appl
}

func (s *InboundGithubSuite) Test_NoAutoDeploy() {
	a := assert.New(s.T())
	appl := s.app(map[string]any{
		"AutoDeploy": false,
	})

	payload := map[string]any{}
	a.NoError(json.Unmarshal([]byte(githubPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "push",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	a.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGithubSuite) Test_ShouldNotDeployTags() {
	s.app(map[string]any{
		"AutoDeploy": true,
	})

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(strings.Replace(githubPushExample, "refs/heads", "refs/tags", 1)), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/webhooks/github",
		payload,
		map[string]string{
			"X-Github-Event":      "push",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGithubSuite) Test_PushEventSuccess_BranchNameMatches() {
	a := assert.New(s.T())
	appl := s.app(nil)

	payload := map[string]any{}
	a.NoError(json.Unmarshal([]byte(githubPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "push",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	a.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return a.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return a.Equal(_depl.CheckoutRepo, "github/stormkit-test-acc/test-repo") &&
				a.Equal(true, _depl.IsAutoDeploy) &&
				a.Equal("main", _depl.Branch) &&
				a.Equal(int64(0), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
}

// githubPushPayload builds a push payload carrying the given commits.
func (s *InboundGithubSuite) githubPushPayload(commits []map[string]any) map[string]any {
	payload := map[string]any{}
	s.Require().NoError(json.Unmarshal([]byte(githubPushExample), &payload))
	payload["commits"] = commits

	return payload
}

func (s *InboundGithubSuite) githubPush(appl *factory.MockApp, payload map[string]any) int {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "push",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	return response.Code
}

func (s *InboundGithubSuite) Test_PushEvent_BuildRootUnchanged() {
	appl := s.app(map[string]any{
		"Data": &buildconf.BuildConf{
			WorkDir:                "apps/frontend",
			SkipUnchangedBuildRoot: null.BoolFrom(true),
		},
	})

	code := s.githubPush(appl, s.githubPushPayload([]map[string]any{
		{
			"message":  "Update infrastructure",
			"modified": []string{"apps/infrastructure/main.tf"},
		},
	}))

	s.Equal(http.StatusNoContent, code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

// A push inside the build root still deploys, which also proves the added,
// modified and removed paths are read off the payload.
func (s *InboundGithubSuite) Test_PushEvent_BuildRootChanged() {
	appl := s.app(map[string]any{
		"Data": &buildconf.BuildConf{
			WorkDir:                "apps/frontend",
			SkipUnchangedBuildRoot: null.BoolFrom(true),
		},
	})

	code := s.githubPush(appl, s.githubPushPayload([]map[string]any{
		{"message": "Add a page", "added": []string{"apps/frontend/page.tsx"}},
		{"message": "Rename a page", "removed": []string{"apps/frontend/old.tsx"}},
	}))

	s.Equal(http.StatusOK, code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

// GitHub caps the push payload at 2048 commits. At that point the change set is
// incomplete, so every otherwise matching candidate has to be kept.
func (s *InboundGithubSuite) Test_PushEvent_TruncatedCommitsDeploys() {
	appl := s.app(map[string]any{
		"Data": &buildconf.BuildConf{
			WorkDir:                "apps/frontend",
			SkipUnchangedBuildRoot: null.BoolFrom(true),
		},
	})

	commits := make([]map[string]any, 2048)

	for i := range commits {
		commits[i] = map[string]any{
			"message":  "Update infrastructure",
			"modified": []string{"apps/infrastructure/main.tf"},
		}
	}

	code := s.githubPush(appl, s.githubPushPayload(commits))

	s.Equal(http.StatusOK, code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

// The filter is opt-in, so an environment with a build root but no filter
// enabled keeps deploying on every push.
func (s *InboundGithubSuite) Test_PushEvent_BuildRootFilterIsOptIn() {
	appl := s.app(map[string]any{
		"Data": &buildconf.BuildConf{WorkDir: "apps/frontend"},
	})

	code := s.githubPush(appl, s.githubPushPayload([]map[string]any{
		{
			"message":  "Update infrastructure",
			"modified": []string{"apps/infrastructure/main.tf"},
		},
	}))

	s.Equal(http.StatusOK, code)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

func (s *InboundGithubSuite) Test_PushEvent_BranchNameDoesNotMatch() {
	appl := s.app(map[string]any{
		"AutoDeployBranches": null.StringFrom("should-not-exist"),
	})

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(githubPushExample), &payload))
	payload["ref"] = "refs/head/some-branch"

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "push",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGithubSuite) Test_PullRequestOpened() {
	appl := s.app(map[string]any{
		"AutoDeployBranches": null.NewString("my-pr-*", true),
	})

	repo := strings.Replace(appl.Repo, "github/", "", 1)
	payload := map[string]any{}
	params := githubMergeParams{status: "opened", merged: false, baseRepo: repo, headRepo: repo}
	s.NoError(json.Unmarshal([]byte(githubMergeExample(params)), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "pull_request",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return s.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return s.Equal(_depl.CheckoutRepo, "github/stormkit-test-acc/test-repo") &&
				s.Equal(true, _depl.IsAutoDeploy) &&
				s.Equal(int64(53), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
}

// Test_PullRequestOpened_Fork verifies that pull requests from forks are not built automatically.
func (s *InboundGithubSuite) Test_PullRequestOpened_Fork() {
	a := assert.New(s.T())
	appl := s.app(nil)

	repo := strings.Replace(appl.Repo, "github/", "", 1)
	payload := map[string]any{}
	params := githubMergeParams{status: "opened", merged: false, baseRepo: repo, headRepo: "fork-repo/test-repo"}
	a.NoError(json.Unmarshal([]byte(githubMergeExample(params)), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "pull_request",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	a.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundGithubSuite) Test_PullRequestMerged() {
	a := assert.New(s.T())
	appl := s.app(nil)

	repo := strings.Replace(appl.Repo, "github/", "", 1)
	payload := map[string]any{}
	params := githubMergeParams{status: "closed", merged: true, baseRepo: repo, headRepo: repo}
	a.NoError(json.Unmarshal([]byte(githubMergeExample(params)), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/github/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Github-Event":      "pull_request",
			"X-Hub-Signature-256": githubSignature(payload),
		},
	)

	a.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func TestInboundGithub(t *testing.T) {
	suite.Run(t, &InboundGithubSuite{})
}
