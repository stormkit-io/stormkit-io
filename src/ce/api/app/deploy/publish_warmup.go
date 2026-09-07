package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// Publish job statuses. A job is terminal once it is published or failed.
const (
	PublishStatusPublishing = "publishing"
	PublishStatusPublished  = "published"
	PublishStatusFailed     = "failed"
)

// maxWarmupLogBytes caps how much of a failing deployment's boot output is
// carried back to the user, so a server that logs in a loop cannot fill Redis.
const maxWarmupLogBytes = 8 * 1024

// maxWarmupReasonBytes caps the human-readable failure reason a node reports.
const maxWarmupReasonBytes = 1024

// finalizeLockTTL has to outlast a finalization, which writes the published
// rows, resets the hosting cache and dispatches the publish webhooks.
const finalizeLockTTL = 2 * time.Minute

// unlockScript releases a lock only when the caller still owns it.
var unlockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	end

	return 0
`)

// publishedPercentage is what every published deployment gets. Percentage-based
// releases are a retired feature: PublishSettings still carries the field
// because the stored column and the public API do, but nothing chooses a value
// other than this one.
const publishedPercentage = 100

// WarmupRequest is what a hosting node receives over the event bus. It carries
// the app's display name because a config can only be resolved for an
// unpublished deployment by deployment id *and* display name, and the display
// name is not part of the config itself.
type WarmupRequest struct {
	JobID        string   `json:"jobId"`
	AppID        types.ID `json:"appId,string"`
	EnvID        types.ID `json:"envId,string"`
	DisplayName  string   `json:"displayName"`
	DeploymentID types.ID `json:"deploymentId,string"`
	Deadline     int64    `json:"deadline"`
}

// DeadlineAt returns the moment after which a node must stop probing.
func (r WarmupRequest) DeadlineAt() time.Time {
	return time.Unix(r.Deadline, 0)
}

// WarmupResult is one hosting node's verdict for a publish job.
type WarmupResult struct {
	ServiceID    string   `json:"serviceId"`
	ServiceName  string   `json:"serviceName"`
	Status       string   `json:"status"`
	DeploymentID types.ID `json:"deploymentId,string,omitempty"`
	StatusCode   int      `json:"statusCode,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Logs         string   `json:"logs,omitempty"`
	Skipped      bool     `json:"skipped,omitempty"`
}

// PublishJob is the record a publish request creates before any traffic moves.
// It is the single source of truth for "is a publish in flight, and did it
// work" — the deployments_published table is only written once the job
// succeeds.
type PublishJob struct {
	ID           string         `json:"id"`
	AppID        types.ID       `json:"appId,string"`
	EnvID        types.ID       `json:"envId,string"`
	DeploymentID types.ID       `json:"deploymentId,string"`
	Nodes        []string       `json:"nodes"`
	Status       string         `json:"status"`
	Reason       string         `json:"reason,omitempty"`
	Failures     []WarmupResult `json:"failures,omitempty"`
	StartedAt    int64          `json:"startedAt"`
	Deadline     int64          `json:"deadline"`
}

// IsTerminal reports whether the job has already been decided.
func (j *PublishJob) IsTerminal() bool {
	return j.Status == PublishStatusPublished || j.Status == PublishStatusFailed
}

// DeadlineAt returns the moment after which the job can no longer succeed.
func (j *PublishJob) DeadlineAt() time.Time {
	return time.Unix(j.Deadline, 0)
}

// Settings turns the job into the arguments the publish itself needs.
func (j *PublishJob) Settings() []*PublishSettings {
	return []*PublishSettings{
		{
			EnvID:        j.EnvID,
			DeploymentID: j.DeploymentID,
			Percentage:   publishedPercentage,
		},
	}
}

// TruncateWarmupLogs trims boot output to what is worth storing and showing.
// The tail is kept: the last thing a crashing server printed is the useful
// part.
func TruncateWarmupLogs(logs string) string {
	if len(logs) <= maxWarmupLogBytes {
		return logs
	}

	return logs[len(logs)-maxWarmupLogBytes:]
}

