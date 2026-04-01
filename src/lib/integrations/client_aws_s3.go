package integrations

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

type S3Args struct {
	BucketName string
	KeyPrefix  string
	ACL        s3types.ObjectCannedACL
}

func (a *AWSClient) getFile(args GetFileArgs) (*GetFileResult, error) {
	bucketName, keyPrefix := a.parseS3Location(args.Location)

	out, err := a.S3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &keyPrefix,
	})

	if err != nil {
		var nsk *s3types.NoSuchKey

		if errors.As(err, &nsk) {
			return nil, nil
		}

		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, out.Body); err != nil {
		return nil, err
	}

	content := buf.Bytes()
	contentType := ""
	contentLength := int64(0)

	if out.ContentType != nil {
		contentType = *out.ContentType
	} else {
		contentType = DetectContentType(keyPrefix, content)
	}

	if out.ContentLength != nil {
		contentLength = *out.ContentLength
	}

	return &GetFileResult{
		Content:     content,
		ContentType: contentType,
		Size:        contentLength,
	}, nil
}

// ZipDownloader downloads the zip file with the given bucket and keyprefix.
// If the folder has been previously created, it returns the path immediately.
// If not, †his method will create a temp folder, download the zip in there,
// unzip it and remove the zip file.
func (a *AWSClient) ZipDownloader(deploymentID, bucket, keyprefix string) (string, error) {
	// First, check if the folder exists
	folder := fmt.Sprintf("d-%s", deploymentID)
	path := filepath.Join(os.TempDir(), folder)

	stat, err := os.Stat(path)

	if err == nil && stat != nil {
		return path, nil
	}

	err = os.Mkdir(path, 0775)

	if err != nil {
		return "", err
	}

	zipPath := filepath.Join(path, "sk-client.zip")
	f, err := os.Create(zipPath)

	if err != nil {
		return "", err
	}

	defer f.Close()

	n, err := a.downloader.Download(context.TODO(), f, &s3.GetObjectInput{
		Bucket: utils.Ptr(bucket),
		Key:    utils.Ptr(keyprefix),
	})

	if n == 0 {
		return "", errors.New("did not download any file")
	}

	unzipOpts := file.UnzipOpts{
		ZipFile:    zipPath,
		ExtractDir: path,
		LowerCase:  false,
	}

	if err := file.Unzip(unzipOpts); err != nil {
		return "", err
	}

	if err := os.Remove(zipPath); err != nil {
		return "", err
	}

	return path, err
}

func (a *AWSClient) serveFromZip(args GetFileArgs) (*GetFileResult, error) {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.zipManager == nil {
		a.zipManager = NewZipManager(a.ZipDownloader)
	}

	return a.zipManager.GetFile(args)
}

// GetFile returns a file from the bucket.
func (a *AWSClient) GetFile(args GetFileArgs) (*GetFileResult, error) {
	// The serveFromZip method is used to serve files from the sk-client.zip file.
	// For other files, we use the getFile method.
	if strings.HasSuffix(args.Location, "sk-client.zip") {
		return a.serveFromZip(args)
	}

	// For other files, we append the file name to the location and use getFile.
	// If FileName is empty, it means the location already points to the file.
	// We have to use path.Join because S3 uses "/" as separator.
	if args.FileName != "" {
		args.Location = path.Join(args.Location, args.FileName)
	}

	return a.getFile(args)
}

func (a *AWSClient) bucketName(args UploadArgs) string {
	bucketName := args.BucketName

	if args.BucketName == "" {
		bucketName = config.Get().AWS.StorageBucket
	}

	return bucketName
}

