package deploy

import (
	"context"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// This file exposes the warm-up internals to the package's external tests. It
// carries the _test suffix, so none of it is built into the binary.

// NodeResultsForTest reads back the verdicts hosting nodes reported.
func NodeResultsForTest(ctx context.Context, warmupID string, nodes []string) map[string]WarmupResult {
	return warmupStore{}.nodeResults(ctx, warmupID, nodes)
}

// SetPublishStatusForTest writes what the dashboard would read.
func SetPublishStatusForTest(ctx context.Context, envID types.ID, status PublishStatus) {
	warmupStore{}.setStatus(ctx, envID, status)
}

// RecordIntentForTest marks a warm-up as the newest publish of an environment.
func RecordIntentForTest(ctx context.Context, envID types.ID, warmupID string) {
	warmupStore{}.recordIntent(ctx, envID, warmupID, time.Now().Add(time.Minute).Unix())
}

// IsNewestIntentForTest reports whether a warm-up is still the newest publish.
func IsNewestIntentForTest(ctx context.Context, envID types.ID, warmupID string) (bool, error) {
	return warmupStore{}.isNewestIntent(ctx, envID, warmupID)
}

// AcquireFlipLockForTest takes the lock a publish needs before it can apply.
func AcquireFlipLockForTest(ctx context.Context, envID types.ID) (string, error) {
	return warmUpGate{}.acquire(ctx, envID)
}

// ReleaseFlipLockForTest hands the lock back.
func ReleaseFlipLockForTest(ctx context.Context, envID types.ID, token string) {
	warmupStore{}.unlock(ctx, envID, token)
}

// PublishNowForTest performs the flip without the warm-up gate, for tests that
// need an environment already pointing at a deployment.
func PublishNowForTest(ctx context.Context, settings []*PublishSettings) error {
	return publishNow(ctx, settings)
}

// WarmUpAndPublishAsyncForTest drives the gate the way production does — on its
// own goroutine — regardless of the inline-under-test shortcut.
func WarmUpAndPublishAsyncForTest(ctx context.Context, p WarmUpAndPublishParams, onReady func() error) {
	go warmUpGate{}.run(ctx, p, onReady)
}
