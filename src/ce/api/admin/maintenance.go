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

// NixRetentionDays returns the configured retention window, falling back to the
// default when the instance has no explicit setting.
func NixRetentionDays(ctx context.Context) int {
	vc, err := Store().Config(ctx)

	if err != nil || vc.SystemConfig == nil || vc.SystemConfig.NixRetentionDays <= 0 {
		return nixstore.DefaultRetentionDays
	}

	return vc.SystemConfig.NixRetentionDays
}

// CollectNixGarbage garbage collects this container's Nix store. It matches the
// rediscache.Handler signature so it can also be triggered on demand, and never
// returns an error: a failed collection must not take the container down.
func CollectNixGarbage(ctx context.Context, _ ...string) {
	if !nixstore.Available() {
		return
	}

	retentionDays := NixRetentionDays(ctx)
	before, _ := nixstore.DiskUsage(nixstore.DefaultPath)

	if err := nixstore.CollectGarbage(ctx, nixstore.CollectGarbageParams{RetentionDays: retentionDays}); err != nil {
		slog.Errorf("error collecting nix garbage: %v", err)
		return
	}

	after, _ := nixstore.DiskUsage(nixstore.DefaultPath)

	slog.Debug(slog.LogOpts{
		Msg:   "collected nix garbage",
		Level: slog.DL1,
		Payload: []zap.Field{
			zap.Int("retentionDays", retentionDays),
			zap.Uint64("freeBytesBefore", before.FreeBytes),
			zap.Uint64("freeBytesAfter", after.FreeBytes),
		},
	})
}

// StartDiskMaintenance starts this container's disk background work and
// returns immediately. Both hosting and workerserver mount their own /nix
// volume, and neither can clean up or measure the other's, so each runs its
// own loops rather than this being a leader-elected workerserver job.
func StartDiskMaintenance(ctx context.Context) {
	go reportDiskUsageLoop(ctx)
	go collectNixGarbageLoop(ctx)
}

func collectNixGarbageLoop(ctx context.Context) {
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

func reportDiskUsageLoop(ctx context.Context) {
	ticker := time.NewTicker(diskReportInterval)
	defer ticker.Stop()

	for {
		ReportDiskUsage(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
