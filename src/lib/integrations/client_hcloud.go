package integrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconf "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"golang.org/x/sync/singleflight"
)

const hcloudPrefix = "hcloud"

var errHCloudArtifactMissing = errors.New("deployment artifact is not in the bucket")

// hcloudLocationPattern matches <bucket>/<app-id>/<deployment-id>/<name>.zip
// with an optional path inside the zip. Locations come back from the runner,
// so they are checked before they become paths on this server's disk.
var hcloudLocationPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]/[0-9]+/[0-9]+/[A-Za-z0-9_-]+\.zip(/[^\x00]+)?$`)

// HCloudClient stores deployment artifacts in Hetzner Object Storage and
// serves them from the local disk. A zip is downloaded and extracted the first
// time a server needs it, and the FilesysClient serves files and runs
// functions from there. The bucket is the source of truth, so every server of
// a multi-server instance can serve every deployment, whichever server built
// it.
//
// Locations look like hcloud:<bucket>/<app-id>/<deployment-id>/sk-client.zip,
// and functions append the handler: .../sk-server.zip/server.mjs:handler.
// Deployments made before the switch keep their local: or aws: locations and
// are still read, run and deleted where they were stored.
type HCloudClient struct {
	s3      *AWSClient
	local   *FilesysClient
	bucket  string
	root    string
	fetches singleflight.Group
}

var CachedHCloudClient *HCloudClient
var hcloudMux sync.Mutex

// HCloud returns the Hetzner Object Storage client. Settings in args win over
// the HCLOUD_OSS_* environment, so the runner can pass the ones it received
// with the deployment.
func HCloud(args ClientArgs) (*HCloudClient, error) {
	hcloudMux.Lock()
	defer hcloudMux.Unlock()

	if CachedHCloudClient != nil {
		return CachedHCloudClient, nil
	}

	settings := config.HCloudConfig{
		AccessKey: args.AccessKey,
		SecretKey: args.SecretKey,
		Region:    args.Region,
		Endpoint:  args.Endpoint,
	}

	if conf := config.Get().HCloud; conf != nil {
		settings.AccessKey = utils.GetString(settings.AccessKey, conf.AccessKey)
		settings.SecretKey = utils.GetString(settings.SecretKey, conf.SecretKey)
		settings.Region = utils.GetString(settings.Region, conf.Region)
		settings.Endpoint = utils.GetString(settings.Endpoint, conf.Endpoint)
		settings.StorageBucket = conf.StorageBucket
	}

	if settings.Region == "" && settings.Endpoint == "" {
		return nil, errors.New("hcloud storage needs HCLOUD_OSS_REGION or HCLOUD_OSS_ENDPOINT")
	}

	if settings.AccessKey == "" || settings.SecretKey == "" {
		return nil, errors.New("hcloud storage needs HCLOUD_OSS_ACCESS_KEY_ID and HCLOUD_OSS_SECRET_ACCESS_KEY")
	}

	cfg, err := awsconf.LoadDefaultConfig(
		context.Background(),
		awsconf.WithBaseEndpoint(settings.StorageEndpoint()),
		awsconf.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			settings.AccessKey,
			settings.SecretKey,
			"",
		)),
		awsconf.WithRegion(utils.GetString(settings.Region, "auto")),
		awsconf.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)

	if err != nil {
		return nil, err
	}

	// Not the cached AWS client: that one may serve S3 buckets of its own,
	// such as volumes or deployments made before the switch.
	awscli, err := newAWSClient(args, &AWSOptions{
		awsConf:            &cfg,
		s3Only:             true,
		noEncryptionHeader: true,
		pathStyle:          true,
	})

	if err != nil {
		return nil, err
	}

	awscli.providerPrefix = hcloudPrefix

	root := config.Get().Deployer.StorageDir

	if root == "" {
		root = path.Join(os.TempDir(), "stormkit")
	}

	CachedHCloudClient = &HCloudClient{
		s3:     awscli,
		local:  Filesys(),
		bucket: settings.StorageBucket,
		root:   path.Join(root, hcloudPrefix),
	}

	return CachedHCloudClient, nil
}

func (c *HCloudClient) Name() string {
	return "HCloud"
}

// legacy returns the client of a location stored before the switch to HCloud.
func (c *HCloudClient) legacy(location string) (ClientInterface, error) {
	switch {
	case strings.HasPrefix(location, "local:"):
		return c.local, nil

	case strings.HasPrefix(location, "aws:"):
		if config.Get().AWS == nil {
			return nil, fmt.Errorf("%s is stored on aws, which is not configured", location)
		}

		return AWS(ClientArgs{}, nil)
	}

	return nil, nil
}

// Upload stores the deployment zips in the bucket as they are. Servers
// extract them when they first need them.
func (c *HCloudClient) Upload(args UploadArgs) (*UploadResult, error) {
	if args.ClientZip == "" && args.ServerZip == "" && args.APIZip == "" && args.MigrationsZip == "" {
		return nil, nil
	}

	args.BucketName = utils.GetString(args.BucketName, c.bucket)

	if args.BucketName == "" {
		return nil, errors.New("hcloud storage needs HCLOUD_OSS_BUCKET_NAME")
	}

	var err error

	result := &UploadResult{}

	if result.Client, err = c.s3.uploadZipToS3(args.ClientZip, args); err != nil {
		return nil, err
	}

	if result.Server, err = c.uploadFunction(uploadFunctionParams{zip: args.ServerZip, handler: args.ServerHandler, args: args}); err != nil {
		return nil, err
	}

	if result.API, err = c.uploadFunction(uploadFunctionParams{zip: args.APIZip, handler: args.APIHandler, args: args}); err != nil {
		return nil, err
	}

	if result.Migrations, err = c.s3.uploadZipToS3(args.MigrationsZip, args); err != nil {
		return nil, err
	}

	return result, nil
}

type uploadFunctionParams struct {
	zip     string
	handler string
	args    UploadArgs
}

// uploadFunction uploads a function zip and points its location at the
// handler inside it.
func (c *HCloudClient) uploadFunction(p uploadFunctionParams) (UploadOverview, error) {
	overview, err := c.s3.uploadZipToS3(p.zip, p.args)

	if err != nil || overview.Location == "" || p.handler == "" {
		return overview, err
	}

	overview.Location = overview.Location + "/" + p.handler

	return overview, nil
}

// GetFile returns a file of a deployment. A location that names a zip
// without a file inside it, like the migrations, returns the zip itself.
func (c *HCloudClient) GetFile(args GetFileArgs) (*GetFileResult, error) {
	if legacy, err := c.legacy(args.Location); err != nil || legacy != nil {
		if err != nil {
			return nil, err
		}

		return legacy.GetFile(args)
	}

	loc, err := c.parse(args.Location)

	if err != nil {
		return nil, err
	}

	if loc.inner == "" && args.FileName == "" {
		return c.s3.getFile(GetFileArgs{Location: loc.bucket + "/" + loc.key})
	}

	dir, err := c.extract(loc)

	if errors.Is(err, errHCloudArtifactMissing) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return c.local.GetFile(GetFileArgs{
		Location:     "local:" + path.Join(dir, loc.inner),
		FileName:     args.FileName,
		DeploymentID: args.DeploymentID,
	})
}

// Invoke runs a function from its extracted copy on this server.
func (c *HCloudClient) Invoke(args InvokeArgs) (*InvokeResult, error) {
	if legacy, err := c.legacy(args.ARN); err != nil || legacy != nil {
		if err != nil {
			return nil, err
		}

		return legacy.Invoke(args)
	}

	loc, err := c.parse(args.ARN)

	if err != nil {
		return nil, err
	}

	dir, err := c.extract(loc)

	if err != nil {
		return nil, err
	}

	// Long-running servers stay registered under the stored location, the
	// name hosting uses to find them again, for instance to stop a deployment
	// that failed its warm-up.
	if args.Command != "" {
		fnPath, _, _ := strings.Cut(loc.inner, ":")

		return c.local.ProcessManager().Invoke(args, path.Dir(path.Join(dir, fnPath)))
	}

	args.ARN = "local:" + path.Join(dir, loc.inner)

	return c.local.Invoke(args)
}

// DeleteArtifacts removes the deployment from the bucket and from this
// server's disk. Copies extracted on other servers stay until they are
// cleaned up there.
func (c *HCloudClient) DeleteArtifacts(ctx context.Context, args DeleteArtifactsArgs) error {
	// A deployment's artifacts all live with one provider.
	location := utils.GetString(args.StorageLocation, args.FunctionLocation, args.APILocation)

	if legacy, err := c.legacy(location); err != nil || legacy != nil {
		if err != nil {
			return err
		}

		return legacy.DeleteArtifacts(ctx, args)
	}

	deleted := map[string]bool{}

	for _, location := range []string{args.StorageLocation, args.FunctionLocation, args.APILocation} {
		if location == "" {
			continue
		}

		loc, err := c.parse(location)

		if err != nil {
			return err
		}

		// <app-id>/<deployment-id>/<zip>
		pieces := strings.Split(loc.key, "/")

		if len(pieces) < 3 {
			return fmt.Errorf("invalid hcloud location: %s", location)
		}

		prefix := path.Join(loc.bucket, pieces[0], pieces[1])

		if deleted[prefix] {
			continue
		}

		deleted[prefix] = true

		if err := c.s3.deleteS3Folder(ctx, loc.bucket, path.Join(pieces[0], pieces[1])); err != nil {
			return err
		}

		dir := path.Join(c.root, prefix)

		// The separator keeps deployment 7 from matching 70 and 700.
		c.local.files.dropPrefix(dir + "/")

		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	return nil
}

type hcloudLocation struct {
	bucket string
	key    string // Object key of the zip
	inner  string // Path inside the zip, with the handler for functions
}

// parse splits hcloud:<bucket>/<key>.zip[/<inner>] into its parts. It
// refuses anything that could resolve outside the local storage directory.
func (c *HCloudClient) parse(location string) (hcloudLocation, error) {
	rest := strings.TrimPrefix(location, hcloudPrefix+":")

	if strings.Contains(rest, "..") || !hcloudLocationPattern.MatchString(rest) {
		return hcloudLocation{}, fmt.Errorf("invalid hcloud location: %s", location)
	}

	bucket, key, _ := strings.Cut(rest, "/")
	end := strings.Index(key, ".zip")

	return hcloudLocation{
		bucket: bucket,
		key:    key[:end+len(".zip")],
		inner:  strings.TrimPrefix(key[end+len(".zip"):], "/"),
	}, nil
}

// localDir returns where the zip is extracted on this server.
func (c *HCloudClient) localDir(loc hcloudLocation) string {
	return path.Join(c.root, loc.bucket, strings.TrimSuffix(loc.key, ".zip"))
}

// extract downloads and extracts the zip once per server and returns the
// directory holding its files. Concurrent requests share one download, and
// the files appear all at once, so a request never sees half a deployment.
func (c *HCloudClient) extract(loc hcloudLocation) (string, error) {
	dir := c.localDir(loc)

	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}

	_, err, _ := c.fetches.Do(dir, func() (any, error) {
		if _, err := os.Stat(dir); err == nil {
			return nil, nil
		}

		if err := os.MkdirAll(path.Dir(dir), 0774); err != nil {
			return nil, err
		}

		tmp, err := os.MkdirTemp(path.Dir(dir), ".fetch-")

		if err != nil {
			return nil, err
		}

		defer os.RemoveAll(tmp)

		zipPath := path.Join(tmp, path.Base(loc.key))

		if err := c.download(downloadParams{loc: loc, dest: zipPath}); err != nil {
			return nil, err
		}

		files := path.Join(tmp, "files")

		if err := file.Unzip(file.UnzipOpts{ZipFile: zipPath, ExtractDir: files}); err != nil {
			return nil, err
		}

		// Another process on this server may have extracted it meanwhile.
		if err := os.Rename(files, dir); err != nil {
			if _, statErr := os.Stat(dir); statErr == nil {
				return nil, nil
			}

			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		return "", err
	}

	return dir, nil
}

type downloadParams struct {
	loc  hcloudLocation
	dest string
}

func (c *HCloudClient) download(p downloadParams) error {
	f, err := os.Create(p.dest)

	if err != nil {
		return err
	}

	defer f.Close()

	_, err = c.s3.downloader.Download(context.Background(), f, &s3.GetObjectInput{
		Bucket: utils.Ptr(p.loc.bucket),
		Key:    utils.Ptr(p.loc.key),
	})

	var nsk *s3types.NoSuchKey

	if errors.As(err, &nsk) {
		return errHCloudArtifactMissing
	}

	return err
}
