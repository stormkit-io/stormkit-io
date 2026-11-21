package integrations_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// fakeAWS records the STS, S3 and Lambda calls of a client and answers them.
// Functions exist with handler unless they are listed in missing.
type fakeAWS struct {
	mu        sync.Mutex
	handler   string
	missing   map[string]bool
	deleteErr error

	assumed []*sts.AssumeRoleInput
	puts    []string // <bucket>/<key> of each PutObject
	created []*lambda.CreateFunctionInput
	updated []*lambda.UpdateFunctionConfigurationInput
	code    []*lambda.UpdateFunctionCodeInput
	deletes []*lambda.DeleteFunctionInput
	invoked []*lambda.InvokeInput
	removed []string // <bucket>/<key> of each deleted object
}

func newFakeAWS() *fakeAWS {
	return &fakeAWS{handler: "stormkit-api.handler", missing: map[string]bool{}}
}

func (f *fakeAWS) answer(ctx context.Context, params any) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch v := params.(type) {
	case *sts.AssumeRoleInput:
		f.assumed = append(f.assumed, v)

		return &sts.AssumeRoleOutput{Credentials: &ststypes.Credentials{
			AccessKeyId:     utils.Ptr("temp-access-key"),
			SecretAccessKey: utils.Ptr("temp-secret-key"),
			SessionToken:    utils.Ptr("temp-session-token"),
		}}, nil

	case *s3.PutObjectInput:
		f.puts = append(f.puts, *v.Bucket+"/"+*v.Key)
		return &s3.PutObjectOutput{}, nil

	case *lambda.GetFunctionConfigurationInput:
		if f.missing[*v.FunctionName] {
			return nil, &lambdatypes.ResourceNotFoundException{}
		}

		return &lambda.GetFunctionConfigurationOutput{Handler: utils.Ptr(f.handler)}, nil

	case *lambda.GetFunctionInput:
		return &lambda.GetFunctionOutput{Configuration: &lambdatypes.FunctionConfiguration{State: lambdatypes.StateActive}}, nil

	case *lambda.CreateFunctionInput:
		f.created = append(f.created, v)
		delete(f.missing, *v.FunctionName)

		return &lambda.CreateFunctionOutput{Version: utils.Ptr("1")}, nil

	case *lambda.UpdateFunctionConfigurationInput:
		f.updated = append(f.updated, v)
		f.handler = *v.Handler

		return &lambda.UpdateFunctionConfigurationOutput{LastUpdateStatus: lambdatypes.LastUpdateStatusSuccessful}, nil

	case *lambda.UpdateFunctionCodeInput:
		f.code = append(f.code, v)
		return &lambda.UpdateFunctionCodeOutput{Version: utils.Ptr("5")}, nil

	case *lambda.InvokeInput:
		f.invoked = append(f.invoked, v)
		return &lambda.InvokeOutput{Payload: []byte(`{"status":200,"body":"from lambda"}`)}, nil

	case *lambda.DeleteFunctionInput:
		f.deletes = append(f.deletes, v)
		return &lambda.DeleteFunctionOutput{}, f.deleteErr

	case *s3.ListObjectsV2Input:
		return &s3.ListObjectsV2Output{
			Contents: []s3types.Object{{Key: utils.Ptr(*v.Prefix + "sk-api.zip")}},
		}, nil

	case *s3.DeleteObjectsInput:
		for _, o := range v.Delete.Objects {
			f.removed = append(f.removed, *v.Bucket+"/"+*o.Key)
		}

		return &s3.DeleteObjectsOutput{}, nil
	}

	panic("unexpected aws call: " + awsmiddleware.GetOperationName(ctx))
}

