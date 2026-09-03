package runner

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
)

// MinFreeDiskEnvVar overrides how much free disk a deployment requires before
// it starts, in megabytes. Set it to 0 to disable the check entirely.
const MinFreeDiskEnvVar = "STORMKIT_MIN_FREE_DISK_MB"

// DefaultMinFreeDiskMB is the headroom a build needs before it is worth
// starting. A checkout plus a node_modules tree routinely exceeds a gigabyte,
// and running out mid-build wastes the whole deployment.
const DefaultMinFreeDiskMB = 2048

// diskPreflight refuses to start a build that has no room to finish.
//
// Without it a full disk surfaces as `No space left on device` from whichever
// command happened to run first — a checkout, an npm install, a nix build —
// which tells the operator nothing about what to clean up.
type diskPreflight struct {
	dir      string
	reporter *ReporterModel
}

// minFreeBytes returns the required headroom, or 0 when the check is disabled.
func (p diskPreflight) minFreeBytes() uint64 {
	mb := int64(DefaultMinFreeDiskMB)

	if v := os.Getenv(MinFreeDiskEnvVar); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)

		if err != nil || parsed < 0 {
			slog.Errorf("ignoring invalid %s=%q", MinFreeDiskEnvVar, v)
		} else {
			mb = parsed
		}
	}

	if mb <= 0 {
		return 0
	}

	return uint64(mb) << 20
}

// check verifies there is room to build, reclaiming Nix store space first if
// there is not. A build host that cannot report its own disk usage is allowed
// to proceed: the check exists to give a better error, not to add a new way to
// fail.
func (p diskPreflight) check(ctx context.Context) error {
	required := p.minFreeBytes()

	if required == 0 {
		return nil
	}

	usage, err := nixstore.DiskUsage(p.dir)

	if err != nil {
		slog.Errorf("could not read disk usage of %s: %v", p.dir, err)
		return nil
	}

	if usage.FreeBytes >= required {
		return nil
	}

	// Last chance before failing: the Nix store is usually what filled the
	// disk, and it is always safe to drop paths outside the retention window.
	if nixstore.Available() {
		p.reporter.AddStep("reclaim disk space")
		p.reporter.AddLine(fmt.Sprintf(
			"Low disk space (%s free). Reclaiming space from the Nix store...",
			humanBytes(usage.FreeBytes),
		))

		if err := nixstore.CollectGarbage(ctx, nixstore.CollectGarbageParams{}); err != nil {
			slog.Errorf("error collecting nix garbage during preflight: %v", err)
		}

		if usage, err = nixstore.DiskUsage(p.dir); err != nil {
			slog.Errorf("could not read disk usage of %s: %v", p.dir, err)
			return nil
		}

		if usage.FreeBytes >= required {
			p.reporter.AddLine(fmt.Sprintf("Reclaimed enough space (%s free).", humanBytes(usage.FreeBytes)))
			return nil
		}
	}

	return fmt.Errorf(
		"Not enough disk space on the build host: %s free of %s, and this deployment needs at least %s. "+
			"Free up space on the server, then retry the deployment.",
		humanBytes(usage.FreeBytes), humanBytes(usage.TotalBytes), humanBytes(required),
	)
}

// humanBytes formats a byte count for a customer-facing build log.
func humanBytes(b uint64) string {
	const gb = 1 << 30

	if b >= gb {
		return fmt.Sprintf("%.1fGB", float64(b)/gb)
	}

	return fmt.Sprintf("%dMB", b/(1<<20))
}
