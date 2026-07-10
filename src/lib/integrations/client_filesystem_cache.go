package integrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

// cachePath returns the on-disk location of a build cache archive, stored
// under the deployer's storage directory next to the deployment artifacts.
// Each cache directory has its own archive.
func (c *FilesysClient) cachePath(args CacheArtifactArgs) string {
	return path.Join(
		config.Get().Deployer.StorageDir,
		"cache",
		fmt.Sprintf("app-%d", args.AppID),
		fmt.Sprintf("env-%d", args.EnvID),
		CacheDirToken(args.Dir)+".tar.gz",
	)
}

// UploadCacheArtifact implements CacheStore by copying the archive into the
// storage directory, replacing any previous cache for the environment.
func (c *FilesysClient) UploadCacheArtifact(ctx context.Context, args CacheArtifactArgs) error {
	dest := c.cachePath(args)

	if err := os.MkdirAll(path.Dir(dest), 0774); err != nil {
		return err
	}

	return file.Copy(args.LocalPath, dest, 0664)
}

// DownloadCacheArtifact implements CacheStore by copying the archive to
// LocalPath. It returns false with a nil error on a cache miss.
func (c *FilesysClient) DownloadCacheArtifact(ctx context.Context, args CacheArtifactArgs) (bool, error) {
	src := c.cachePath(args)

	if _, err := os.Stat(src); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	if err := file.Copy(src, args.LocalPath, 0664); err != nil {
		return false, err
	}

	return true, nil
}