// middlewares answer every call of a client with the fake.
func (f *fakeAWS) middlewares() []func(stack *middleware.Stack) error {
	return []func(stack *middleware.Stack) error{
		func(stack *middleware.Stack) error {
			var params any

			err := stack.Initialize.Add(
				middleware.InitializeMiddlewareFunc("Params", func(ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
					params = in.Parameters
					return next.HandleInitialize(ctx, in)
				}),
				middleware.Before,
			)

			if err != nil {
				return err
			}

			return stack.Finalize.Add(
				middleware.FinalizeMiddlewareFunc("Fake", func(ctx context.Context, _ middleware.FinalizeInput, _ middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
					result, err := f.answer(ctx, params)
					return middleware.FinalizeOutput{Result: result}, middleware.Metadata{}, err
				}),
				middleware.Before,
			)
		},
	}
}

// useLambda configures Stormkit Cloud with AWS next to HCloud. The
// returned fake answers the instance's and the build's Lambda clients, shared
// answers the shared AWS client that serves older deployments.
func (s *HCloudSuite) useLambda() (fake, shared *fakeAWS) {
	config.Get().AWS = &config.AwsConfig{
		AccountID:      "123456789",
		Region:         "eu-central-1",
		StorageBucket:  "aws-bucket",
		LambdaRoleName: "lambda-role",
		RunnerRoleARN:  "arn:aws:iam::123456789:role/runner",
	}

	config.SetIsStormkitCloud(true)
	s.T().Cleanup(func() { config.SetIsStormkitCloud(false) })

	s.Require().NoError(os.Setenv("AWS_ACCESS_KEY_ID", "instance-access-key"))
	s.Require().NoError(os.Setenv("AWS_SECRET_ACCESS_KEY", "instance-secret-key"))

	s.T().Cleanup(func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	})

	fake, shared = newFakeAWS(), newFakeAWS()

	_, err := integrations.AWS(integrations.ClientArgs{
		AccessKey:   "volume-access-key",
		SecretKey:   "volume-secret-key",
		Middlewares: shared.middlewares(),
	}, nil)

	s.Require().NoError(err)
	s.T().Cleanup(integrations.SetLambdaMiddlewares(fake.middlewares()...))

	return fake, shared
}

type signFunctionUploadsParams struct {
	functions *integrations.FunctionHandlers
}

// signFunctionUploads signs the uploads of deployment 7 of app 3, env 5.
func (s *HCloudSuite) signFunctionUploads(p signFunctionUploadsParams) (*integrations.SignedUploads, error) {
	return s.client().SignUploads(context.Background(), integrations.SignUploadsParams{
		AppID:        3,
		EnvID:        5,
		DeploymentID: 7,
		Functions:    p.functions,
		EnvVars:      map[string]string{"SK_APP_ID": "3"},
	})
}

// Test_SignUploads_FunctionCredentials verifies that with functions on
// Lambda, the build gets credentials that can only publish code to its own
// functions, issued with the instance's credentials.
func (s *HCloudSuite) Test_SignUploads_FunctionCredentials() {
	fake, shared := s.useLambda()
	uploads, err := s.signFunctionUploads(signFunctionUploadsParams{})

	s.Require().NoError(err)
	s.Require().NotNil(uploads.Functions)
	s.Equal("temp-access-key", uploads.Functions.AccessKey)
	s.Equal("temp-secret-key", uploads.Functions.SecretKey)
	s.Equal("temp-session-token", uploads.Functions.SessionToken)
	s.Equal("aws-bucket", uploads.Functions.Bucket)
	s.Equal("lambda-role", uploads.Functions.LambdaRole)
	s.Contains(uploads.Artifacts, "sk-client.zip")

	s.Empty(shared.assumed)
	s.Require().Len(fake.assumed, 1)

	input := fake.assumed[0]

	s.Equal("arn:aws:iam::123456789:role/runner", *input.RoleArn)
	s.Equal("deployment-7", *input.RoleSessionName)
	s.Equal(int32(1800), *input.DurationSeconds)

	policy := struct {
		Statement []struct {
			Action   []string `json:"Action"`
			Resource []string `json:"Resource"`
		} `json:"Statement"`
	}{}

	s.Require().NoError(json.Unmarshal([]byte(*input.Policy), &policy))
	s.Require().Len(policy.Statement, 2)

	s.Equal([]string{"arn:aws:s3:::aws-bucket/3/7/*"}, policy.Statement[0].Resource)
	s.Contains(policy.Statement[0].Action, "s3:GetObject")
	s.Equal([]string{
		"arn:aws:lambda:eu-central-1:123456789:function:3-5",
		"arn:aws:lambda:eu-central-1:123456789:function:3-5-api",
	}, policy.Statement[1].Resource)
	s.ElementsMatch([]string{
		"lambda:GetFunction",
		"lambda:GetFunctionConfiguration",
		"lambda:PublishVersion",
		"lambda:UpdateFunctionCode",
	}, policy.Statement[1].Action)
}

