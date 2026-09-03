// Package nixstore reports disk usage for a filesystem path and garbage
// collects the Nix store.
//
// Self-hosted instances persist /nix in a Docker volume so packages survive
// restarts. Every deployment whose repository ships a flake.nix adds new store
// paths, and nothing ever removes them, so the volume grows without bound until
// the host disk is full.
//
// Collecting alone is not enough. Nix keeps only what a garbage collection root
// points at, and a deployment leaves none behind: once its build finishes, or
// its service goes idle and is killed, its packages are unreferenced and a
// collection deletes them all. This package therefore registers one profile per
// environment (see profile.go) and settles those roots before collecting, so a
// run reclaims superseded deployments while leaving each active environment its
// dev shell.
package nixstore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// LookPath resolves a binary on PATH. It is a variable so tests, in this
// package and in callers, can simulate a container with or without Nix.
var LookPath = exec.LookPath

// DefaultPath is where the Nix store lives inside Stormkit containers.
// It is a variable so tests can point it at a temporary directory.
var DefaultPath = "/nix"

// DefaultRetentionDays is how long an environment keeps its packages after its
// last deployment, so redeploying or waking an idle service does not have to
// download the dev shell again.
const DefaultRetentionDays = 7

// Usage describes the filesystem backing a path.
type Usage struct {
	Path        string  `json:"path"`
	TotalBytes  uint64  `json:"totalBytes"`
	FreeBytes   uint64  `json:"freeBytes"`
	UsedBytes   uint64  `json:"usedBytes"`
	UsedPercent float64 `json:"usedPercent"`
}

// Available reports whether this container has a Nix store to collect.
// The hosting and workerserver images ship Nix; other callers (tests, the API
// container, external runners) do not, and must not fail because of it.
func Available() bool {
	if _, err := os.Stat(DefaultPath); err != nil {
		return false
	}

	_, err := LookPath("nix-collect-garbage")

	return err == nil
}

// DiskUsage reports the usage of the filesystem containing path. It describes
// the whole filesystem, not the path's own size: /nix and / usually share one
// device, which is exactly the disk that runs out.
func DiskUsage(path string) (Usage, error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(path, &stat); err != nil {
		return Usage{}, err
	}

	blockSize := uint64(stat.Bsize)
	total := stat.Blocks * blockSize
	// Bavail, not Bfree: blocks reserved for root are not free to us.
	free := stat.Bavail * blockSize

	usage := Usage{
		Path:       path,
		TotalBytes: total,
		FreeBytes:  free,
	}

	if total > 0 {
		usage.UsedBytes = total - free
		usage.UsedPercent = float64(usage.UsedBytes) / float64(total) * 100
	}

	return usage, nil
}

// CollectGarbageParams configures a garbage collection run.
type CollectGarbageParams struct {
	// RetentionDays keeps store generations younger than this many days.
	// Values below 1 fall back to DefaultRetentionDays.
	RetentionDays int
}

// CollectGarbage deletes Nix store paths that no longer belong to a live
// environment. It is a no-op when the container has no Nix store.
//
// Roots are settled before collecting, because nix-collect-garbage only keeps
// what something points at: stale generations are pruned so a redeployed
// environment stops pinning its previous toolchain, then the profiles of
// environments that have not deployed within the retention window are dropped.
// What remains is one dev shell per active environment.
func CollectGarbage(ctx context.Context, p CollectGarbageParams) error {
	if !Available() {
		return nil
	}

	if p.RetentionDays < 1 {
		p.RetentionDays = DefaultRetentionDays
	}

	if err := PruneAllGenerations(ctx); err != nil {
		slog.Errorf("error pruning nix profile generations: %v", err)
	}

	if _, err := ReapProfiles(ReapProfilesParams{RetentionDays: p.RetentionDays}); err != nil {
		slog.Errorf("error reaping nix profiles: %v", err)
	}

	out, err := sys.Command(ctx, sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", strconv.Itoa(p.RetentionDays) + "d"},
	}).CombinedOutput()

	if err != nil {
		return fmt.Errorf("nix-collect-garbage: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}
