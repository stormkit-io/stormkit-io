//go:build alibaba

package integrations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/alibabacloud-go/fc-20230330/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AlibabaFunctionsSuite struct {
	suite.Suite

	sdk    *mocks.AlibabaSDK
	tmpdir string
}

func (s *AlibabaFunctionsSuite) SetupSuite() {
	setAlibabaEnvVars()
}

func (s *AlibabaFunctionsSuite) BeforeTest(_, _ string) {
	s.sdk = &mocks.AlibabaSDK{}
	integrations.DefaultAlibabaSDK = s.sdk

	tmpDir, err := os.MkdirTemp("", "tmp-integrations-alibaba-")

	s.NoError(err)

	s.tmpdir = tmpDir
	clientDir := path.Join(tmpDir, "client")
	serverDir := path.Join(tmpDir, "server")

	s.NoError(os.MkdirAll(clientDir, 0775))
	s.NoError(os.MkdirAll(serverDir, 0775))
	s.NoError(os.WriteFile(path.Join(clientDir, "index.js"), []byte("export const handler = () => {}"), 0664))
	s.NoError(os.WriteFile(path.Join(serverDir, "index.js"), []byte("export const handler = () => {}"), 0664))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{serverDir}, ZipName: path.Join(tmpDir, "sk-server.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-api.zip")}))
}

func (s *AlibabaFunctionsSuite) AfterTest(_, _ string) {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}

	integrations.DefaultAlibabaSDK = nil
	integrations.CachedAlibabaClient = nil
	integrations.CachedAWSClient = nil
}

func (s *AlibabaFunctionsSuite) Test_Invoke() {
	alibaba, err := integrations.Alibaba(integrations.ClientArgs{})
	s.NoError(err)

	url, err := url.Parse("https://www.example.org/my-function/path?p=1&a=2&a=3#my-hash")
	s.NoError(err)

	expected := new(bytes.Buffer)
	s.NoError(json.Compact(expected, []byte(`{
		"method": "POST",
		"url": "https://www.example.org/my-function/path?p=1\u0026a=2\u0026a=3#my-hash",
		"path": "/my-function/path",
		"body": "Hello World",
		"query": {
			"a": ["2","3"],
			"p": ["1"]
		},
		"headers": {
			"host":"www.example.org"
		}
	}`)))

	s.sdk.On("InvokeFunction", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
		fnName := args.Get(0).(*string)
		request := args.Get(1).(*client.InvokeFunctionRequest)

		body, err := io.ReadAll(request.Body)
		s.NoError(err)
		s.Equal("sk-1-1-api", *fnName)
		s.Equal("5", *request.Qualifier)
		s.Equal(expected.String(), string(body))
	}).Return(&client.InvokeFunctionResponse{
		Body: strings.NewReader(`{"body":"Hello World!", "status": 207, "headers": { "x-custom-header": "true" }}`),
	}, nil)

	response, err := alibaba.Invoke(integrations.InvokeArgs{
		ARN:      "alibaba:acs:fc:me-central-1:3907502398410900:functions/sk-1-1-api/5",
		URL:      url,
		Body:     io.NopCloser(strings.NewReader("Hello World")),
		Method:   http.MethodPost,
		HostName: "www.example.org",
	})

	s.NoError(err)
	s.NotNil(response)
	s.Equal("Hello World!", string(response.Body))
	s.Equal(http.StatusMultiStatus, response.StatusCode)
	s.Empty(response.ErrorMessage)
	s.Empty(response.ErrorStack)
}

