package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// Publish statuses reported back to whoever asked for the publish.
const (
	PublishStatusPublishing = "publishing"
	PublishStatusPublished  = "published"
	PublishStatusFailed     = "failed"
)

const (
	// maxWarmupReasonBytes caps the human-readable failure reason a node
	// reports.
	maxWarmupReasonBytes = 1024

	// defaultWarmupTimeout is how long every hosting node together has to get
	// the deployment answering. Runtime dependencies are normally installed
	// during the build, so the long case is a deployment whose nix profile was
	// collected in the meantime.
	defaultWarmupTimeout = 180 * time.Second

	// warmupPollInterval is how often the waiter re-reads the node verdicts.
	warmupPollInterval = 500 * time.Millisecond

	// liveNodeCacheTTL is how long the waiter reuses its view of which hosting
	// nodes are registered.
	liveNodeCacheTTL = 5 * time.Second

	// warmupReportGrace is how long the waiter keeps reading after the nodes'
	// own deadline, so it sees the verdict a node writes when it gives up.
	warmupReportGrace = 2 * time.Second

	// publishStatusTTL keeps a finished publish readable long enough for the
	// dashboard to explain a failure.
	publishStatusTTL = time.Hour

	// flipLockWait is how long a publish waits for another one to finish
	// applying before giving up.
	flipLockWait = 30 * time.Second

	// flipLockRetryInterval is how often the flip lock is retried.
	flipLockRetryInterval = 250 * time.Millisecond

	// flipLockTTL has to outlast a flip, which writes the published rows,
	// resets the hosting cache and dispatches the publish webhooks
	// synchronously.
	flipLockTTL = 2 * time.Minute
)

// unlockScript releases a lock only when the caller still owns it.
var unlockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	end

	return 0
`)

// WarmupRequest is what a hosting node receives over the event bus.
//
// It carries the app's display name because a config can only be resolved for
// an unpublished deployment by deployment id *and* display name, and the
// environment name because the config does not hold one.
type WarmupRequest struct {
	WarmupID     string   `json:"warmupId"`
	AppID        types.ID `json:"appId"`
	EnvID        types.ID `json:"envId"`
	DisplayName  string   `json:"displayName"`
	EnvName      string   `json:"envName"`
	DeploymentID types.ID `json:"deploymentId"`
	Deadline     int64    `json:"deadline"`
}

// DeadlineAt returns the moment after which a node must stop probing.
func (r WarmupRequest) DeadlineAt() time.Time {
	return time.Unix(r.Deadline, 0)
}

// WarmupResult is one hosting node's verdict.
type WarmupResult struct {
	ServiceID    string   `json:"serviceId"`
	ServiceName  string   `json:"serviceName"`
	Status       string   `json:"status"`
	DeploymentID types.ID `json:"deploymentId,omitempty"`
	StatusCode   int      `json:"statusCode,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

// PublishStatus is what the dashboard reads while a publish is in flight, and
// afterwards to find out why one did not happen.
type PublishStatus struct {
	Status       string         `json:"status"`
	DeploymentID types.ID       `json:"deploymentId"`
	Reason       string         `json:"reason,omitempty"`
	Failures     []WarmupResult `json:"failures,omitempty"`
	UpdatedAt    int64          `json:"updatedAt"`
}

// truncateReason bounds a node's failure reason.
func truncateReason(reason string) string {
	if len(reason) <= maxWarmupReasonBytes {
		return reason
	}

	return reason[:maxWarmupReasonBytes]
}

// WarmupTimeout is how long a publish waits for the hosting nodes.
func WarmupTimeout() time.Duration {
	if seconds := utils.StringToInt(os.Getenv("STORMKIT_PUBLISH_WARMUP_TIMEOUT")); seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	return defaultWarmupTimeout
}

// warmupStore holds the transient state of a publish: the verdict each hosting
// node reported, which publish is the newest for an environment, and the
// status the dashboard reads.
//
// None of it is durable on purpose. Losing it means the environment keeps
// serving what it already served, which is the safe outcome.
type warmupStore struct{}

func (warmupStore) nodeKey(warmupID, serviceID string) string {
	return fmt.Sprintf("publish:warmup:%s:node:%s", warmupID, serviceID)
}

func (warmupStore) intentKey(envID types.ID) string {
	return fmt.Sprintf("publish:intent:%s", envID.String())
}

func (warmupStore) lockKey(envID types.ID) string {
	return fmt.Sprintf("publish:intent:%s:lock", envID.String())
}

func (warmupStore) statusKey(envID types.ID) string {
	return fmt.Sprintf("publish:status:%s", envID.String())
}

// ttlForDeadline outlives the warm-up window by an hour, so a verdict cannot
// expire while the publish waiting for it is still running.
func ttlForDeadline(deadline int64) time.Duration {
	return max(time.Until(time.Unix(deadline, 0)), 0) + time.Hour
}

// SaveNodeResultParams identifies the warm-up a verdict belongs to. A hosting
// node only ever sees the broadcast request, so it addresses the warm-up by id
// and deadline.
type SaveNodeResultParams struct {
	WarmupID string
	Deadline int64
	Result   WarmupResult
}