func (a *AWSClient) uploadZipToS3(pathToZip string, args UploadArgs) (UploadOverview, error) {
	if pathToZip == "" {
		return UploadOverview{}, nil
	}

	keyPrefix := fmt.Sprintf("%d/%d", args.AppID, args.DeploymentID)
	bucketName := a.bucketName(args)
	result := UploadOverview{}

	content, err := os.ReadFile(pathToZip)

	if err != nil {
		return result, err
	}

	stat, err := os.Stat(pathToZip)

	if err != nil {
		return result, err
	}

	zipName := path.Base(pathToZip)
	size := stat.Size()
	file := File{
		Content:      content,
		ContentType:  DetectContentType(args.ClientZip, content),
		RelativePath: zipName, // to the key prefix
		Size:         size,
	}

	err = a.UploadFile(file, S3Args{
		BucketName: bucketName,
		KeyPrefix:  keyPrefix,
		ACL:        s3types.ObjectCannedACLPrivate,
	})

	if err != nil {
		return result, err
	}

	result.BytesUploaded = size
	result.FilesUploaded = 1
	result.Location = fmt.Sprintf("%s:%s/%s/%s", a.providerPrefix, bucketName, keyPrefix, zipName)
	return result, err
}

// UploadFile uploads a single file to S3 destination.
func (a *AWSClient) UploadFile(file File, s3args any) error {
	opts := s3args.(S3Args)
	filePath := filepath.Join(opts.KeyPrefix, file.RelativePath)

	if opts.ACL == "" {
		opts.ACL = s3types.ObjectCannedACLPrivate
	}

	input := &s3.PutObjectInput{
		Bucket:               &opts.BucketName,
		Key:                  &filePath,
		ContentType:          &file.ContentType,
		ContentLength:        &file.Size,
		ServerSideEncryption: s3types.ServerSideEncryptionAes256,
		// This is a required to allow reading the file through our CDN,
		// but it may create problems with custom storages.
		// Therefore, we may want to make this variable.
		ACL: opts.ACL,
	}

	if file.Content != nil {
		input.Body = bytes.NewReader(file.Content)
	} else {
		input.Body = file.Pointer
	}

	_, err := a.uploader.Upload(context.Background(), input)
	return err
}

// deleteS3Folder lists and deletes all objects under keyPrefix in the given bucket.
// It pages through the listing (each page holds up to 1,000 keys) so folders with
// more than 1,000 objects are fully deleted. Object entries with a nil or empty Key
// are skipped to avoid a MissingArgument error from S3-compatible providers such as
// Alibaba OSS.
func (a *AWSClient) deleteS3Folder(ctx context.Context, bucketName, keyPrefix string) error {
	// Ensure the prefix ends with "/" to avoid matching keys from deployments
	// whose IDs share the same numeric prefix (e.g. "1/50919" would otherwise
	// also match "1/509190/...").
	if !strings.HasSuffix(keyPrefix, "/") {
		keyPrefix += "/"
	}

	var continuationToken *string

	for {
		listResp, err := a.S3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            utils.Ptr(bucketName),
			Prefix:            utils.Ptr(keyPrefix),
			ContinuationToken: continuationToken,
		})

		if err != nil {
			return fmt.Errorf("unable to list objects in bucket %q with prefix %q: %w", bucketName, keyPrefix, err)
		}

		var objectsToDelete []s3types.ObjectIdentifier

		for _, object := range listResp.Contents {
			if object.Key != nil && *object.Key != "" {
				objectsToDelete = append(objectsToDelete, s3types.ObjectIdentifier{
					Key: object.Key,
				})
			}
		}

		if len(objectsToDelete) > 0 {
			_, err = a.S3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: utils.Ptr(bucketName),
				Delete: &s3types.Delete{
					Objects: objectsToDelete,
					Quiet:   utils.Ptr(true),
				},
			})

			if err != nil {
				return fmt.Errorf("unable to delete objects in bucket %q with prefix %q: %w", bucketName, keyPrefix, err)
			}
		}

		if listResp.IsTruncated == nil || !*listResp.IsTruncated {
			break
		}

		if listResp.NextContinuationToken == nil || *listResp.NextContinuationToken == "" {
			return fmt.Errorf("truncated S3 ListObjectsV2 response for bucket %q and prefix %q is missing NextContinuationToken", bucketName, keyPrefix)
		}

		continuationToken = listResp.NextContinuationToken
	}

	return nil
}

// parseS3Location parses a string in the following format:
// aws:/bucket-name/path-to-file
func (a *AWSClient) parseS3Location(location string) (string, string) {
	pieces := strings.Split(strings.TrimPrefix(location, a.providerPrefix+":"), "/")
	return pieces[0], filepath.Join(pieces[1:]...)
}
