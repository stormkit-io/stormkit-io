package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// CacheArtifactArgs identifies a build cache archive. Cache archives are
// scoped per environment and cache directory: previews deploying to the same
// environment share a cache, while environments and applications never share
// one. Each cache directory is stored as its own archive so an unchanged
// directory can skip the upload independently.
type CacheArtifactArgs struct {
	AppID      types.ID
	EnvID      types.ID
	Dir        string // Cache directory held by the archive.
	BucketName string // Optional bucket override. Falls back to the provider's configured bucket.
	LocalPath  string // Absolute path of the local archive to upload from or download to.
}

// CacheDirToken returns a storage-key-safe identifier for a cache directory.
// Directories such as ".next/cache" contain path separators, so the key uses
// a digest of the directory path instead of the path itself.
func CacheDirToken(dir string) string {
	sum := sha256.Sum256([]byte(dir))

	return hex.EncodeToString(sum[:8])
}

// CacheStore is implemented by storage clients that can persist build cache
// archives between deployments. The runner type-asserts the configured client
// against this interface; providers that do not implement it simply skip
// build caching.
type CacheStore interface {
	// UploadCacheArtifact stores the archive at LocalPath, replacing any
	// previous cache for the same environment and directory.
	UploadCacheArtifact(context.Context, CacheArtifactArgs) error

	// DownloadCacheArtifact fetches the cache archive into LocalPath. It
	// returns false with a nil error on a cache miss.
	DownloadCacheArtifact(context.Context, CacheArtifactArgs) (bool, error)
}