// truncateReason bounds a node's failure reason.
func truncateReason(reason string) string {
	if len(reason) <= maxWarmupReasonBytes {
		return reason
	}

	return reason[:maxWarmupReasonBytes]
}

// PublishJobStore persists publish jobs and the per-node warm-up results.
//
// The state is deliberately transient: a lost job means the environment keeps
// serving whatever it served before, which is the safe outcome.
type PublishJobStore struct{}

// NewPublishJobStore returns the store used to read and write publish jobs.
func NewPublishJobStore() PublishJobStore {
	return PublishJobStore{}
}

func (PublishJobStore) jobKey(jobID string) string {
	return fmt.Sprintf("publish:job:%s", jobID)
}

func (PublishJobStore) nodeKey(jobID, serviceID string) string {
	return fmt.Sprintf("publish:job:%s:node:%s", jobID, serviceID)
}

func (PublishJobStore) lockKey(jobID string) string {
	return fmt.Sprintf("publish:job:%s:lock", jobID)
}

func (PublishJobStore) envKey(envID types.ID) string {
	return fmt.Sprintf("publish:env:%s", envID.String())
}

// ttl keeps a job readable for a while after it is decided, so the dashboard
// can still show why a publish failed.
func (PublishJobStore) ttl(job *PublishJob) time.Duration {
	return max(time.Until(job.DeadlineAt()), 0) + time.Hour
}

// Create writes a new job and points its environment at it.
//
// Only creation moves the environment pointer. An older job recording its
// outcome must not steal the pointer from a publish that started after it.
func (s PublishJobStore) Create(ctx context.Context, job *PublishJob) error {
	client := rediscache.Client()

	if client == nil {
		return nil
	}

	if err := s.Update(ctx, job); err != nil {
		return err
	}

	return client.Set(ctx, s.envKey(job.EnvID), job.ID, s.ttl(job)).Err()
}

// Update writes the job's current state, leaving the environment pointer alone.
func (s PublishJobStore) Update(ctx context.Context, job *PublishJob) error {
	client := rediscache.Client()

	if client == nil {
		return nil
	}

	data, err := json.Marshal(job)

	if err != nil {
		return err
	}

	return client.Set(ctx, s.jobKey(job.ID), data, s.ttl(job)).Err()
}

// Load returns the job, or nil when there is no such job.
//
// A Redis failure is reported as an error rather than as a missing job: a
// caller that mistook one for the other would abandon a publish that is still
// in flight, leaving it neither committed nor marked failed.
func (s PublishJobStore) Load(ctx context.Context, jobID string) (*PublishJob, error) {
	client := rediscache.Client()

	if client == nil {
		return nil, nil
	}

	data, err := client.Get(ctx, s.jobKey(jobID)).Bytes()

	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	job := &PublishJob{}

	if err := json.Unmarshal(data, job); err != nil {
		return nil, err
	}

	return job, nil
}

// LoadByEnv returns the most recent job of an environment, or nil when it has
// never published. As with Load, a Redis failure is an error and not an
// absence: callers use this to decide whether a publish is already running.
func (s PublishJobStore) LoadByEnv(ctx context.Context, envID types.ID) (*PublishJob, error) {
	client := rediscache.Client()

	if client == nil {
		return nil, nil
	}

	jobID, err := client.Get(ctx, s.envKey(envID)).Result()

	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if jobID == "" {
		return nil, nil
	}

	return s.Load(ctx, jobID)
}

