package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"go.uber.org/zap"
)

// WarmUpAndPublishParams describes the deployment a publish is waiting on.
type WarmUpAndPublishParams struct {
	AppID        types.ID
	EnvID        types.ID
	DeploymentID types.ID
	DisplayName  string
	EnvName      string
	Timeout      time.Duration
}

// WarmUpAndPublish asks every hosting node to boot the deployment and runs
// onReady once they all report that it serves.
//
// It returns immediately. The caller is not blocked, and onReady — which is
// where the environment actually moves to the new deployment — runs only after
// the deployment has answered. If any node cannot get it answering, onReady is
// never called and the environment keeps serving what it served before.
//
// The progress and the reason for a failure are readable through
// PublishStatusOf.
func WarmUpAndPublish(ctx context.Context, p WarmUpAndPublishParams, onReady func() error) {
	// The caller is usually an HTTP handler whose context is cancelled the
	// moment it responds, and this outlives the response by design.
	go warmUpGate{}.run(context.WithoutCancel(ctx), p, onReady)
}

// warmUpGate holds a publish back until the deployment answers.
type warmUpGate struct{}

func (g warmUpGate) run(ctx context.Context, p WarmUpAndPublishParams, onReady func() error) {
	store := warmupStore{}
	warmupID := uuid.New().String()

	if p.Timeout <= 0 {
		p.Timeout = WarmupTimeout()
	}

	deadline := time.Now().Add(p.Timeout)

	store.recordIntent(ctx, p.EnvID, warmupID, deadline.Unix())
	store.setStatus(ctx, p.EnvID, PublishStatus{
		Status:       PublishStatusPublishing,
		DeploymentID: p.DeploymentID,
	})

	nodes, err := g.hostingNodes()

	// The gate is an improvement on publishing, not a precondition for it.
	// Refusing to publish because the node list could not be read would stop
	// releases altogether whenever Redis is unavailable — including on installs
	// that run without it.
	if err != nil {
		slog.Errorf(
			"publishing deployment %s without a warm-up: cannot list the hosting nodes: %s",
			p.DeploymentID.String(), err.Error(),
		)

		g.commit(ctx, p, warmupID, nil, onReady)
		return
	}

	// Nothing is serving this environment yet, so there is nothing to warm and
	// nothing to protect. Publishing straight away keeps a fresh install, and
	// every test, behaving as it did before.
	if len(nodes) == 0 {
		slog.Infof(
			"publishing deployment %s without a warm-up: no hosting node is registered",
			p.DeploymentID.String(),
		)

		g.commit(ctx, p, warmupID, nil, onReady)
		return
	}

	if err := g.broadcast(p, warmupID, deadline); err != nil {
		g.fail(ctx, p, warmupID, fmt.Sprintf("cannot ask the hosting nodes to warm up: %s", err.Error()), nil)
		return
	}

	failures, err := g.wait(ctx, waitParams{
		WarmupID: warmupID,
		Nodes:    nodes,
		// The nodes stop probing at the deadline and only then write their
		// verdict. Without this grace the waiter would give up a moment before
		// the reason it wants to report arrives.
		Deadline: deadline.Add(warmupReportGrace),
		Timeout:  p.Timeout,
	})

	if err != nil {
		g.fail(ctx, p, warmupID, err.Error(), failures)
		return
	}

	g.commit(ctx, p, warmupID, failures, onReady)
}

// hostingNodes snapshots the nodes expected to report.
//
// Taking the set once is what makes the wait decidable: a node that registers
// afterwards is not waited on, and cold-starts on its first real request the
// way every node does today.
func (g warmUpGate) hostingNodes() ([]string, error) {
	services, err := rediscache.Service().List([]string{rediscache.ServiceHosting})

	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(services))

	for _, service := range services {
		ids = append(ids, service.ID)
	}

	return ids, nil
}

func (g warmUpGate) broadcast(p WarmUpAndPublishParams, warmupID string, deadline time.Time) error {
	payload, err := json.Marshal(WarmupRequest{
		WarmupID:     warmupID,
		AppID:        p.AppID,
		EnvID:        p.EnvID,
		DisplayName:  p.DisplayName,
		EnvName:      p.EnvName,
		DeploymentID: p.DeploymentID,
		Deadline:     deadline.Unix(),
	})

	if err != nil {
		return err
	}

	return rediscache.Broadcast(rediscache.EventPublishWarmup, string(payload))
}

type waitParams struct {
	WarmupID string
	Nodes    []string
	Deadline time.Time
	Timeout  time.Duration
}

// wait blocks until every snapshotted node reports ready, one reports a
// failure, or the deadline passes.
func (g warmUpGate) wait(ctx context.Context, p waitParams) ([]WarmupResult, error) {
	store := warmupStore{}
	roster := &nodeRoster{nodes: p.Nodes}

	for {
		results := store.nodeResults(ctx, p.WarmupID, p.Nodes)
		failures := []WarmupResult{}

		for _, result := range results {
			if result.Status == rediscache.StatusErr {
				failures = append(failures, result)
			}
		}

		if len(failures) > 0 {
			return failures, fmt.Errorf("the deployment did not come up: %s", failures[0].Reason)
		}

		pending := roster.pending(results)

		if len(pending) == 0 {
			// Every node either answered or is gone. If they all went away
			// without answering, nothing verified this deployment and it must
			// not be published.
			if len(results) == 0 {
				return nil, errors.New("every hosting node disappeared while warming up the deployment")
			}

			return nil, nil
		}

		if !time.Now().Before(p.Deadline) {
			return nil, fmt.Errorf(
				"%d of %d hosting nodes did not finish warming up the deployment within %s",
				len(pending), len(p.Nodes), p.Timeout.Round(time.Second),
			)
		}

		select {
		case <-time.After(warmupPollInterval):
		case <-ctx.Done():
			return nil, fmt.Errorf("the publish was cancelled while waiting for the deployment")
		}
	}
}