// Test_SignUploads_CreatesMissingFunction verifies that the instance creates a
// new environment's function, so the build never needs to.
func (s *HCloudSuite) Test_SignUploads_CreatesMissingFunction() {
	fake, _ := s.useLambda()
	fake.missing["arn:aws:lambda:eu-central-1:123456789:function:3-5-api"] = true

	_, err := s.signFunctionUploads(signFunctionUploadsParams{functions: &integrations.FunctionHandlers{
		API:     "stormkit-api.mjs:handler",
		Runtime: "nodejs22.x",
	}})

	s.Require().NoError(err)
	s.Require().Len(fake.created, 1)

	created := fake.created[0]

	s.Equal("arn:aws:lambda:eu-central-1:123456789:function:3-5-api", *created.FunctionName)
	s.Equal("stormkit-api.handler", *created.Handler)
	s.Equal("lambda-role", *created.Role)
	s.Equal(lambdatypes.Runtime("nodejs22.x"), created.Runtime)
	s.Equal(map[string]string{"SK_APP_ID": "3"}, created.Environment.Variables)
	s.NotEmpty(created.Code.ZipFile)
	s.Empty(fake.updated)
}

// Test_SignUploads_SyncsHandler verifies that the instance points an existing
// function at the build's handler.
func (s *HCloudSuite) Test_SignUploads_SyncsHandler() {
	fake, _ := s.useLambda()
	fake.handler = "old.handler"

	_, err := s.signFunctionUploads(signFunctionUploadsParams{functions: &integrations.FunctionHandlers{
		Server: ".output/server/index.mjs:handler",
	}})

	s.Require().NoError(err)
	s.Empty(fake.created)
	s.Require().Len(fake.updated, 1)
	s.Equal("arn:aws:lambda:eu-central-1:123456789:function:3-5", *fake.updated[0].FunctionName)
	s.Equal(".output/server/index.handler", *fake.updated[0].Handler)
}

// Test_SignUploads_NoRunnerRole verifies that a missing runner role fails the
// build instead of leaving its functions on the hosting servers.
func (s *HCloudSuite) Test_SignUploads_NoRunnerRole() {
	s.useLambda()
	config.Get().AWS.RunnerRoleARN = ""

	_, err := s.signFunctionUploads(signFunctionUploadsParams{})

	s.ErrorContains(err, "AWS_RUNNER_ROLE_ARN")
}

// Test_SignUploads_SelfHosted verifies that self-hosted instances keep their
// functions on HCloud even with AWS and a Lambda role configured.
func (s *HCloudSuite) Test_SignUploads_SelfHosted() {
	fake, _ := s.useLambda()
	config.SetIsStormkitCloud(false)

	uploads, err := s.signFunctionUploads(signFunctionUploadsParams{functions: &integrations.FunctionHandlers{API: "stormkit-api.mjs:handler"}})

	s.Require().NoError(err)
	s.Nil(uploads.Functions)
	s.Empty(fake.assumed)
	s.Empty(fake.created)

	err = s.client().VerifyUploads(integrations.VerifyUploadsParams{
		AppID:        3,
		EnvID:        5,
		DeploymentID: 7,
		Result:       &integrations.UploadResult{API: integrations.UploadOverview{Location: lambdaAPILocation}},
	})

	s.ErrorContains(err, "is not an upload of this deployment")
}

