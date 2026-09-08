package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
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
	// Tests publish without the gate. Two reasons: work that outlives the
	// request would write to a transaction that has since been rolled back,
	// corrupting whichever test runs next rather than the one that started it;
	// and service discovery is shared, so a registration left behind by another
	// package's test binary would have a publish wait for a node that is never
	// going to answer.
	//
	// The gate's own behaviour is covered by PublishGateSuite, which drives it
	// with service discovery faked.
	if config.IsTest() {
		if err := onReady(); err != nil {
			slog.Errorf("cannot publish deployment %s: %s", p.DeploymentID.String(), err.Error())
		}

		return
	}

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

// PublishSettingsFor returns the arguments that point an environment at a
// deployment. An environment serves exactly one, so there is only ever one.
func PublishSettingsFor(envID, deploymentID types.ID) []*PublishSettings {
	return []*PublishSettings{
		{
			EnvID:        envID,
			DeploymentID: deploymentID,
		},
	}
}

// PublishWithWarmupParams describes a publish that has to wait for the
// deployment to answer.
type PublishWithWarmupParams struct {
	EnvID        types.ID
	DeploymentID types.ID

	// OnPublished runs after the environment has moved, for work that must not
	// happen when a publish is held back — an audit entry, say.
	//
	// It cannot fail the publish. By the time it runs the environment is
	// already serving the new deployment, so reporting a failure here would
	// tell the user their release did not happen when it did.
	OnPublished func()
}

// PublishWithWarmup warms the deployment on every hosting node and moves the
// environment onto it once they all report that it serves.
//
// It returns as soon as the warm-up has been requested. An error means the
// publish could not be started at all; a deployment that fails to come up is
// reported through PublishStatusOf instead, because by then the caller is long
// gone.
func PublishWithWarmup(ctx context.Context, p PublishWithWarmupParams) error {
	env, err := buildconf.NewStore().EnvironmentByID(ctx, p.EnvID)

	if err != nil {
		return err
	}

	if env == nil {
		return fmt.Errorf("environment %s does not exist", p.EnvID.String())
	}

	appl, err := app.NewStore().AppByEnvID(ctx, p.EnvID)

	if err != nil {
		return err
	}

	if appl == nil {
		return fmt.Errorf("app of environment %s does not exist", p.EnvID.String())
	}

	// The request's context is done the moment it responds, and the warm-up
	// outlives that.
	bg := context.WithoutCancel(ctx)
	settings := PublishSettingsFor(p.EnvID, p.DeploymentID)

	WarmUpAndPublish(bg, WarmUpAndPublishParams{
		AppID:        env.AppID,
		EnvID:        p.EnvID,
		DeploymentID: p.DeploymentID,
		DisplayName:  appl.DisplayName,
		EnvName:      env.Name,
	}, func() error {
		if err := publishNow(bg, settings); err != nil {
			return err
		}

		if p.OnPublished != nil {
			p.OnPublished()
		}

		return nil
	})

	return nil
}

// AttachPublishStatus marks the deployments that are currently warming up.
//
// Publishing waits for the deployment to answer before traffic moves to it, so
// "not published" alone no longer says whether nothing is happening or a
// release is on its way.
//
// It reads once per environment rather than once per deployment, since a list
// commonly holds many deployments of the same one.
func AttachPublishStatus(ctx context.Context, deployments []*Deployment) {
	statuses := map[types.ID]*PublishStatus{}

	for _, d := range deployments {
		if _, read := statuses[d.EnvID]; read {
			continue
		}

		status, err := PublishStatusOf(ctx, d.EnvID)

		if err != nil {
			slog.Errorf("cannot read the publish status of env %s: %s", d.EnvID.String(), err.Error())
		}

		statuses[d.EnvID] = status
	}

	for _, d := range deployments {
		d.IsWarmingUp = isWarmingUp(d, statuses[d.EnvID])
	}
}

// isWarmingUp reports whether a publish of this deployment is under way.
//
// The environment's status only speaks for the deployment it names: an older
// deployment must not be shown as warming up because the one replacing it is.
func isWarmingUp(d *Deployment, status *PublishStatus) bool {
	return status != nil &&
		status.DeploymentID == d.ID &&
		status.Status == PublishStatusPublishing
}