// nodeRoster tracks which of the snapshotted nodes are still owed a verdict.
type nodeRoster struct {
	nodes []string

	// missingSince is when a node was first seen absent from service discovery.
	missingSince map[string]time.Time

	live        []string
	refreshedAt time.Time
}

// pending returns the nodes still expected to report.
//
// A node that has de-registered is eventually dropped: a hosting container that
// restarts mid-publish would otherwise hold the release until the deadline and
// then fail it, even though nothing serves from it any more. It is only dropped
// after it has been absent for longer than a registration lifetime, because a
// single delayed heartbeat makes a perfectly healthy node disappear for a few
// seconds — and dropping one of those would let a publish commit while a node
// that never warmed the deployment carries on serving.
func (r *nodeRoster) pending(results map[string]WarmupResult) []string {
	alive := r.aliveNow()
	waiting := []string{}

	for _, id := range r.nodes {
		if _, reported := results[id]; reported {
			delete(r.missingSince, id)
			continue
		}

		if alive[id] {
			delete(r.missingSince, id)
			waiting = append(waiting, id)

			continue
		}

		if r.missingSince == nil {
			r.missingSince = map[string]time.Time{}
		}

		if _, seen := r.missingSince[id]; !seen {
			r.missingSince[id] = time.Now()
		}

		if time.Since(r.missingSince[id]) < rediscache.ServiceRegistrationTTL {
			waiting = append(waiting, id)
		}
	}

	return waiting
}

// aliveNow returns the registered hosting nodes, re-listing at most once every
// few seconds: the wait polls far more often than service discovery changes,
// and each listing is a scan plus a read per service.
func (r *nodeRoster) aliveNow() map[string]bool {
	if time.Since(r.refreshedAt) > liveNodeCacheTTL {
		gate := warmUpGate{}
		live, err := gate.hostingNodes()

		if err == nil {
			r.live = live
			r.refreshedAt = time.Now()
		} else {
			// Without a reliable list, assume every snapshotted node is still
			// there. Waiting too long is recoverable; publishing unverified is
			// not.
			r.live = r.nodes
		}
	}

	alive := make(map[string]bool, len(r.live))

	for _, id := range r.live {
		alive[id] = true
	}

	return alive
}

// commit runs onReady if this publish is still the one the environment should
// end up on.
func (g warmUpGate) commit(ctx context.Context, p WarmUpAndPublishParams, warmupID string, failures []WarmupResult, onReady func() error) {
	store := warmupStore{}
	token, err := g.acquire(ctx, p.EnvID)

	if err != nil {
		g.fail(ctx, p, warmupID, fmt.Sprintf("the deployment came up but could not be published: %s", err.Error()), failures)
		return
	}

	defer store.unlock(ctx, p.EnvID, token)

	// A publish that started later may already have flipped while this one was
	// still warming up. Applying this one now would quietly roll the
	// environment back to the older deployment.
	newest, err := store.isNewestIntent(ctx, p.EnvID, warmupID)

	if err != nil {
		g.fail(ctx, p, warmupID, fmt.Sprintf("cannot tell whether this is still the newest publish: %s", err.Error()), failures)
		return
	}

	if !newest {
		slog.Debug(slog.LogOpts{
			Msg:   "skipping a publish that a newer one has superseded",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("env_id", p.EnvID.String()),
				zap.String("deployment_id", p.DeploymentID.String()),
			},
		})

		return
	}

	if err := onReady(); err != nil {
		g.fail(ctx, p, warmupID, fmt.Sprintf("the deployment came up but could not be published: %s", err.Error()), failures)
		return
	}

	store.setStatus(ctx, p.EnvID, PublishStatus{
		Status:       PublishStatusPublished,
		DeploymentID: p.DeploymentID,
	})
}

// acquire takes the flip lock, waiting briefly if another publish holds it.
//
// Losing a race is not a reason to throw away a deployment that already passed
// its warm-up, so contention is retried; only an unusable lock is an error.
func (g warmUpGate) acquire(ctx context.Context, envID types.ID) (string, error) {
	store := warmupStore{}
	deadline := time.Now().Add(flipLockWait)

	for {
		token, locked, err := store.lock(ctx, envID)

		if err != nil {
			return "", err
		}

		if locked {
			return token, nil
		}

		if !time.Now().Before(deadline) {
			return "", errors.New("another publish for this environment is still being applied")
		}

		select {
		case <-time.After(flipLockRetryInterval):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// fail records why the environment is still serving the previous deployment.
//
// A superseded publish stays quiet: overwriting the status now would report a
// failure for an environment that a newer publish has already put right.
func (g warmUpGate) fail(ctx context.Context, p WarmUpAndPublishParams, warmupID, reason string, failures []WarmupResult) {
	slog.Errorf("publish of deployment %s did not happen: %s", p.DeploymentID.String(), reason)

	store := warmupStore{}
	newest, err := store.isNewestIntent(ctx, p.EnvID, warmupID)

	if err == nil && !newest {
		return
	}

	store.setStatus(ctx, p.EnvID, PublishStatus{
		Status:       PublishStatusFailed,
		DeploymentID: p.DeploymentID,
		Reason:       reason,
		Failures:     failures,
	})
}

// PublishSettingsFor returns the arguments that publish a deployment in full.
// Percentage-based releases are retired, so there is only ever one of them.
func PublishSettingsFor(envID, deploymentID types.ID) []*PublishSettings {
	return []*PublishSettings{
		{
			EnvID:        envID,
			DeploymentID: deploymentID,
			Percentage:   publishedPercentage,
		},
	}
}
