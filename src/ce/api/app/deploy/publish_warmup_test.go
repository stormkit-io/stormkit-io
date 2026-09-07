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

	ctx   context.Context
	store deploy.PublishJobStore
}

func (s *PublishWarmupSuite) SetupSuite() {
	s.ctx = context.Background()
	s.store = deploy.NewPublishJobStore()
}

// newJob returns a job with a unique id so tests never collide in a shared Redis.
func (s *PublishWarmupSuite) newJob() *deploy.PublishJob {
	return &deploy.PublishJob{
		ID:           uuid.New().String(),
		AppID:        types.ID(1),
		EnvID:        types.ID(time.Now().UnixNano()),
		DeploymentID: types.ID(42),
		Nodes:        []string{"node-a", "node-b"},
		Status:       deploy.PublishStatusPublishing,
		StartedAt:    time.Now().Unix(),
		Deadline:     time.Now().Add(3 * time.Minute).Unix(),
	}
}

func (s *PublishWarmupSuite) Test_SaveAndLoad() {
	job := s.newJob()

	s.NoError(s.store.Create(s.ctx, job))

	loaded, err := s.store.Load(s.ctx, job.ID)
	s.NoError(err)
	s.Require().NotNil(loaded)
	s.Equal(job.ID, loaded.ID)
	s.Equal(deploy.PublishStatusPublishing, loaded.Status)
	s.Equal([]string{"node-a", "node-b"}, loaded.Nodes)
	s.Equal(types.ID(42), loaded.DeploymentID)
}

func (s *PublishWarmupSuite) Test_Load_UnknownJob() {
	loaded, err := s.store.Load(s.ctx, uuid.New().String())

	s.NoError(err)
	s.Nil(loaded)
}

func (s *PublishWarmupSuite) Test_LoadByEnv_ReturnsLatestJob() {
	first := s.newJob()
	second := s.newJob()
	second.EnvID = first.EnvID

	s.NoError(s.store.Create(s.ctx, first))
	s.NoError(s.store.Create(s.ctx, second))

	loaded, err := s.store.LoadByEnv(s.ctx, first.EnvID)
	s.NoError(err)
	s.Require().NotNil(loaded)
	s.Equal(second.ID, loaded.ID)
}

// Test_NodeResults_MissingNodeIsAbsent guards the core safety rule: a node that
// has not reported must never be mistaken for a node that reported success.
func (s *PublishWarmupSuite) Test_NodeResults_MissingNodeIsAbsent() {
	job := s.newJob()
	s.NoError(s.store.Create(s.ctx, job))

	s.NoError(s.store.SaveNodeResult(s.ctx, job, deploy.WarmupResult{
		ServiceID:   "node-a",
		ServiceName: rediscache.ServiceHosting,
		Status:      rediscache.StatusOK,
	}))

	results := s.store.NodeResults(s.ctx, job)

	s.Len(results, 1)
	s.Equal(rediscache.StatusOK, results["node-a"].Status)

	_, reported := results["node-b"]
	s.False(reported)
}

func (s *PublishWarmupSuite) Test_NodeResults_CarriesFailureDetail() {
	job := s.newJob()
	s.NoError(s.store.Create(s.ctx, job))

	s.NoError(s.store.SaveNodeResult(s.ctx, job, deploy.WarmupResult{
		ServiceID:    "node-a",
		ServiceName:  rediscache.ServiceHosting,
		Status:       rediscache.StatusErr,
		DeploymentID: types.ID(42),
		StatusCode:   502,
		Reason:       "timed out after 180s",
		Logs:         "Error: connect ECONNREFUSED",
	}))

	result := s.store.NodeResults(s.ctx, job)["node-a"]

	s.Equal(rediscache.StatusErr, result.Status)
	s.Equal(502, result.StatusCode)
	s.Equal("timed out after 180s", result.Reason)
	s.Contains(result.Logs, "ECONNREFUSED")
}

func (s *PublishWarmupSuite) Test_Lock_IsExclusiveUntilReleased() {
	job := s.newJob()

	token, ok := s.store.Lock(s.ctx, job.ID)
	s.True(ok)
	s.NotEmpty(token)

	_, ok = s.store.Lock(s.ctx, job.ID)
	s.False(ok)

	s.store.Unlock(s.ctx, job.ID, token)

	token, ok = s.store.Lock(s.ctx, job.ID)
	s.True(ok)
	s.store.Unlock(s.ctx, job.ID, token)
}

// Test_Unlock_OnlyReleasesOwnLock covers a finalizer that outran the lock's
// expiry: it must not release the lock a second finalizer has since taken, or
// the same publish would be committed twice.
func (s *PublishWarmupSuite) Test_Unlock_OnlyReleasesOwnLock() {
	job := s.newJob()

	stale, ok := s.store.Lock(s.ctx, job.ID)
	s.Require().True(ok)

	// Simulate the first holder's lock expiring and a second one taking over.
	s.store.Unlock(s.ctx, job.ID, stale)

	current, ok := s.store.Lock(s.ctx, job.ID)
	s.Require().True(ok)

	defer s.store.Unlock(s.ctx, job.ID, current)

	s.store.Unlock(s.ctx, job.ID, stale)

	_, ok = s.store.Lock(s.ctx, job.ID)
	s.False(ok, "the second holder's lock must survive the first holder's release")
}