// Test_SignUploads_NoLambda verifies that without Lambda, functions keep
// going to the bucket, as on self-hosted instances.
func (s *HCloudSuite) Test_SignUploads_NoLambda() {
	uploads, err := s.signFunctionUploads(signFunctionUploadsParams{})

	s.Require().NoError(err)
	s.Nil(uploads.Functions)
}

// Test_SignedUploads_PublishesToLambda verifies that a build with function
// credentials publishes its API to Lambda and the rest to the bucket.
func (s *HCloudSuite) Test_SignedUploads_PublishesToLambda() {
	fake, _ := s.useLambda()
	uploads, err := s.signFunctionUploads(signFunctionUploadsParams{functions: &integrations.FunctionHandlers{API: "stormkit-api.mjs:handler"}})
	s.Require().NoError(err)

	s.zip("sk-api.zip", map[string]string{"stormkit-api.mjs": "export const handler = () => {}"})

	result, err := uploads.Upload(integrations.UploadArgs{
		AppID:        3,
		EnvID:        5,
		DeploymentID: 7,
		ClientZip:    path.Join(s.tmpdir, "sk-client.zip"),
		APIZip:       path.Join(s.tmpdir, "sk-api.zip"),
		APIHandler:   "stormkit-api.mjs:handler",
	})

	s.Require().NoError(err)

	s.Equal("hcloud:artifacts/3/7/sk-client.zip", result.Client.Location)
	s.Equal("aws:arn:aws:lambda:eu-central-1:123456789:function:3-5-api/5", result.API.Location)
	s.True(s.storage.has("artifacts/3/7/sk-client.zip"))
	s.False(s.storage.has("artifacts/3/7/sk-api.zip"))

	s.Contains(fake.puts, "aws-bucket/3/7/sk-api.zip")
	s.Require().Len(fake.code, 1)
	s.Equal("3/7/sk-api.zip", *fake.code[0].S3Key)
	s.Empty(fake.updated)

	s.NoError(s.client().VerifyUploads(integrations.VerifyUploadsParams{
		AppID:        3,
		EnvID:        5,
		DeploymentID: 7,
		Result:       result,
	}))
}

