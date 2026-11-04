package integrations

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconf "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

type AWSOptions struct {
	s3Only  bool
	awsConf *aws.Config

	// noEncryptionHeader leaves out the SSE-S3 header for S3-compatible
	// stores that encrypt at rest on their own or reject the header.
	noEncryptionHeader bool

	// pathStyle puts the bucket in the path instead of the host name.
	pathStyle bool
}

type AWSClient struct {
	route53Client  *route53.Client
	lambdaClient   *lambda.Client
	S3Client       *s3.Client
	uploader       *manager.Uploader
	downloader     *manager.Downloader
	zipManager     *ZipManager
	zipOnce        sync.Once
	providerPrefix string
	encryption     s3types.ServerSideEncryption
}

var CachedAWSClient *AWSClient
var awsmux sync.Mutex

// AWS returns the process-wide AWS client, creating it on the first call.
func AWS(args ClientArgs, opts *AWSOptions) (*AWSClient, error) {
	awsmux.Lock()
	defer awsmux.Unlock()

	if CachedAWSClient != nil {
		return CachedAWSClient, nil
	}

	client, err := newAWSClient(args, opts)

	if err != nil {
		return nil, err
	}

	CachedAWSClient = client

	return CachedAWSClient, nil
}

// newAWSClient creates an AWS client that is not shared. S3-compatible
// providers that must not take over the cached AWS client use it.
func newAWSClient(args ClientArgs, opts *AWSOptions) (*AWSClient, error) {
	conf := config.Get()

	if opts == nil {
		opts = &AWSOptions{}
	}

	if args.Region == "" && conf.AWS != nil {
		args.Region = conf.AWS.Region
	}

	if args.AccessKey == "" && args.SecretKey == "" {
		if accessKey := os.Getenv("AWS_ACCESS_KEY_ID"); accessKey != "" {
			args.AccessKey = accessKey
		}

		if secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY"); secretKey != "" {
			args.SecretKey = secretKey
		}
	}

	var awsConfig aws.Config
	var err error

	ctxbkgr := context.Background()
	retrier := awsconf.WithRetryer(func() aws.Retryer {
		return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
	})

	if opts.awsConf != nil {
		awsConfig = *opts.awsConf
	} else if args.AccessKey != "" && args.SecretKey != "" {
		awsConfig, err = awsconf.LoadDefaultConfig(
			ctxbkgr,
			awsconf.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				args.AccessKey,
				args.SecretKey,
				args.SessionToken,
			)),
			awsconf.WithRegion(args.Region),
			retrier,
		)
	} else {
		awsConfig, err = awsconf.LoadDefaultConfig(
			ctxbkgr,
			awsconf.WithSharedConfigProfile(args.Profile),
			awsconf.WithRegion(args.Region),
			retrier,
		)
	}

	if err != nil {
		return nil, err
	}

	if args.Middlewares != nil {
		awsConfig.APIOptions = append(awsConfig.APIOptions, args.Middlewares...)
	}

	client := &AWSClient{
		S3Client: s3.NewFromConfig(awsConfig, func(o *s3.Options) {
			o.DisableLogOutputChecksumValidationSkipped = true
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			o.UsePathStyle = opts.pathStyle
		}),
		providerPrefix: "aws",
		encryption:     s3types.ServerSideEncryptionAes256,
	}

	if opts.noEncryptionHeader {
		client.encryption = ""
	}

	if !opts.s3Only {
		client.lambdaClient = lambda.NewFromConfig(awsConfig)
		client.route53Client = route53.NewFromConfig(awsConfig)
	}

	client.uploader = manager.NewUploader(client.S3Client)
	client.downloader = manager.NewDownloader(client.S3Client)

	return client, nil
}

func (c *AWSClient) Name() string {
	return "AWS"
}

// Route53 returns the route53 client.
func (c *AWSClient) Route53() *route53.Client {
	return c.route53Client
}

// Upload the deployment artifacts to the configured destinations.
func (c *AWSClient) Upload(args UploadArgs) (*UploadResult, error) {
	if args.ClientZip == "" && args.ServerZip == "" && args.APIZip == "" && args.MigrationsZip == "" {
		return nil, nil
	}

	var err error
	result := &UploadResult{}

	if result.Client, err = c.uploadZipToS3(args.ClientZip, args); err != nil {
		return nil, err
	}

	if result.Migrations, err = c.uploadZipToS3(args.MigrationsZip, args); err != nil {
		return nil, err
	}

	if c.lambdaClient == nil {
		return result, nil
	}

	if args.ServerZip != "" {
		copy := args
		copy.funcType = FuncTypeRenderer
		copy.handler = args.ServerHandler
		copy.zip = args.ServerZip

		if result.Server, err = c.uploadToLambda(copy); err != nil {
			return nil, err
		}
	}

	if args.APIZip != "" {
		copy := args
		copy.funcType = FuncTypeAPI
		copy.handler = args.APIHandler
		copy.zip = args.APIZip

		if result.API, err = c.uploadToLambda(copy); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// DeleteArtifacts deletes all artifacts for the given deployment. Once this method is complete, there
// is no way to recover the deleted files.
func (a *AWSClient) DeleteArtifacts(ctx context.Context, args DeleteArtifactsArgs) error {
	deleteFunctionVersion := func(location string) error {
		fnName, fnVersion := a.parseFunctionLocation(location)

		if fnName == "" {
			return fmt.Errorf("cannot delete function: invalid function name %s", location)
		}

		input := &lambda.DeleteFunctionInput{
			FunctionName: &fnName,
		}

		if fnVersion != "" {
			input.Qualifier = &fnVersion
		}

		_, err := a.lambdaClient.DeleteFunction(ctx, input)
		return err
	}

	if args.FunctionLocation != "" {
		if err := deleteFunctionVersion(args.FunctionLocation); err != nil {
			return err
		}
	}

	if args.APILocation != "" {
		if err := deleteFunctionVersion(args.APILocation); err != nil {
			return err
		}
	}

	if args.StorageLocation != "" {
		// aws:<bucket-name>/<app-id>/<deployment-id>
		location := strings.TrimPrefix(args.StorageLocation, "aws:")

		// <bucket-name>/<app-id>/<deployment-id>/
		pieces := strings.Split(location, "/")

		if len(pieces) < 3 {
			return fmt.Errorf("invalid storage location provided: %s", args.StorageLocation)
		}

		bucketName := pieces[0]
		keyPrefix := fmt.Sprintf("%s/%s", pieces[1], pieces[2])

		if err := a.deleteS3Folder(ctx, bucketName, keyPrefix); err != nil {
			return err
		}
	}

	return nil
}