// Test_Update_DoesNotStealTheEnvironmentPointer covers an older job recording
// its outcome after a newer publish has already started.
func (s *PublishWarmupSuite) Test_Update_DoesNotStealTheEnvironmentPointer() {
	first := s.newJob()
	second := s.newJob()
	second.EnvID = first.EnvID

	s.NoError(s.store.Create(s.ctx, first))
	s.NoError(s.store.Create(s.ctx, second))

	first.Status = deploy.PublishStatusFailed
	s.NoError(s.store.Update(s.ctx, first))

	loaded, err := s.store.LoadByEnv(s.ctx, first.EnvID)
	s.NoError(err)
	s.Require().NotNil(loaded)
	s.Equal(second.ID, loaded.ID)
}

func (s *PublishWarmupSuite) Test_SaveNodeResult_RejectsAnUnidentifiedNode() {
	job := s.newJob()
	s.NoError(s.store.Create(s.ctx, job))

	s.Error(s.store.SaveNodeResult(s.ctx, job, deploy.WarmupResult{Status: rediscache.StatusOK}))
}

// Test_SaveNodeResult_BoundsWhatANodeReports keeps a crash-looping server from
// writing unbounded output into Redis on every publish.
func (s *PublishWarmupSuite) Test_SaveNodeResult_BoundsWhatANodeReports() {
	job := s.newJob()
	s.NoError(s.store.Create(s.ctx, job))

	s.NoError(s.store.SaveNodeResult(s.ctx, job, deploy.WarmupResult{
		ServiceID: "node-a",
		Status:    rediscache.StatusErr,
		Logs:      strings.Repeat("x", 40_000),
		Reason:    strings.Repeat("y", 4_000),
	}))

	result := s.store.NodeResults(s.ctx, job)["node-a"]

	s.Len(result.Logs, 8*1024)
	s.Len(result.Reason, 1024)
}

// Test_NodeResults_OutliveTheJobDeadline guards a job whose warm-up window is
// longer than an hour: a node that already succeeded must not read back as
// pending and fail an otherwise healthy publish.
func (s *PublishWarmupSuite) Test_NodeResults_OutliveTheJobDeadline() {
	job := s.newJob()
	job.Deadline = time.Now().Add(4 * time.Hour).Unix()

	s.NoError(s.store.Create(s.ctx, job))
	s.NoError(s.store.SaveNodeResult(s.ctx, job, deploy.WarmupResult{
		ServiceID: "node-a",
		Status:    rediscache.StatusOK,
	}))

	ttl, err := rediscache.Client().TTL(s.ctx, "publish:job:"+job.ID+":node:node-a").Result()
	s.NoError(err)
	s.Greater(ttl, 4*time.Hour)
}

// Test_JobIDs_SkipsNodeAndLockKeys verifies the sweeper is not handed the
// per-node and lock keys, which live under the same prefix.
func (s *PublishWarmupSuite) Test_JobIDs_SkipsNodeAndLockKeys() {
	job := s.newJob()

	s.NoError(s.store.Create(s.ctx, job))
	s.NoError(s.store.SaveNodeResult(s.ctx, job, deploy.WarmupResult{ServiceID: "node-a"}))
	token, ok := s.store.Lock(s.ctx, job.ID)
	s.True(ok)

	defer s.store.Unlock(s.ctx, job.ID, token)

	ids, err := s.store.JobIDs(s.ctx)
	s.NoError(err)
	s.Contains(ids, job.ID)

	for _, id := range ids {
		s.False(strings.Contains(id, ":"), "job id %q must not contain a sub-key separator", id)
	}
}

func (s *PublishWarmupSuite) Test_IsTerminal() {
	job := s.newJob()
	s.False(job.IsTerminal())

	job.Status = deploy.PublishStatusPublished
	s.True(job.IsTerminal())

	job.Status = deploy.PublishStatusFailed
	s.True(job.IsTerminal())
}

// Test_Settings_PublishesTheDeploymentInFull verifies a job always resolves to
// a single deployment at 100%: percentage-based releases are retired.
func (s *PublishWarmupSuite) Test_Settings_PublishesTheDeploymentInFull() {
	job := s.newJob()

	settings := job.Settings()

	s.Require().Len(settings, 1)
	s.Equal(job.EnvID, settings[0].EnvID)
	s.Equal(types.ID(42), settings[0].DeploymentID)
	s.Equal(float64(100), settings[0].Percentage)
}

func (s *PublishWarmupSuite) Test_TruncateWarmupLogs_KeepsTail() {
	short := "boot failed"
	s.Equal(short, deploy.TruncateWarmupLogs(short))

	long := strings.Repeat("a", 9000) + "TAIL"
	truncated := deploy.TruncateWarmupLogs(long)

	s.Len(truncated, 8*1024)
	s.True(strings.HasSuffix(truncated, "TAIL"))
}

func TestPublishWarmupSuite(t *testing.T) {
	suite.Run(t, new(PublishWarmupSuite))
}
