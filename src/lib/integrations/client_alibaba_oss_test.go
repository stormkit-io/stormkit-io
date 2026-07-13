//go:build alibaba

package integrations_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stretchr/testify/suite"
)

type AlibabaOSSSuite struct {
	suite.Suite
	*factory.Factory

	conn   databasetest.TestDB
	tmpdir string
}

func setAlibabaEnvVars() {
	config.Get().Alibaba = &config.AlibabaConfig{
		Region: "me-central-1",
	}
}

func (s *AlibabaOSSSuite) SetupSuite() {
	setAlibabaEnvVars()
}

func (s *AlibabaOSSSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	integrations.CachedAlibabaClient = nil
	integrations.CachedAWSClient = nil
	integrations.DefaultAlibabaSDK = nil

	tmpDir, err := os.MkdirTemp("", "tmp-integrations-aws-")

	s.NoError(err)

	s.tmpdir = tmpDir
	clientDir := path.Join(tmpDir, "client")

	s.NoError(os.MkdirAll(clientDir, 0774))
	s.NoError(os.WriteFile(path.Join(clientDir, "index.html"), []byte("Hello world"), 0664))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-client.zip")}))
}

func (s *AlibabaOSSSuite) AfterTest(_, _ string) {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}

	s.conn.CloseTx()
}

func (s *AlibabaOSSSuite) TearDownSuite() {
	config.Get().Alibaba = nil

	integrations.CachedAlibabaClient = nil
	integrations.CachedAWSClient = nil
	integrations.DefaultAlibabaSDK = nil
}

func (s *AlibabaOSSSuite) Test_Upload() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	stat, err := os.Stat(path.Join(s.tmpdir, "sk-client.zip"))
	s.NoError(err)

	oss, err := integrations.Alibaba(integrations.ClientArgs{
		AccessKey: "my-access-key",
		SecretKey: "my-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.PutObjectInput:
							s.Equal("my-s3-bucket", *v.Bucket)
							s.Equal(s3types.ServerSideEncryptionAes256, v.ServerSideEncryption)
							s.Equal(stat.Size(), *v.ContentLength)
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
	s.NotNil(oss)

	result, err := oss.Upload(integrations.UploadArgs{
		AppID:        app.ID,
		EnvID:        env.ID,
		DeploymentID: 50919,
		ClientZip:    path.Join(s.tmpdir, "sk-client.zip"),
		BucketName:   "my-s3-bucket",
	})

	s.NoError(err)
	s.Empty(result.API.BytesUploaded)
	s.Empty(result.Server.BytesUploaded)
	s.Equal(stat.Size(), result.Client.BytesUploaded)
	s.Equal("alibaba:my-s3-bucket/1/50919/sk-client.zip", result.Client.Location)
	s.Equal("", result.Server.Location)
	s.Equal("", result.API.Location)
}

func (s *AlibabaOSSSuite) Test_GetFile() {
	oss, err := integrations.Alibaba(integrations.ClientArgs{
		AccessKey: "my-access-key",
		SecretKey: "my-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("GetObject", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.GetObjectInput:
							s.Equal("my-s3-bucket", *v.Bucket)
							s.Equal("client/index.html", *v.Key)
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
					middleware.FinalizeMiddlewareFunc("GetObject", func(ctx context.Context, fi middleware.FinalizeInput, fh middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)

						if opName == "GetObject" {
							return middleware.FinalizeOutput{
								Result: &s3.GetObjectOutput{
									Body:          io.NopCloser(bytes.NewReader([]byte("Hello world"))),
									ContentType:   utils.Ptr("text/html; charset=utf-8"),
									ContentLength: utils.Ptr(int64(len("Hello world"))),
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
	})

	s.NoError(err)
	s.NotNil(oss)

	result, err := oss.GetFile(integrations.GetFileArgs{
		Location: "alibaba:my-s3-bucket/client/index.html",
	})

	s.NoError(err)
	s.Equal("Hello world", string(result.Content))
	s.Equal(int64(len("Hello world")), result.Size)
	s.Equal("text/html; charset=utf-8", result.ContentType)
}

// Test_DeleteArtifacts_ContentMD5Header verifies that DeleteObjects requests sent
// to Alibaba OSS include the Content-Md5 header required by the OSS API.
func (s *AlibabaOSSSuite) Test_DeleteArtifacts_ContentMD5Header() {
	contentMD5Seen := ""

	oss, err := integrations.Alibaba(integrations.ClientArgs{
		AccessKey: "my-access-key",
		SecretKey: "my-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("AssertContentMD5", func(ctx context.Context, fi middleware.FinalizeInput, fh middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						opName := awsmiddleware.GetOperationName(ctx)

						switch opName {
						case "ListObjectsV2":
							return middleware.FinalizeOutput{
								Result: &s3.ListObjectsV2Output{
									Contents:    []s3types.Object{{Key: utils.Ptr("1/50919/index.html")}},
									IsTruncated: utils.Ptr(false),
								},
							}, middleware.Metadata{}, nil
						case "DeleteObjects":
							if req, ok := fi.Request.(*smithyhttp.Request); ok {
								contentMD5Seen = req.Header.Get("Content-Md5")
							}
							return middleware.FinalizeOutput{
								Result: &s3.DeleteObjectsOutput{},
							}, middleware.Metadata{}, nil
						default:
							return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("unexpected AWS operation: " + opName)
						}
					}),
					middleware.Before,
				)
			},
		},
	})

	s.Require().NoError(err)

	err = oss.DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		StorageLocation: "alibaba:my-s3-bucket/1/50919",
	})

	s.NoError(err)
	s.NotEmpty(contentMD5Seen, "Content-Md5 header must be present on DeleteObjects requests to Alibaba OSS")
}

func TestAlibabaOSS(t *testing.T) {
	suite.Run(t, &AlibabaOSSSuite{})
}
