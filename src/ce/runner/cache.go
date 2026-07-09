package runner

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// maxCacheArchiveSize caps the snapshot size so a misconfigured cacheDirs
// cannot fill up the storage with multi-gigabyte archives.
const maxCacheArchiveSize = 2 << 30 // 2GiB

// DefaultCacheStore overrides the storage client used for build caches.
// It is meant for tests.
var DefaultCacheStore integrations.CacheStore

// cacheManager restores and snapshots the build cache archive around a
// build. All of its operations are best-effort: a cache failure is reported
// to the build log but never fails the deployment.
type cacheManager struct {
	opts        RunnerOpts
	archivePath string
}

func newCacheManager(opts RunnerOpts) *cacheManager {
	return &cacheManager{
		opts:        opts,
		archivePath: path.Join(opts.RootDir, "build-cache.tar.gz"),
	}
}

// dirs returns the sanitized cache directories. The API validates these on
// save; this re-validation keeps a forged deployment message from extracting
// or archiving paths outside the working directory.
func (c *cacheManager) dirs() []string {
	valid := []string{}

	for _, dir := range c.opts.Build.CacheDirs {
		if len(buildconf.ValidateCacheDirs([]string{dir})) == 0 {
			valid = append(valid, path.Clean(dir))
		}
	}

	return valid
}

// enabled reports whether caching should run. The deployer clears CacheDirs
// when caching is not allowed, so a non-empty (valid) list is the signal.
func (c *cacheManager) enabled() bool {
	return len(c.dirs()) > 0
}

func (c *cacheManager) store() integrations.CacheStore {
	if DefaultCacheStore != nil {
		return DefaultCacheStore
	}

	if c.opts.Uploader == nil {
		return nil
	}

	client := integrations.Client(integrations.ClientArgs{
		Provider:  c.opts.Uploader.Provider,
		AccessKey: c.opts.Uploader.AccessKey,
		SecretKey: c.opts.Uploader.SecretKey,
		Region:    c.opts.Uploader.Region,
	})

	if store, ok := client.(integrations.CacheStore); ok {
		return store
	}

	return nil
}

func (c *cacheManager) artifactArgs() integrations.CacheArtifactArgs {
	args := integrations.CacheArtifactArgs{
		AppID:     utils.StringToID(c.opts.Build.AppID),
		EnvID:     utils.StringToID(c.opts.Build.EnvID),
		LocalPath: c.archivePath,
	}

	if c.opts.Uploader != nil {
		args.BucketName = c.opts.Uploader.BucketName
	}

	return args
}

// Restore downloads the environment's cache archive and extracts it into
// the working directory. It runs before the install step.
func (c *cacheManager) Restore(ctx context.Context) {
	if !c.enabled() {
		return
	}

	store := c.store()

	if store == nil {
		return
	}

	c.opts.Reporter.AddStep("restoring build cache")

	found, err := store.DownloadCacheArtifact(ctx, c.artifactArgs())

	if err != nil {
		c.reportError("could not download build cache", err)
		return
	}

	if !found {
		c.opts.Reporter.AddLine("no build cache found - starting from a clean state")
		return
	}

	defer os.Remove(c.archivePath)

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "tar",
		Args: []string{"-xzf", c.archivePath, "-C", c.opts.WorkDir},
		Dir:  c.opts.RootDir,
	})

	if out, err := cmd.CombinedOutput(); err != nil {
		c.reportError(fmt.Sprintf("could not extract build cache: %s", string(out)), err)
		return
	}

	c.opts.Reporter.AddLine(fmt.Sprintf("restored cached directories: %v", c.dirs()))
}

// Snapshot archives the cache directories and uploads them, replacing the
// environment's previous cache. It runs after a successful build.
func (c *cacheManager) Snapshot(ctx context.Context) {
	if !c.enabled() {
		return
	}

	store := c.store()

	if store == nil {
		return
	}

	dirs := []string{}

	for _, dir := range c.dirs() {
		if stat, err := os.Stat(path.Join(c.opts.WorkDir, dir)); err == nil && stat.IsDir() {
			dirs = append(dirs, dir)
		}
	}

	if len(dirs) == 0 {
		c.opts.Reporter.AddLine("no cache directories found after the build - skipping")
		return
	}

	c.opts.Reporter.AddStep("saving build cache")

	defer os.Remove(c.archivePath)

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "tar",
		Args: append([]string{"-czf", c.archivePath, "-C", c.opts.WorkDir}, dirs...),
		Dir:  c.opts.RootDir,
	})

	if out, err := cmd.CombinedOutput(); err != nil {
		c.reportError(fmt.Sprintf("could not archive build cache: %s", string(out)), err)
		return
	}

	stat, err := os.Stat(c.archivePath)

	if err != nil {
		c.reportError("could not stat build cache archive", err)
		return
	}

	if stat.Size() > maxCacheArchiveSize {
		c.opts.Reporter.AddLine(
			fmt.Sprintf(
				"build cache is too large (%dMB > %dMB) - reduce the cache directories to enable caching",
				stat.Size()>>20,
				int64(maxCacheArchiveSize)>>20,
			),
		)
		return
	}

	if err := store.UploadCacheArtifact(ctx, c.artifactArgs()); err != nil {
		c.reportError("could not upload build cache", err)
		return
	}

	c.opts.Reporter.AddLine(fmt.Sprintf("saved build cache (%dMB): %v", stat.Size()>>20, dirs))
}

func (c *cacheManager) reportError(msg string, err error) {
	slog.Errorf("%s: %v", msg, err)
	c.opts.Reporter.AddLine(fmt.Sprintf("%s - continuing without cache", msg))
}
