package integrations

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// CacheArtifactArgs identifies a build cache archive. Cache archives are
// scoped per environment: previews deploying to the same environment share
// a cache, while environments and applications never share one.
type CacheArtifactArgs struct {
	AppID      types.ID
	EnvID      types.ID
	BucketName string // Optional bucket override. Falls back to the provider's configured bucket.
	LocalPath  string // Absolute path of the local archive to upload from or download to.
}

// CacheStore is implemented by storage clients that can persist build cache
// archives between deployments. The runner type-asserts the configured client
// against this interface; providers that do not implement it simply skip
// build caching.
type CacheStore interface {
	// UploadCacheArtifact stores the archive at LocalPath, replacing any
	// previous cache for the same environment.
	UploadCacheArtifact(context.Context, CacheArtifactArgs) error

	// DownloadCacheArtifact fetches the environment's cache archive into
	// LocalPath. It returns false with a nil error on a cache miss.
	DownloadCacheArtifact(context.Context, CacheArtifactArgs) (bool, error)

	// DeleteCacheArtifact removes the environment's cache archive. Deleting
	// a missing cache is not an error.
	DeleteCacheArtifact(context.Context, CacheArtifactArgs) error
}
