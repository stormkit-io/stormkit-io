package admin

import (
	"context"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"go.uber.org/zap"
)

// nixGarbageCollectInterval is how often each container collects its own Nix
// store. It is a variable so tests do not have to wait a day.
var nixGarbageCollectInterval = 24 * time.Hour

// nixGarbageCollectDelay staggers the first run so a container that is
// restarting in a loop does not spend its whole life garbage collecting.
var nixGarbageCollectDelay = 5 * time.Minute

// CollectNixGarbage garbage collects this container's Nix store. It matches the
// rediscache.Handler signature so it can also be triggered on demand, and never
// returns an error: a failed collection must not take the container down.
func CollectNixGarbage(ctx context.Context, _ ...string) {
	if !nixstore.Available() {
		return
	}

	before, _ := nixstore.DiskUsage(nixstore.DefaultPath)

	if err := nixstore.CollectGarbage(ctx, nixstore.CollectGarbageParams{}); err != nil {
		slog.Errorf("error collecting nix garbage: %v", err)
		return
	}

	after, _ := nixstore.DiskUsage(nixstore.DefaultPath)

	slog.Debug(slog.LogOpts{
		Msg:   "collected nix garbage",
		Level: slog.DL1,
		Payload: []zap.Field{
			zap.Int("retentionDays", nixstore.DefaultRetentionDays),
			zap.Uint64("freeBytesBefore", before.FreeBytes),
			zap.Uint64("freeBytesAfter", after.FreeBytes),
		},
	})
}

// StartDiskMaintenance periodically garbage collects the Nix store of the
// container it runs in. Both hosting and workerserver mount their own /nix
// volume, and neither can clean up the other's, so each starts its own loop
// rather than this being a leader-elected workerserver job.
func StartDiskMaintenance(ctx context.Context) {
	if !nixstore.Available() {
		return
	}

	timer := time.NewTimer(nixGarbageCollectDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	}

	ticker := time.NewTicker(nixGarbageCollectInterval)
	defer ticker.Stop()

	for {
		CollectNixGarbage(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
