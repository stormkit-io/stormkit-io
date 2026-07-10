package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// maxCacheArchiveSize caps the size of a single directory's archive so a
// misconfigured cacheDirs cannot fill up the storage with multi-gigabyte
// archives.
const maxCacheArchiveSize = 2 << 30 // 2GiB

// DefaultCacheStore overrides the storage client used for build caches.
// It is meant for tests.
var DefaultCacheStore integrations.CacheStore

// cacheManager restores and snapshots the build cache archives around a
// build. Each cache directory is stored as its own archive so an unchanged
// directory (e.g. node_modules) skips the upload even when a volatile one
// (e.g. .next/cache) changed. All of its operations are best-effort: a cache
// failure is reported to the build log but never fails the deployment.
type cacheManager struct {
	opts        RunnerOpts
	cacheStore  integrations.CacheStore
	initialized bool

	// restoredHash maps each restored cache directory to the content hash
	// taken right after extraction. Snapshot compares against it to skip the
	// upload when the build did not change the cached contents.
	restoredHash map[string]string
}

func newCacheManager(opts RunnerOpts) *cacheManager {
	return &cacheManager{
		opts:         opts,
		restoredHash: map[string]string{},
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
	if c.initialized {
		return c.cacheStore
	}

	c.initialized = true

	if DefaultCacheStore != nil {
		c.cacheStore = DefaultCacheStore
		return c.cacheStore
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
		c.cacheStore = store
	}

	return c.cacheStore
}

// archivePath returns the local path of the archive holding the given cache
// directory.
func (c *cacheManager) archivePath(dir string) string {
	return path.Join(c.opts.RootDir, fmt.Sprintf("build-cache-%s.tar.gz", integrations.CacheDirToken(dir)))
}

func (c *cacheManager) artifactArgs(dir string) integrations.CacheArtifactArgs {
	args := integrations.CacheArtifactArgs{
		AppID:     utils.StringToID(c.opts.Build.AppID),
		EnvID:     utils.StringToID(c.opts.Build.EnvID),
		Dir:       dir,
		LocalPath: c.archivePath(dir),
	}

	if c.opts.Uploader != nil {
		args.BucketName = c.opts.Uploader.BucketName
	}

	return args
}

// Restore downloads the environment's cache archives and extracts them into
// the working directory. It runs before the install step.
func (c *cacheManager) Restore(ctx context.Context) {
	if !c.enabled() || c.store() == nil {
		return
	}

	c.opts.Reporter.AddStep("restoring build cache")

	restored := []string{}

	for _, dir := range c.dirs() {
		found, err := c.restoreDir(ctx, dir)

		if err != nil {
			c.reportError(fmt.Sprintf("could not restore build cache for %s", dir), err)
			continue
		}

		if found {
			restored = append(restored, dir)
		}
	}

	if len(restored) == 0 {
		c.opts.Reporter.AddLine("no build cache found - starting from a clean state")
		return
	}

	c.opts.Reporter.AddLine(fmt.Sprintf("restored cached directories: %v", restored))

	for _, dir := range restored {
		if hash, err := c.contentHash(ctx, dir); err != nil {
			slog.Errorf("could not hash restored build cache for %s: %v", dir, err)
		} else {
			c.restoredHash[dir] = hash
		}
	}
}

// restoreDir downloads and extracts the archive holding the given cache
// directory. It returns false with a nil error on a cache miss.
func (c *cacheManager) restoreDir(ctx context.Context, dir string) (bool, error) {
	found, err := c.store().DownloadCacheArtifact(ctx, c.artifactArgs(dir))

	if err != nil || !found {
		return false, err
	}

	defer os.Remove(c.archivePath(dir))

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "tar",
		Args: []string{"-xzf", c.archivePath(dir), "-C", c.opts.WorkDir},
		Dir:  c.opts.RootDir,
	})

	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("could not extract archive: %s: %w", string(out), err)
	}

	return true, nil
}

// contentHash digests the paths and contents of all files inside the given
// cache directory. It hashes contents rather than modification times, which
// package managers rewrite on every install even when nothing changed.
func (c *cacheManager) contentHash(ctx context.Context, dir string) (string, error) {
	hash := sha256.New()
	root := path.Join(c.opts.WorkDir, dir)

	if stat, err := os.Stat(root); err != nil || !stat.IsDir() {
		return hex.EncodeToString(hash.Sum(nil)), nil
	}

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(c.opts.WorkDir, p)

		if err != nil {
			return err
		}

		fmt.Fprintf(hash, "%s\x00", rel)

		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(p)

			if err != nil {
				return err
			}

			fmt.Fprintf(hash, "->%s\x00", target)

			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		f, err := os.Open(p)

		if err != nil {
			return err
		}

		defer f.Close()

		if _, err := io.Copy(hash, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Snapshot archives the cache directories and uploads them, replacing the
// environment's previous cache. It runs after a successful build. A directory
// whose contents are unchanged since Restore skips the archive and upload.
func (c *cacheManager) Snapshot(ctx context.Context) {
	if !c.enabled() || c.store() == nil {
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

	skipped := []string{}

	for _, dir := range dirs {
		hash, err := c.contentHash(ctx, dir)

		if err != nil {
			slog.Errorf("could not hash build cache for %s: %v", dir, err)
			hash = ""
		}

		if hash != "" && hash == c.restoredHash[dir] {
			skipped = append(skipped, dir)
			continue
		}

		c.snapshotDir(ctx, dir)
	}

	if len(skipped) > 0 {
		c.opts.Reporter.AddLine(fmt.Sprintf("build cache unchanged - skipping upload: %v", skipped))
	}
}

// snapshotDir archives the given cache directory and uploads it, replacing
// the directory's previous archive. Failures are reported to the build log;
// caching is best-effort.
func (c *cacheManager) snapshotDir(ctx context.Context, dir string) {
	defer os.Remove(c.archivePath(dir))

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "tar",
		Args: []string{"-czf", c.archivePath(dir), "-C", c.opts.WorkDir, dir},
		Dir:  c.opts.RootDir,
	})

	if out, err := cmd.CombinedOutput(); err != nil {
		c.reportError(fmt.Sprintf("could not archive build cache for %s", dir), fmt.Errorf("%s: %w", string(out), err))
		return
	}

	stat, err := os.Stat(c.archivePath(dir))

	if err != nil {
		c.reportError(fmt.Sprintf("could not stat build cache archive for %s", dir), err)
		return
	}

	if stat.Size() > maxCacheArchiveSize {
		c.opts.Reporter.AddLine(
			fmt.Sprintf(
				"build cache for %s is too large (%dMB > %dMB) - reduce the cache directories to enable caching",
				dir,
				stat.Size()>>20,
				int64(maxCacheArchiveSize)>>20,
			),
		)
		return
	}

	if err := c.store().UploadCacheArtifact(ctx, c.artifactArgs(dir)); err != nil {
		c.reportError(fmt.Sprintf("could not upload build cache for %s", dir), err)
		return
	}

	c.opts.Reporter.AddLine(fmt.Sprintf("saved build cache (%dMB): %s", stat.Size()>>20, dir))
}

func (c *cacheManager) reportError(msg string, err error) {
	slog.Errorf("%s: %v", msg, err)
	c.opts.Reporter.AddLine(fmt.Sprintf("%s - continuing without cache", msg))
}
