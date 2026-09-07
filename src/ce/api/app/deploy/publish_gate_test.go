package deploy_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

// PublishGateSuite drives the gate on its own goroutine, the way production
// runs it. WarmUpAndPublish takes an inline shortcut under test so that
// detached work cannot write to a rolled-back transaction, which means nothing
// else here exercises the path that actually ships.
type PublishGateSuite struct {
	suite.Suite

	ctx     context.Context
	service *fakeService
}

func (s *PublishGateSuite) SetupTest() {
	s.ctx = context.Background()
	s.service = &fakeService{id: uuid.New().String()}

	rediscache.DefaultService = s.service
}

func (s *PublishGateSuite) TearDownTest() {
	rediscache.DefaultService = nil
}

func (s *PublishGateSuite) params(envID types.ID) deploy.WarmUpAndPublishParams {
	return deploy.WarmUpAndPublishParams{
		AppID:        types.ID(1),
		EnvID:        envID,
		DeploymentID: types.ID(42),
		DisplayName:  "my-app",
		EnvName:      "production",
		Timeout:      5 * time.Second,
	}
}

// envID returns an environment nothing else in the suite touches.
func (s *PublishGateSuite) envID() types.ID {
	return types.ID(time.Now().UnixNano())
}

// awaitStatus waits for the environment to reach a terminal publish status.
func (s *PublishGateSuite) awaitStatus(envID types.ID) *deploy.PublishStatus {
	for range 200 {
		status, err := deploy.PublishStatusOf(s.ctx, envID)
		s.Require().NoError(err)

		if status != nil && status.Status != deploy.PublishStatusPublishing {
			return status
		}

		time.Sleep(25 * time.Millisecond)
	}

	return nil
}

// Test_PublishesOnceEveryNodeReports covers the whole path: the broadcast, the
// wait, a node reporting, and the flip.
func (s *PublishGateSuite) Test_PublishesOnceEveryNodeReports() {
	envID := s.envID()
	published := make(chan struct{}, 1)

	s.service.nodes = []string{"node-a", "node-b"}
	s.service.onBroadcast = func(warmupID string) {
		for _, node := range []string{"node-a", "node-b"} {
			s.NoError(deploy.SaveNodeResult(context.Background(), deploy.SaveNodeResultParams{
				WarmupID: warmupID,
				Deadline: time.Now().Add(time.Minute).Unix(),
				Result:   deploy.WarmupResult{ServiceID: node, Status: rediscache.StatusOK},
			}))
		}
	}

	deploy.WarmUpAndPublishAsyncForTest(s.ctx, s.params(envID), func() error {
		published <- struct{}{}
		return nil
	})

	select {
	case <-published:
	case <-time.After(10 * time.Second):
		s.Fail("the publish never happened")
	}

	status := s.awaitStatus(envID)
	s.Require().NotNil(status)
	s.Equal(deploy.PublishStatusPublished, status.Status)
}

// Test_DoesNotPublishWhenANodeFails is the guarantee the feature exists for:
// traffic must not move to a deployment that did not come up.
func (s *PublishGateSuite) Test_DoesNotPublishWhenANodeFails() {
	envID := s.envID()
	published := make(chan struct{}, 1)

	s.service.nodes = []string{"node-a"}
	s.service.onBroadcast = func(warmupID string) {
		s.NoError(deploy.SaveNodeResult(context.Background(), deploy.SaveNodeResultParams{
			WarmupID: warmupID,
			Deadline: time.Now().Add(time.Minute).Unix(),
			Result: deploy.WarmupResult{
				ServiceID: "node-a",
				Status:    rediscache.StatusErr,
				Reason:    "the deployment did not serve a request within 3m0s",
			},
		}))
	}

	deploy.WarmUpAndPublishAsyncForTest(s.ctx, s.params(envID), func() error {
		published <- struct{}{}
		return nil
	})

	status := s.awaitStatus(envID)
	s.Require().NotNil(status)
	s.Equal(deploy.PublishStatusFailed, status.Status)
	s.Contains(status.Reason, "did not serve a request")

	select {
	case <-published:
		s.Fail("a deployment that never came up must not be published")
	default:
	}
}

// Test_PublishesWithoutHostingNodes keeps a fresh install, where nothing is
// serving yet, publishing as it did before.
func (s *PublishGateSuite) Test_PublishesWithoutHostingNodes() {
	envID := s.envID()
	published := make(chan struct{}, 1)

	s.service.nodes = nil

	deploy.WarmUpAndPublishAsyncForTest(s.ctx, s.params(envID), func() error {
		published <- struct{}{}
		return nil
	})

	select {
	case <-published:
	case <-time.After(5 * time.Second):
		s.Fail("a publish with nothing to warm up should go straight through")
	}
}

// Test_SupersededPublishIsSkipped covers two publishes racing: the slower one
// must not roll the environment back when it finishes second.
func (s *PublishGateSuite) Test_SupersededPublishIsSkipped() {
	envID := s.envID()
	published := make(chan struct{}, 1)

	s.service.nodes = []string{"node-a"}
	s.service.onBroadcast = func(warmupID string) {
		// A newer publish for the same environment starts while this one is
		// still warming up.
		deploy.RecordIntentForTest(context.Background(), envID, uuid.New().String())

		s.NoError(deploy.SaveNodeResult(context.Background(), deploy.SaveNodeResultParams{
			WarmupID: warmupID,
			Deadline: time.Now().Add(time.Minute).Unix(),
			Result:   deploy.WarmupResult{ServiceID: "node-a", Status: rediscache.StatusOK},
		}))
	}

	deploy.WarmUpAndPublishAsyncForTest(s.ctx, s.params(envID), func() error {
		published <- struct{}{}
		return nil
	})

	select {
	case <-published:
		s.Fail("a superseded publish must not move the environment")
	case <-time.After(2 * time.Second):
	}
}

func TestPublishGateSuite(t *testing.T) {
	suite.Run(t, new(PublishGateSuite))
}
