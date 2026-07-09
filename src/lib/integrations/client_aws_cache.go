package integrations

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const cacheContentType = "application/gzip"

// cacheKey returns the S3 key for an environment's build cache archive.
// It lives under the app prefix (next to `{appID}/{deploymentID}/...`
// deployment artifacts) so deleting the app prefix also removes caches.
// The literal "cache" segment cannot collide with numeric deployment IDs.
func (a *AWSClient) cacheKey(args CacheArtifactArgs) string {
	return fmt.Sprintf("%d/cache/env-%d.tar.gz", args.AppID, args.EnvID)
}

func (a *AWSClient) cacheBucket(args CacheArtifactArgs) string {
	bucketName := args.BucketName

	if bucketName == "" && config.Get().AWS != nil {
		bucketName = config.Get().AWS.StorageBucket
	}

	return bucketName
}

// UploadCacheArtifact implements CacheStore by streaming the archive to S3,
// replacing any previous cache for the environment.
func (a *AWSClient) UploadCacheArtifact(ctx context.Context, args CacheArtifactArgs) error {
	f, err := os.Open(args.LocalPath)

	if err != nil {
		return err
	}

	defer f.Close()

	_, err = a.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:               utils.Ptr(a.cacheBucket(args)),
		Key:                  utils.Ptr(a.cacheKey(args)),
		Body:                 f,
		ContentType:          utils.Ptr(cacheContentType),
		ServerSideEncryption: s3types.ServerSideEncryptionAes256,
		ACL:                  s3types.ObjectCannedACLPrivate,
	})

	return err
}

// DownloadCacheArtifact implements CacheStore by downloading the archive
// into LocalPath. It returns false with a nil error on a cache miss.
func (a *AWSClient) DownloadCacheArtifact(ctx context.Context, args CacheArtifactArgs) (bool, error) {
	f, err := os.Create(args.LocalPath)

	if err != nil {
		return false, err
	}

	defer f.Close()

	_, err = a.downloader.Download(ctx, f, &s3.GetObjectInput{
		Bucket: utils.Ptr(a.cacheBucket(args)),
		Key:    utils.Ptr(a.cacheKey(args)),
	})

	if err != nil {
		os.Remove(args.LocalPath)

		var nsk *s3types.NoSuchKey

		if errors.As(err, &nsk) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// DeleteCacheArtifact implements CacheStore. Deleting a missing cache is a
// no-op on S3, so this is safe to call unconditionally.
func (a *AWSClient) DeleteCacheArtifact(ctx context.Context, args CacheArtifactArgs) error {
	_, err := a.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: utils.Ptr(a.cacheBucket(args)),
		Key:    utils.Ptr(a.cacheKey(args)),
	})

	return err
}
