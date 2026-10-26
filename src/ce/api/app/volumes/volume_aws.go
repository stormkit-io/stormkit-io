package volumes

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type vmAWS struct {
	BucketName string
	cli        *integrations.AWSClient
}

var CachedAWS *vmAWS

// clientAWS is a singleton function to return either the cached
// file system volume manager or create one from scratch.
func clientAWS(c *admin.VolumesConfig) *vmAWS {
	mux.Lock()
	defer mux.Unlock()

	if CachedAWS == nil {
		// If these keys are empty, make sure to use an IAM profile to grant access
		awscli, err := integrations.AWS(integrations.ClientArgs{
			AccessKey:   c.AccessKey,
			SecretKey:   c.SecretKey,
			Region:      c.Region,
			Middlewares: c.Middlewares,
		}, nil)

		if err != nil {
			panic(err)
		}

		CachedAWS = &vmAWS{
			BucketName: c.BucketName,
			cli:        awscli,
		}
	}

	return CachedAWS
}

func (c *vmAWS) download(file *File) (io.ReadSeeker, error) {
	bucket, keyPrefix := c.parseBucketInfo(file)

	f, err := c.cli.S3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &keyPrefix,
	})

	if err != nil {
		return nil, err
	}

	if f == nil || f.Body == nil {
		return nil, nil
	}

	defer f.Body.Close()

	// Read all data into memory
	data, err := io.ReadAll(f.Body)

	if err != nil {
		return nil, err
	}

	// Return as ReadSeeker
	return bytes.NewReader(data), nil
}

func (c *vmAWS) upload(args uploadArgs) (*File, error) {
	file := integrations.File{
		Pointer:      args.file,
		Size:         args.size,
		RelativePath: args.dstFileName,
		ContentType:  integrations.DetectContentType(args.dstFileName, args.file),
	}

	opts := integrations.S3Args{
		BucketName: c.BucketName,
		KeyPrefix:  args.dstFilePath,
	}

	err := c.cli.UploadFile(file, opts)

	if err != nil {
		return nil, err
	}

	return &File{
		Size:      args.size,
		Path:      path.Join(c.BucketName, args.dstFilePath, path.Dir(args.dstFileName)),
		Name:      args.dstFileName,
		EnvID:     args.envID,
		CreatedAt: utils.NewUnix(),
		Metadata: utils.Map{
			"mountType": AWSS3,
		},
	}, nil
}

// removeFiles deletes files from the S3 destination. This function
// supports deleting from different buckets. Files whose path carries no
// bucket are skipped and reported as an error.
func (c *vmAWS) removeFiles(files []*File) ([]*File, error) {
	objectsToDelete := map[string][]types.ObjectIdentifier{}
	filesByBucket := map[string][]*File{}

	var firstErr error

	for _, file := range files {
		bucket, keyPrefix := c.parseBucketInfo(file)

		if bucket == "" {
			if firstErr == nil {
				firstErr = fmt.Errorf("volume file %d has no bucket in its path", file.ID)
			}

			continue
		}

		objectsToDelete[bucket] = append(objectsToDelete[bucket], types.ObjectIdentifier{
			Key: utils.Ptr(keyPrefix),
		})

		filesByBucket[bucket] = append(filesByBucket[bucket], file)
	}

	removed := []*File{}

	for bucketName, objects := range objectsToDelete {
		_, err := c.cli.S3Client.DeleteObjects(context.Background(), &s3.DeleteObjectsInput{
			Bucket: utils.Ptr(bucketName),
			Delete: &types.Delete{
				Objects: objects,
				Quiet:   utils.Ptr(true),
			},
		})

		if err != nil {
			return removed, err
		}

		removed = append(removed, filesByBucket[bucketName]...)
	}

	return removed, firstErr
}

func (c *vmAWS) parseBucketInfo(file *File) (string, string) {
	pieces := strings.SplitN(file.Path, "/", 2)

	if len(pieces) != 2 {
		return "", ""
	}

	return pieces[0], path.Join(pieces[1], file.Name)
}