func (s *AlibabaFunctionsSuite) Test_Upload_Renderer() {
	alibaba, err := integrations.Alibaba(integrations.ClientArgs{
		AccessKey: "my-access-key",
		SecretKey: "my-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.PutObjectInput:
							s.Equal("stormkit-test-local", *v.Bucket)
							s.Equal("1/10/sk-server.zip", *v.Key)
							s.Equal(s3types.ServerSideEncryptionAes256, v.ServerSideEncryption)
							s.Greater(*v.ContentLength, int64(167))
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

	s.sdk.On("GetFunction", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
		fnName := args.Get(0).(*string)
		request := args.Get(1).(*client.GetFunctionRequest)

		s.Equal("sk-1-2-local", *fnName)
		s.NotNil(request)
	}).Return(&client.GetFunctionResponse{
		Headers: nil,
	}, nil)

	s.sdk.On("CreateFunction", mock.Anything).Once().Run(func(args mock.Arguments) {
		req := args.Get(0).(*client.CreateFunctionRequest)
		s.Equal("sk-1-2-local", *req.Body.FunctionName)
		s.Equal("index.handler", *req.Body.Handler)
		s.Equal("production", *req.Body.EnvironmentVariables["NODE_ENV"])
	}).Return(&client.CreateFunctionResponse{
		Body: &client.Function{
			FunctionArn: tea.String("my-function-arn/sk-1-2"),
		},
	}, nil)

	s.sdk.On("PublishFunctionVersion", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
		fnName := args.Get(0).(*string)
		req := args.Get(1).(*client.PublishFunctionVersionRequest)
		s.Equal("sk-1-2-local", *fnName)
		s.Equal("Deployment ID: 10", *req.Body.Description)
	}).Return(&client.PublishFunctionVersionResponse{
		Body: &client.Version{
			VersionId: tea.String("my-version-id"),
		},
	}, nil)

	result, err := alibaba.Upload(integrations.UploadArgs{
		AppID:         1,
		EnvID:         2,
		DeploymentID:  10,
		BucketName:    "stormkit-test-local",
		ServerZip:     path.Join(s.tmpdir, "sk-server.zip"),
		ServerHandler: "index.js:handler",
		EnvVars: map[string]string{
			"NODE_ENV": "production",
		},
	})

	s.NoError(err)
	s.NotNil(result)
}

func (s *AlibabaFunctionsSuite) Test_Upload_API() {
	alibaba, err := integrations.Alibaba(integrations.ClientArgs{
		AccessKey: "my-access-key",
		SecretKey: "my-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Initialize.Add(
					middleware.InitializeMiddlewareFunc("Upload", func(ctx context.Context, fi middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
						switch v := fi.Parameters.(type) {
						case *s3.PutObjectInput:
							s.Equal("stormkit-test-local", *v.Bucket)
							s.Equal("1/10/sk-api.zip", *v.Key)
							s.Equal(s3types.ServerSideEncryptionAes256, v.ServerSideEncryption)
							s.Greater(*v.ContentLength, int64(100))
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

	s.sdk.On("GetFunction", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
		fnName := args.Get(0).(*string)
		request := args.Get(1)

		s.Equal("sk-1-2-local", *fnName)
		s.NotNil(request)
	}).Return(&client.GetFunctionResponse{
		Headers: nil,
	}, nil)

	s.sdk.On("CreateFunction", mock.Anything).Once().Run(func(args mock.Arguments) {
		req := args.Get(0).(*client.CreateFunctionRequest)
		s.Equal("sk-1-2-local", *req.Body.FunctionName)
		s.Equal("stormkit-api.handler", *req.Body.Handler)
		s.Equal("production", *req.Body.EnvironmentVariables["NODE_ENV"])
	}).Return(&client.CreateFunctionResponse{
		Body: &client.Function{
			FunctionArn: tea.String("my-function-arn/sk-1-2"),
		},
	}, nil)

	s.sdk.On("PublishFunctionVersion", mock.Anything, mock.Anything).Once().Run(func(args mock.Arguments) {
		fnName := args.Get(0).(*string)
		req := args.Get(1).(*client.PublishFunctionVersionRequest)
		s.Equal("sk-1-2-local", *fnName)
		s.Equal("Deployment ID: 10", *req.Body.Description)
	}).Return(&client.PublishFunctionVersionResponse{
		Body: &client.Version{
			VersionId: tea.String("my-version-id"),
		},
	}, nil)

	result, err := alibaba.Upload(integrations.UploadArgs{
		AppID:         1,
		EnvID:         2,
		DeploymentID:  10,
		ServerZip:     path.Join(s.tmpdir, "sk-api.zip"),
		BucketName:    "stormkit-test-local",
		ServerHandler: "stormkit-api.mjs:handler",
		EnvVars: map[string]string{
			"NODE_ENV": "production",
		},
	})

	s.NoError(err)
	s.NotNil(result)
}

func TestAlibabaFunctionsSuite(t *testing.T) {
	suite.Run(t, &AlibabaFunctionsSuite{})
}
