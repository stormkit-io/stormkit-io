//go:build alibaba

package integrations_test

import (
	"context"
	"errors"
	"os"
	"path"
	"testing"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type AlibabaSuite struct {
	suite.Suite

	tmpdir   string
	mockExec *mocks.CommandInterface
}

func (s *AlibabaSuite) SetupSuite() {
	setAlibabaEnvVars()
}

func (s *AlibabaSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "alibaba-test-")
	s.NoError(err)
	s.tmpdir = tmpDir

	integrations.CachedAlibabaClient = nil
	integrations.CachedAWSClient = nil
}

func (s *AlibabaSuite) AfterTest(_, _ string) {
	if s.tmpdir != "" {
		os.RemoveAll(s.tmpdir)
	}

	integrations.CachedAlibabaClient = nil
	integrations.CachedAWSClient = nil
}

func (s *AlibabaSuite) TearDownSuite() {
	config.Get().Alibaba = nil
}

func (s *AlibabaSuite) Test_Client() {
	client, err := integrations.Alibaba(integrations.ClientArgs{})
	s.NoError(err)
	s.NotNil(client)
}

func (s *AlibabaSuite) Test_Upload() {
	client, err := integrations.Alibaba(integrations.ClientArgs{
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.PutObjectInput:
							s.Equal("test-bucket", *v.Bucket)
							s.Equal(s3types.ServerSideEncryptionAes256, v.ServerSideEncryption)
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

						s.NoError(errors.New("unknown call"))

						return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	})
	s.NoError(err)

	clientZip := path.Join(s.tmpdir, "sk-client.zip")
	serverZip := path.Join(s.tmpdir, "sk-server.zip")
	apiZip := path.Join(s.tmpdir, "sk-api.zip")
	migrationsZip := path.Join(s.tmpdir, "sk-migrations.zip")

	s.NoError(os.WriteFile(clientZip, make([]byte, 150), 0664))
	s.NoError(os.WriteFile(serverZip, make([]byte, 200), 0664))
	s.NoError(os.WriteFile(apiZip, make([]byte, 250), 0664))
	s.NoError(os.WriteFile(migrationsZip, make([]byte, 100), 0664))

	result, err := client.Upload(integrations.UploadArgs{
		AppID:         types.ID(123),
		EnvID:         types.ID(456),
		DeploymentID:  types.ID(789),
		ClientZip:     clientZip,
		MigrationsZip: migrationsZip,
		ServerHandler: "index.handler",
		APIHandler:    "api.handler",
		BucketName:    "test-bucket",
	})

	s.NoError(err)
	s.NotNil(result)
	s.Equal(int64(150), result.Client.BytesUploaded)
	s.Equal(int64(100), result.Migrations.BytesUploaded)
	s.Empty(result.Server.BytesUploaded)
	s.Empty(result.API.BytesUploaded)
	s.Empty(result.API.Location)
	s.Empty(result.Server.Location)
	s.Equal(result.Client.Location, "alibaba:test-bucket/123/789/sk-client.zip")
	s.Equal(result.Migrations.Location, "alibaba:test-bucket/123/789/sk-migrations.zip")
}

func TestAlibabaClient(t *testing.T) {
	suite.Run(t, &AlibabaSuite{})
}