// SaveNodeResult records one hosting node's verdict.
//
// The size caps are applied here rather than by the caller: this is where data
// reported by a hosting node enters Redis, and a server that crash-loops with
// noisy output must not be able to write an unbounded value.
func (s PublishJobStore) SaveNodeResult(ctx context.Context, job *PublishJob, result WarmupResult) error {
	if result.ServiceID == "" {
		return errors.New("cannot record a warm-up result without a service id")
	}

	client := rediscache.Client()

	if client == nil {
		return nil
	}

	result.Logs = TruncateWarmupLogs(result.Logs)
	result.Reason = truncateReason(result.Reason)

	data, err := json.Marshal(result)

	if err != nil {
		return err
	}

	// The verdict has to outlive the job that is waiting for it, or a node that
	// already succeeded reads back as pending and the publish fails at its
	// deadline.
	return client.Set(ctx, s.nodeKey(job.ID, result.ServiceID), data, s.ttl(job)).Err()
}

// NodeResults returns the verdicts reported so far, keyed by service id. A
// service that has not reported is absent from the map — never assume silence
// means success.
func (s PublishJobStore) NodeResults(ctx context.Context, job *PublishJob) map[string]WarmupResult {
	results := map[string]WarmupResult{}
	client := rediscache.Client()

	if client == nil {
		return results
	}

	for _, serviceID := range job.Nodes {
		data, err := client.Get(ctx, s.nodeKey(job.ID, serviceID)).Bytes()

		// A node that cannot be read counts as pending, which fails the publish
		// safely rather than committing one that was never verified. Log the
		// real errors, or a Redis hiccup near the deadline would fail a healthy
		// publish with nothing to explain it.
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				slog.Errorf("cannot read warm-up result for job %s node %s: %s", job.ID, serviceID, err.Error())
			}

			continue
		}

		result := WarmupResult{}

		if err := json.Unmarshal(data, &result); err != nil {
			slog.Errorf("cannot decode warm-up result for job %s node %s: %s", job.ID, serviceID, err.Error())
			continue
		}

		results[serviceID] = result
	}

	return results
}

// Lock takes the finalization lock for a job. The returned token identifies
// this holder and must be handed back to Unlock. A false second return means
// another process holds the lock and the caller must do nothing: the holder is
// about to decide the same job.
func (s PublishJobStore) Lock(ctx context.Context, jobID string) (string, bool) {
	token := uuid.New().String()
	client := rediscache.Client()

	if client == nil {
		return token, true
	}

	ok, err := client.SetNX(ctx, s.lockKey(jobID), token, finalizeLockTTL).Result()

	if err != nil || !ok {
		return "", false
	}

	return token, true
}

// Unlock releases the finalization lock, but only if this caller still holds
// it.
//
// Finalizing writes to the database, resets the hosting cache and dispatches
// publish webhooks, so it can outrun the lock's expiry. Without the token check
// a slow finalizer would delete the lock a second one had since taken, and the
// same publish would be committed twice.
func (s PublishJobStore) Unlock(ctx context.Context, jobID, token string) {
	client := rediscache.Client()

	if client == nil {
		return
	}

	if err := unlockScript.Run(ctx, client, []string{s.lockKey(jobID)}, token).Err(); err != nil && !errors.Is(err, redis.Nil) {
		slog.Errorf("cannot release publish lock for job %s: %s", jobID, err.Error())
	}
}

// JobIDs returns every job currently held in Redis, decided or not.
func (s PublishJobStore) JobIDs(ctx context.Context) ([]string, error) {
	client := rediscache.Client()

	if client == nil {
		return nil, nil
	}

	keys, err := client.Keys(ctx, "publish:job:*")

	if err != nil {
		return nil, err
	}

	ids := []string{}

	for _, key := range keys {
		// Skip the per-node and lock keys, which share the prefix.
		if len(key) > len("publish:job:") && !hasSubKeySuffix(key) {
			ids = append(ids, key[len("publish:job:"):])
		}
	}

	return ids, nil
}

// hasSubKeySuffix reports whether a key under the job prefix belongs to a node
// result or a lock rather than to a job itself.
func hasSubKeySuffix(key string) bool {
	for i := len("publish:job:"); i < len(key); i++ {
		if key[i] == ':' {
			return true
		}
	}

	return false
}
