package apphandlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const bitbucketPushEvent = "repo:push"

var bitbucketMergeExample = func(status string) string {
	return fmt.Sprintf(`
	{
		"pullrequest": {
		  "id": 50,
		  "title": "[app] Update description and label",
		  "state": "%s",
		  "source": {
			"branch": {
			  "name": "sk-change-label"
			},
			"commit": {
			  "hash": "d69e33da8fb8"
			},
			"repository": {
			  "type": "repository",
			  "full_name": "stormkit-test/test-repo",
			  "is_private": false
			}
		  },
		  "destination": {
			"branch": {
			  "name": "master"
			},
			"commit": {
			  "hash": "ef2edd4a3960"
			},
			"repository": {
			  "type": "repository",
			  "uuid": "{94fa72db-b63e-4b94-a72c-71a24b4d87d5}",
			  "full_name": "stormkit-test/test-repo",
			  "name": "app-www",
			  "is_private": false
			}
		  },
		  "merge_commit": {
			"hash": ""
		  },
		  "created_on": "2019-08-06T08:41:55.577773Z",
		  "updated_on": "2019-08-06T08:41:55.620392Z"
		},
		"repository": {
		  "type": "repository",
		  "uuid": "{94fa72db-b63e-4b94-a72c-71a24b4d87d5}",
		  "full_name": "stormkit-test/test-repo",
		  "name": "app-www",
		  "website": "",
		  "scm": "git",
		  "is_private": true
		}
	}`, status)
}

const bitbucketPushExample = `{
  "repository": {
	"full_name": "stormkit-test/test-repo"
  },
  "push": {
	"changes": [
	  {
		"new": {
	      "type": "branch",
	      "name": "main",
	      "target": {
	        "type": "commit",
	        "message": "[www] commit message"
	      }
	    }
	  }
	]
  }
}`

type InboundBitbucketSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *InboundBitbucketSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *InboundBitbucketSuite) AfterTest(_, _ string) {
	deployservice.MockDeployer = nil
	s.conn.CloseTx()
}

func (s *InboundBitbucketSuite) app(autoDeploy bool) *factory.MockApp {
	appl := s.MockApp(nil, map[string]any{
		"Repo": "bitbucket/stormkit-test/test-repo",
	})

	s.MockEnv(appl, map[string]any{
		"AutoDeploy": autoDeploy,
	})

	return appl
}

func (s *InboundBitbucketSuite) Test_NoAutoDeploy() {
	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(bitbucketPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/bitbucket/%s", s.app(false).Secret()),
		payload,
		map[string]string{
			"X-Event-Key": bitbucketPushEvent,
		},
	)

	s.Equal(http.StatusNoContent, response.Code)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy")
}

func (s *InboundBitbucketSuite) Test_PushEventSuccess() {
	appl := s.app(true)

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(bitbucketPushExample), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/bitbucket/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Event-Key": bitbucketPushEvent,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything,
		mock.MatchedBy(func(a *app.App) bool {
			return s.Equal(appl.ID, a.ID)
		}),
		mock.MatchedBy(func(d *deploy.Deployment) bool {
			return s.Equal(d.CheckoutRepo, "bitbucket/stormkit-test/test-repo") &&
				s.Equal(true, d.IsAutoDeploy) &&
				s.Equal("main", d.Branch) &&
				s.Equal(int64(0), d.PullRequestNumber.ValueOrZero())
		}),
	)
}

func (s *InboundBitbucketSuite) Test_MergeRequestOpened() {
	appl := s.app(true)

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(bitbucketMergeExample("created")), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/bitbucket/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Event-Key": "pullrequest:created",
		},
	)

	s.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything,
		mock.MatchedBy(func(_appl *app.App) bool {
			return s.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return s.Equal(_depl.CheckoutRepo, "bitbucket/stormkit-test/test-repo") &&
				s.Equal(true, _depl.IsAutoDeploy) &&
				s.Equal(int64(50), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
}

func (s *InboundBitbucketSuite) Test_MergeRequestMerged() {
	appl := s.app(true)

	payload := map[string]any{}
	s.NoError(json.Unmarshal([]byte(bitbucketMergeExample("fulfilled")), &payload))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		fmt.Sprintf("/app/webhooks/bitbucket/%s", appl.Secret()),
		payload,
		map[string]string{
			"X-Event-Key": "pullrequest:fulfilled",
		},
	)

	s.Equal(http.StatusOK, response.Code)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything,
		mock.MatchedBy(func(_appl *app.App) bool {
			return s.Equal(appl.ID, _appl.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return s.Equal(_depl.CheckoutRepo, "bitbucket/stormkit-test/test-repo") &&
				s.Equal(true, _depl.IsAutoDeploy) &&
				s.Equal(int64(0), _depl.PullRequestNumber.ValueOrZero())
		}),
	)
}

func TestInboundBitbucket(t *testing.T) {
	suite.Run(t, &InboundBitbucketSuite{})
}