// Test_Hosting_StaticFromHCloud_APIFromLambda verifies that hosting serves a
// Cloud deployment: static files from the bucket and API calls on Lambda.
func (s *HCloudSuite) Test_Hosting_StaticFromHCloud_APIFromLambda() {
	_, shared := s.useLambda()
	result := s.upload()
	client := s.client()

	file, err := client.GetFile(integrations.GetFileArgs{Location: result.Client.Location, FileName: "index.html"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Hello world", string(file.Content))

	invoked, err := client.Invoke(integrations.InvokeArgs{
		ARN:    "aws:arn:aws:lambda:eu-central-1:123456789:function:3-5-api/5",
		Method: shttp.MethodGet,
		URL:    &url.URL{Path: "/api/books"},
	})

	s.NoError(err)
	s.Require().NotNil(invoked)
	s.Equal(http.StatusOK, invoked.StatusCode)
	s.Equal("from lambda", string(invoked.Body))

	s.Require().Len(shared.invoked, 1)
	s.Equal("arn:aws:lambda:eu-central-1:123456789:function:3-5-api", *shared.invoked[0].FunctionName)
	s.Equal("5", *shared.invoked[0].Qualifier)
}

// Test_VerifyUploads_ForeignFunctions verifies that only versions of the
// deployment environment's own functions are accepted.
func (s *HCloudSuite) Test_VerifyUploads_ForeignFunctions() {
	s.useLambda()

	for _, location := range []string{
		"aws:arn:aws:lambda:eu-central-1:123456789:function:3-6-api/5",
		"aws:arn:aws:lambda:eu-central-1:123456789:function:4-5-api/5",
		"aws:arn:aws:lambda:eu-central-1:123456789:function:3-5/5",
		"aws:arn:aws:lambda:eu-central-1:123456789:function:3-5-api/",
		"aws:arn:aws:lambda:eu-central-1:123456789:function:3-5-api/5/6",
	} {
		err := s.client().VerifyUploads(integrations.VerifyUploadsParams{
			AppID:        3,
			EnvID:        5,
			DeploymentID: 7,
			Result:       &integrations.UploadResult{API: integrations.UploadOverview{Location: location}},
		})

		s.ErrorContains(err, "is not", location)
	}
}

const lambdaAPILocation = "aws:arn:aws:lambda:eu-central-1:123456789:function:3-5-api/5"

// Test_DeleteArtifacts_LambdaFunctions verifies that a deployment with static
// files on HCloud and functions on Lambda is deleted from both, with the
// instance's credentials.
func (s *HCloudSuite) Test_DeleteArtifacts_LambdaFunctions() {
	fake, shared := s.useLambda()
	result := s.upload()

	s.NoError(s.client().DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		StorageLocation: result.Client.Location,
		APILocation:     lambdaAPILocation,
		AppID:           3,
		DeploymentID:    7,
	}))

	s.Empty(shared.deletes)
	s.Require().Len(fake.deletes, 1)
	s.Equal("arn:aws:lambda:eu-central-1:123456789:function:3-5-api", *fake.deletes[0].FunctionName)
	s.Equal("5", *fake.deletes[0].Qualifier)
	s.Equal([]string{"aws-bucket/3/7/sk-api.zip"}, fake.removed)

	s.False(s.storage.has("artifacts/3/7/sk-client.zip"))
}

// Test_DeleteArtifacts_LambdaFunctionGone verifies that a version deleted
// already does not keep the deployment from being deleted.
func (s *HCloudSuite) Test_DeleteArtifacts_LambdaFunctionGone() {
	fake, _ := s.useLambda()
	fake.deleteErr = &lambdatypes.ResourceNotFoundException{}
	result := s.upload()

	s.NoError(s.client().DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		StorageLocation: result.Client.Location,
		APILocation:     lambdaAPILocation,
		AppID:           3,
		DeploymentID:    7,
	}))

	s.False(s.storage.has("artifacts/3/7/sk-client.zip"))
}

// Test_DeleteArtifacts_LambdaFails verifies that the files in the bucket are
// deleted even when deleting the function fails.
func (s *HCloudSuite) Test_DeleteArtifacts_LambdaFails() {
	fake, _ := s.useLambda()
	fake.deleteErr = errors.New("lambda is down")
	result := s.upload()

	err := s.client().DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		StorageLocation: result.Client.Location,
		APILocation:     lambdaAPILocation,
		AppID:           3,
		DeploymentID:    7,
	})

	s.ErrorContains(err, "lambda is down")
	s.False(s.storage.has("artifacts/3/7/sk-client.zip"))
	s.NoDirExists(path.Join(s.tmpdir, "deployments", "hcloud", "artifacts", "3", "7"))
}

// Test_DeleteArtifacts_FunctionsOnly verifies that a deployment without static
// files still has its bucket files and AWS copies deleted.
func (s *HCloudSuite) Test_DeleteArtifacts_FunctionsOnly() {
	fake, _ := s.useLambda()
	s.upload()

	s.NoError(s.client().DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		APILocation:  lambdaAPILocation,
		AppID:        3,
		DeploymentID: 7,
	}))

	s.Len(fake.deletes, 1)
	s.Equal([]string{"aws-bucket/3/7/sk-api.zip"}, fake.removed)
	s.False(s.storage.has("artifacts/3/7/sk-migrations.zip"))
}
