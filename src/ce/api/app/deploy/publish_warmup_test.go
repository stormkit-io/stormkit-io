package deploy_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type PublishWarmupSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *PublishWarmupSuite) SetupSuite() {
	s.ctx = context.Background()
}

// envID returns an environment nothing else in the suite touches.
func (s *PublishWarmupSuite) envID() types.ID {
	return types.ID(time.Now().UnixNano())
}

func (s *PublishWarmupSuite) Test_SaveNodeResult_RejectsAnUnidentifiedNode() {
	err := deploy.SaveNodeResult(s.ctx, deploy.SaveNodeResultParams{
		WarmupID: uuid.New().String(),
		Deadline: time.Now().Add(time.Minute).Unix(),
		Result:   deploy.WarmupResult{Status: rediscache.StatusOK},
	})

	s.Error(err)
}

// Test_SaveNodeResult_BoundsWhatANodeReports keeps a node from writing an
// unbounded value into Redis.
func (s *PublishWarmupSuite) Test_SaveNodeResult_BoundsWhatANodeReports() {
	warmupID := uuid.New().String()

	s.NoError(deploy.SaveNodeResult(s.ctx, deploy.SaveNodeResultParams{
		WarmupID: warmupID,
		Deadline: time.Now().Add(time.Minute).Unix(),
		Result: deploy.WarmupResult{
			ServiceID: "node-a",
			Status:    rediscache.StatusErr,
			Reason:    strings.Repeat("y", 4_000),
		},
	}))

	result := deploy.NodeResultsForTest(s.ctx, warmupID, []string{"node-a"})["node-a"]

	s.Len(result.Reason, 1024)
}

// Test_NodeResults_MissingNodeIsAbsent guards the core safety rule: a node that
// has not reported must never be mistaken for one that reported success.
func (s *PublishWarmupSuite) Test_NodeResults_MissingNodeIsAbsent() {
	warmupID := uuid.New().String()

	s.NoError(deploy.SaveNodeResult(s.ctx, deploy.SaveNodeResultParams{
		WarmupID: warmupID,
		Deadline: time.Now().Add(time.Minute).Unix(),
		Result: deploy.WarmupResult{
			ServiceID:   "node-a",
			ServiceName: rediscache.ServiceHosting,
			Status:      rediscache.StatusOK,
		},
	}))

	results := deploy.NodeResultsForTest(s.ctx, warmupID, []string{"node-a", "node-b"})

	s.Len(results, 1)
	s.Equal(rediscache.StatusOK, results["node-a"].Status)

	_, reported := results["node-b"]
	s.False(reported)
}

// Test_NodeResults_OutliveTheDeadline guards a warm-up window longer than an
// hour: a node that already succeeded must not read back as pending.
func (s *PublishWarmupSuite) Test_NodeResults_OutliveTheDeadline() {
	warmupID := uuid.New().String()
	deadline := time.Now().Add(4 * time.Hour).Unix()

	s.NoError(deploy.SaveNodeResult(s.ctx, deploy.SaveNodeResultParams{
		WarmupID: warmupID,
		Deadline: deadline,
		Result:   deploy.WarmupResult{ServiceID: "node-a", Status: rediscache.StatusOK},
	}))

	ttl, err := rediscache.Client().TTL(s.ctx, "publish:warmup:"+warmupID+":node:node-a").Result()
	s.NoError(err)
	s.Greater(ttl, 4*time.Hour)
}

func (s *PublishWarmupSuite) Test_PublishStatus_RoundTrip() {
	envID := s.envID()

	status, err := deploy.PublishStatusOf(s.ctx, envID)
	s.NoError(err)
	s.Nil(status, "an environment that never published has no status")

	deploy.SetPublishStatusForTest(s.ctx, envID, deploy.PublishStatus{
		Status:       deploy.PublishStatusFailed,
		DeploymentID: types.ID(42),
		Reason:       "the deployment answered with 502",
	})

	status, err = deploy.PublishStatusOf(s.ctx, envID)
	s.NoError(err)
	s.Require().NotNil(status)
	s.Equal(deploy.PublishStatusFailed, status.Status)
	s.Equal(types.ID(42), status.DeploymentID)
	s.Contains(status.Reason, "502")
	s.NotZero(status.UpdatedAt)
}

// Test_Intent_OnlyTheNewestPublishApplies covers two publishes racing: the
// slower one must not roll the environment back when it finishes second.
func (s *PublishWarmupSuite) Test_Intent_OnlyTheNewestPublishApplies() {
	envID := s.envID()
	first := uuid.New().String()
	second := uuid.New().String()

	deploy.RecordIntentForTest(s.ctx, envID, first)

	newest, err := deploy.IsNewestIntentForTest(s.ctx, envID, first)
	s.NoError(err)
	s.True(newest)

	deploy.RecordIntentForTest(s.ctx, envID, second)

	newest, err = deploy.IsNewestIntentForTest(s.ctx, envID, first)
	s.NoError(err)
	s.False(newest)

	newest, err = deploy.IsNewestIntentForTest(s.ctx, envID, second)
	s.NoError(err)
	s.True(newest)
}

// Test_Intent_UnknownEnvironmentIsNewest keeps a publish for an environment
// that has never published from being treated as superseded.
func (s *PublishWarmupSuite) Test_Intent_UnknownEnvironmentIsNewest() {
	newest, err := deploy.IsNewestIntentForTest(s.ctx, s.envID(), uuid.New().String())

	s.NoError(err)
	s.True(newest)
}

// Test_FlipLock_WaitsForTheHolderRatherThanDiscarding covers a publish that
// lost the race: it must not throw away a deployment that already warmed up.
func (s *PublishWarmupSuite) Test_FlipLock_WaitsForTheHolderRatherThanDiscarding() {
	envID := s.envID()

	held, err := deploy.AcquireFlipLockForTest(s.ctx, envID)
	s.Require().NoError(err)

	go func() {
		time.Sleep(300 * time.Millisecond)
		deploy.ReleaseFlipLockForTest(s.ctx, envID, held)
	}()

	token, err := deploy.AcquireFlipLockForTest(s.ctx, envID)

	s.NoError(err)
	s.NotEmpty(token)

	deploy.ReleaseFlipLockForTest(s.ctx, envID, token)
}

func (s *PublishWarmupSuite) Test_PublishSettingsFor_PublishesInFull() {
	settings := deploy.PublishSettingsFor(types.ID(7), types.ID(42))

	s.Require().Len(settings, 1)
	s.Equal(types.ID(7), settings[0].EnvID)
	s.Equal(types.ID(42), settings[0].DeploymentID)
	s.Equal(float64(100), settings[0].Percentage)
}

func TestPublishWarmupSuite(t *testing.T) {
	suite.Run(t, new(PublishWarmupSuite))
}
