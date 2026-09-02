// Package nixstore reports disk usage for a filesystem path and garbage
// collects the Nix store.
//
// Self-hosted instances persist /nix in a Docker volume so packages survive
// restarts. Every deployment whose repository ships a flake.nix adds new store
// paths, and nothing ever removes them, so the volume grows without bound until
// the host disk is full.
package nixstore

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// LookPath resolves a binary on PATH. It is a variable so tests, in this
// package and in callers, can simulate a container with or without Nix.
var LookPath = exec.LookPath

// DefaultPath is where the Nix store lives inside Stormkit containers.
// It is a variable so tests can point it at a temporary directory.
var DefaultPath = "/nix"

// DefaultRetentionDays is the age below which store generations are kept, so a
// rollback to a recent deployment does not need to re-download its packages.
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

// CollectGarbage deletes Nix store paths that no longer belong to a generation
// newer than the retention window. It is a no-op when the container has no Nix
// store.
func CollectGarbage(ctx context.Context, p CollectGarbageParams) error {
	if !Available() {
		return nil
	}

	if p.RetentionDays < 1 {
		p.RetentionDays = DefaultRetentionDays
	}

	return sys.Command(ctx, sys.CommandOpts{
		Name: "nix-collect-garbage",
		Args: []string{"--delete-older-than", strconv.Itoa(p.RetentionDays) + "d"},
	}).Run()
}