// SaveNodeResult records one hosting node's verdict.
//
// The size cap is applied here rather than by the caller: this is where text
// reported by a hosting node enters Redis.
func SaveNodeResult(ctx context.Context, p SaveNodeResultParams) error {
	if p.Result.ServiceID == "" {
		return errors.New("cannot record a warm-up result without a service id")
	}

	client := rediscache.Client()

	if client == nil {
		return nil
	}

	p.Result.Reason = truncateReason(p.Result.Reason)

	data, err := json.Marshal(p.Result)

	if err != nil {
		return err
	}

	store := warmupStore{}

	return client.Set(ctx, store.nodeKey(p.WarmupID, p.Result.ServiceID), data, ttlForDeadline(p.Deadline)).Err()
}

// nodeResults returns the verdicts reported so far, keyed by service id.
//
// A node that has not reported is absent from the map. Silence is never read as
// success: that is what stops a publish from going ahead on the word of nodes
// that never answered.
func (s warmupStore) nodeResults(ctx context.Context, warmupID string, nodes []string) map[string]WarmupResult {
	results := map[string]WarmupResult{}
	client := rediscache.Client()

	if client == nil {
		return results
	}

	for _, serviceID := range nodes {
		data, err := client.Get(ctx, s.nodeKey(warmupID, serviceID)).Bytes()

		// A node that cannot be read counts as pending, which fails the publish
		// safely rather than committing one that was never verified. Log the
		// real errors, or a Redis hiccup near the deadline would fail a healthy
		// publish with nothing to explain it.
		if err != nil {
			if !errors.Is(err, redis.Nil) {
				slog.Errorf("cannot read warm-up result for %s node %s: %s", warmupID, serviceID, err.Error())
			}

			continue
		}

		result := WarmupResult{}

		if err := json.Unmarshal(data, &result); err != nil {
			slog.Errorf("cannot decode warm-up result for %s node %s: %s", warmupID, serviceID, err.Error())
			continue
		}

		results[serviceID] = result
	}

	return results
}

// recordIntent marks this warm-up as the newest publish for the environment.
// The marker has to outlive the warm-up window, or a configured window longer
// than the status TTL would disable the guard it exists for.
func (s warmupStore) recordIntent(ctx context.Context, envID types.ID, warmupID string, deadline int64) {
	client := rediscache.Client()

	if client == nil {
		return
	}

	if err := client.Set(ctx, s.intentKey(envID), warmupID, ttlForDeadline(deadline)).Err(); err != nil {
		slog.Errorf("cannot record publish intent for env %s: %s", envID.String(), err.Error())
	}
}

// isNewestIntent reports whether this warm-up is still the one the environment
// should end up on.
//
// Warm-ups run detached, so a slow publish can finish after a later one already
// flipped. Without this check the older deployment would quietly win.
//
// An unreadable marker is an error rather than a "no": answering "superseded"
// would drop a publish that warmed up perfectly, with nothing recorded to
// explain why the environment never moved.
func (s warmupStore) isNewestIntent(ctx context.Context, envID types.ID, warmupID string) (bool, error) {
	client := rediscache.Client()

	if client == nil {
		return true, nil
	}

	newest, err := client.Get(ctx, s.intentKey(envID)).Result()

	if errors.Is(err, redis.Nil) {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return newest == warmupID, nil
}

// setStatus publishes what the dashboard should show for this environment.
func (s warmupStore) setStatus(ctx context.Context, envID types.ID, status PublishStatus) {
	client := rediscache.Client()

	if client == nil {
		return
	}

	status.UpdatedAt = time.Now().Unix()
	data, err := json.Marshal(status)

	if err != nil {
		slog.Errorf("cannot encode publish status for env %s: %s", envID.String(), err.Error())
		return
	}

	if err := client.Set(ctx, s.statusKey(envID), data, publishStatusTTL).Err(); err != nil {
		slog.Errorf("cannot record publish status for env %s: %s", envID.String(), err.Error())
	}
}

// PublishStatusOf returns the last known publish status of an environment, or
// nil when there is none.
func PublishStatusOf(ctx context.Context, envID types.ID) (*PublishStatus, error) {
	client := rediscache.Client()

	if client == nil {
		return nil, nil
	}

	data, err := client.Get(ctx, warmupStore{}.statusKey(envID)).Bytes()

	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	status := &PublishStatus{}

	if err := json.Unmarshal(data, status); err != nil {
		return nil, err
	}

	return status, nil
}

// lock takes the flip lock for an environment. The returned token identifies
// this holder and must be handed back to unlock.
//
// Contention and failure are told apart deliberately: a publish that merely
// lost a race is worth retrying, while one that could not reach Redis is not.
func (s warmupStore) lock(ctx context.Context, envID types.ID) (string, bool, error) {
	token := uuid.New().String()
	client := rediscache.Client()

	if client == nil {
		return token, true, nil
	}

	ok, err := client.SetNX(ctx, s.lockKey(envID), token, flipLockTTL).Result()

	if err != nil {
		return "", false, err
	}

	return token, ok, nil
}

// unlock releases the flip lock, but only if this caller still holds it.
//
// The flip writes to the database, resets the hosting cache and dispatches
// publish webhooks, so it can outrun the lock's expiry. Without the token check
// a slow flip would delete a lock another one had since taken.
func (s warmupStore) unlock(ctx context.Context, envID types.ID, token string) {
	client := rediscache.Client()

	if client == nil {
		return
	}

	if err := unlockScript.Run(ctx, client, []string{s.lockKey(envID)}, token).Err(); err != nil && !errors.Is(err, redis.Nil) {
		slog.Errorf("cannot release publish lock for env %s: %s", envID.String(), err.Error())
	}
}
