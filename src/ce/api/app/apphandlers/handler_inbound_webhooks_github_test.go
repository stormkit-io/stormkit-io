package apphandlers_test

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
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

// githubMac computes the hash value with the github secret. It is used
// to authenticate a request.
func githubMac(payload map[string]any) hash.Hash {
	// Need to write the body by unmarshaling and marshaling to make sure
	// that it mimics the request and the whitespaces do not create a problem.
	mac := hmac.New(sha1.New, []byte("random-token"))
	body, _ := json.Marshal(payload)
	_, _ = mac.Write(body)
	return mac
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
}

func (s *InboundGithubSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	deployservice.MockDeployer = nil
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
			"X-Github-Event":  "push",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
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
			"X-Github-Event":  "push",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
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
			"X-Github-Event":  "push",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
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
			"X-Github-Event":  "push",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
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
			"X-Github-Event":  "pull_request",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
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
			"X-Github-Event":  "pull_request",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
		},
	)

	a.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_appl *app.App) bool {
			return a.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return a.Equal(_depl.CheckoutRepo, "github/fork-repo/test-repo") &&
				a.Equal(true, _depl.IsAutoDeploy) &&
				a.Equal(int64(53), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
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
			"X-Github-Event":  "pull_request",
			"X-Hub-Signature": fmt.Sprintf("sha1=%s", hex.EncodeToString(githubMac(payload).Sum(nil))),
		},
	)

	a.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func TestInboundGithub(t *testing.T) {
	suite.Run(t, &InboundGithubSuite{})
}
