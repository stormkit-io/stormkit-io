package integrations_test

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"testing"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stretchr/testify/suite"
)

type AwsLambdaSuite struct {
	suite.Suite
	*factory.Factory

	conn   databasetest.TestDB
	tmpdir string
}

func (s *AwsLambdaSuite) SetupSuite() {
	setAwsEnvVars()
}

func (s *AwsLambdaSuite) BeforeTest(suiteName, _ string) {
	integrations.CachedAWSClient = nil

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	tmpDir, err := os.MkdirTemp("", "tmp-integrations-aws-")

	s.NoError(err)

	s.tmpdir = tmpDir
	clientDir := path.Join(tmpDir, "client")

	s.NoError(os.MkdirAll(clientDir, 0774))
	s.NoError(os.WriteFile(path.Join(clientDir, "index.html"), []byte("Hello world"), 0664))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-server.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-api.zip")}))
}

func (s *AwsLambdaSuite) AfterTest(_, _ string) {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}

	integrations.CachedAWSClient = nil

	s.conn.CloseTx()
}

func (s *AwsLambdaSuite) Test_Upload() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	aws, err := integrations.AWS(integrations.ClientArgs{
		SessionToken: "my-session",
		AccessKey:    "my-access-key",
		SecretKey:    "my-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.PutObjectInput:
							s.Equal("my-s3-bucket", *v.Bucket)
							s.Equal("1/50919/sk-server.zip", *v.Key)
							s.Equal(s3types.ServerSideEncryptionAes256, v.ServerSideEncryption)
							s.Greater(*v.ContentLength, int64(0))
						case *lambda.GetFunctionConfigurationInput:
							s.Equal("arn:aws:lambda:eu-central-1:123456789:function:1-1", *v.FunctionName)
						case *lambda.UpdateFunctionConfigurationInput:
							s.Equal("stormkit-server.handler", *v.Handler)
							s.Equal("arn:aws:lambda:eu-central-1:123456789:function:1-1", *v.FunctionName)
						case *lambda.UpdateFunctionCodeInput:
							s.Equal("arn:aws:lambda:eu-central-1:123456789:function:1-1", *v.FunctionName)
							s.Equal("my-s3-bucket", *v.S3Bucket)
							s.Equal("1/50919/sk-server.zip", *v.S3Key)
						case *lambda.CreateFunctionInput:
							s.Equal("arn:aws:lambda:eu-central-1:123456789:function:1-1", *v.FunctionName)
							s.Equal("my-s3-bucket", *v.Code.S3Bucket)
							s.Equal("1/50919/sk-server.zip", *v.Code.S3Key)
						default:
							s.NoError(errors.New("unknown call"))
						}

						return next.HandleInitialize(ctx, fi)
					}),
					middleware.Before,
				)
			},
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.FinalizeInput, fh middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)

						if opName == "PutObject" {
							return middleware.FinalizeOutput{
								Result: &s3.PutObjectOutput{},
							}, middleware.Metadata{}, nil
						}

						if opName == "GetFunctionConfiguration" {
							return middleware.FinalizeOutput{
								Result: &lambda.GetFunctionConfigurationOutput{
									Handler: utils.Ptr("old-handler-name"),
								},
							}, middleware.Metadata{}, nil
						}

						if opName == "UpdateFunctionConfiguration" {
							return middleware.FinalizeOutput{
								Result: &lambda.UpdateFunctionConfigurationOutput{
									LastUpdateStatus: types.LastUpdateStatusSuccessful,
								},
							}, middleware.Metadata{}, nil
						}

						if opName == "UpdateFunctionCode" {
							return middleware.FinalizeOutput{
								Result: &lambda.UpdateFunctionCodeOutput{},
							}, middleware.Metadata{}, &types.ResourceNotFoundException{}
						}

						if opName == "CreateFunction" {
							return middleware.FinalizeOutput{
								Result: &lambda.CreateFunctionOutput{
									Version: utils.Ptr("15"),
								},
							}, middleware.Metadata{}, nil
						}

						s.NoError(errors.New("unknown call"))

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	}, nil)

	s.NoError(err)
	s.NotNil(aws)

	result, err := aws.Upload(integrations.UploadArgs{
		AppID:         app.ID,
		EnvID:         env.ID,
		DeploymentID:  50919,
		ServerZip:     path.Join(s.tmpdir, "sk-server.zip"),
		ServerHandler: "stormkit-server.js:handler",
		BucketName:    "my-s3-bucket",
	})

	s.NoError(err)
	s.Empty(result.API.BytesUploaded)
	s.Empty(result.Client.BytesUploaded)
	s.Greater(result.Server.BytesUploaded, int64(0))
	s.Equal("aws:arn:aws:lambda:eu-central-1:123456789:function:1-1/15", result.Server.Location)
	s.Equal("", result.API.Location)
	s.Equal("", result.Client.Location)
}

func TestAwsLambda(t *testing.T) {
	suite.Run(t, &AwsLambdaSuite{})
}
