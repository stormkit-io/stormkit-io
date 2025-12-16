//go:build alibaba

package integrations

import (
	"context"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	alibaba "github.com/alibabacloud-go/fc-20230330/v4/client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconf "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func init() {
	slog.Debug(slog.LogOpts{
		Msg:   "alibaba integration is enabled",
		Level: slog.DL1,
	})
}

type AlibabaSDK interface {
	GetFunction(*string, *alibaba.GetFunctionRequest) (*alibaba.GetFunctionResponse, error)
	InvokeFunction(*string, *alibaba.InvokeFunctionRequest) (*alibaba.InvokeFunctionResponse, error)
	UpdateFunction(*string, *alibaba.UpdateFunctionRequest) (*alibaba.UpdateFunctionResponse, error)
	DeleteFunctionVersion(*string, *string) (*alibaba.DeleteFunctionVersionResponse, error)
	PublishFunctionVersion(*string, *alibaba.PublishFunctionVersionRequest) (*alibaba.PublishFunctionVersionResponse, error)
	CreateFunction(*alibaba.CreateFunctionRequest) (*alibaba.CreateFunctionResponse, error)
}

type AlibabaClient struct {
	awsClient *AWSClient // Used for S3 Compatibility
	client    AlibabaSDK // Used for other operations
}

var CachedAlibabaClient *AlibabaClient
var DefaultAlibabaSDK AlibabaSDK

func Alibaba(args ClientArgs) (*AlibabaClient, error) {
	if CachedAlibabaClient != nil {
		return CachedAlibabaClient, nil
	}

	conf := config.Get()

	if conf.Alibaba == nil {
		return nil, fmt.Errorf("alibaba integration is not configured")
	}

	if args.AccessKey == "" {
		if conf.Runner != nil && conf.Runner.AccessKey != "" {
			args.AccessKey = conf.Runner.AccessKey
		} else {
			args.AccessKey = conf.Alibaba.AccessKey
		}
	}

	if args.SecretKey == "" {
		if conf.Runner != nil && conf.Runner.SecretKey != "" {
			args.SecretKey = conf.Runner.SecretKey
		} else {
			args.SecretKey = conf.Alibaba.SecretKey
		}
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "oss",
			URL:           fmt.Sprintf("https://oss-%s.aliyuncs.com", conf.Alibaba.Region),
			SigningRegion: conf.Alibaba.Region,
		}, nil
	})

	cfg, err := awsconf.LoadDefaultConfig(
		context.Background(),
		awsconf.WithEndpointResolverWithOptions(customResolver),
		awsconf.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			args.AccessKey,
			args.SecretKey,
			"",
		)),
		awsconf.WithRegion("auto"),
		awsconf.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)

	if err != nil {
		return nil, err
	}

	// Alibaba is S3 Compatible, so use that interface.
	awscli, err := AWS(
		ClientArgs{
			AccessKey:   args.AccessKey,
			SecretKey:   args.SecretKey,
			Middlewares: args.Middlewares,
		}, &AWSOptions{
			awsConf: &cfg,
			s3Only:  true,
		},
	)

	if err != nil {
		return nil, err
	}

	awscli.providerPrefix = "alibaba"
	endpoint := utils.Ptr(fmt.Sprintf("%s.%s.fc.aliyuncs.com", conf.Alibaba.AccountID, conf.Alibaba.Region))

	slog.Infof("using alibaba endpoint: %s", *endpoint)

	client := DefaultAlibabaSDK

	// Now register Alibaba Cloud SDK Client -- if not provided already with DefaultAlibabaSDK
	if client == nil {
		client, err = alibaba.NewClient(&openapi.Config{
			AccessKeyId:     &args.AccessKey,
			AccessKeySecret: &args.SecretKey,
			Endpoint:        endpoint,
		})

		if err != nil {
			return nil, err
		}
	}

	CachedAlibabaClient = &AlibabaClient{
		awsClient: awscli,
		client:    client,
	}

	return CachedAlibabaClient, nil
}

func (c AlibabaClient) Name() string {
	return "Alibaba"
}

// Upload uses AWS SDK under the hood to upload a file to OSS.
func (a AlibabaClient) Upload(args UploadArgs) (*UploadResult, error) {
	if args.ClientZip == "" && args.ServerZip == "" && args.APIZip == "" && args.MigrationsZip == "" {
		return nil, nil
	}

	var err error

	if args.BucketName == "" {
		args.BucketName = config.Get().Alibaba.StorageBucket
	}

	result := &UploadResult{}

	if result.Client, err = a.awsClient.uploadZipToS3(args.ClientZip, args); err != nil {
		return nil, err
	}

	if result.Migrations, err = a.awsClient.uploadZipToS3(args.MigrationsZip, args); err != nil {
		return nil, err
	}

	if args.ServerZip != "" {
		copy := args
		copy.funcType = FuncTypeRenderer
		copy.handler = args.ServerHandler
		copy.zip = args.ServerZip

		if result.Server, err = a.uploadToFunctions(copy); err != nil {
			return nil, err
		}
	}

	if args.APIZip != "" {
		copy := args
		copy.funcType = FuncTypeAPI
		copy.handler = args.APIHandler
		copy.zip = args.APIZip

		if result.API, err = a.uploadToFunctions(copy); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// DeleteArtifacts deletes all artifacts for the given deployment. Once this method is complete, there
// is no way to recover the deleted files.
func (a AlibabaClient) DeleteArtifacts(ctx context.Context, args DeleteArtifactsArgs) error {
	deleteFunctionVersion := func(location string) error {
		fnName, fnVersion := a.parseFunctionLocation(location)

		if fnName == "" {
			return fmt.Errorf("cannot delete function: invalid function name %s", args.FunctionLocation)
		}

		_, err := a.client.DeleteFunctionVersion(&fnName, &fnVersion)
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
		// alibaba:<bucket-name>/<app-id>/<deployment-id>
		location := strings.TrimPrefix(args.StorageLocation, "alibaba:")

		// <bucket-name>/<app-id>/<deployment-id>
		pieces := strings.Split(location, "/")

		if len(pieces) < 3 {
			return fmt.Errorf("invalid storage location provided: %s", args.StorageLocation)
		}

		bucketName := pieces[0]
		keyPrefix := fmt.Sprintf("%s/%s", pieces[1], pieces[2])

		if err := a.awsClient.deleteS3Folder(ctx, bucketName, keyPrefix); err != nil {
			return err
		}
	}

	return nil
}
