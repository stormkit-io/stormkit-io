package integrations

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	sktypes "github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

// FuncTypeAPI  represents api functions, which are used as backend functions.
// FuncTypeRenderer represents serverless functions that render dynamic applications and serve html.
const (
	FuncTypeAPI      = "api"
	FuncTypeRenderer = "renderer"
)

// AllowedRuntimes specifies the list of supported runtimes.
var AllowedRuntimes = []string{
	config.NodeRuntime20,
	config.NodeRuntime22,
}

func (a *AWSClient) Invoke(args InvokeArgs) (*InvokeResult, error) {
	fnName, fnVersion := a.parseFunctionLocation(args.ARN)
	requestPayload, err := json.Marshal(prepareInvokeRequest(args))

	if err != nil {
		return nil, err
	}

	input := &lambda.InvokeInput{
		FunctionName: &fnName,
		Payload:      requestPayload,
		LogType:      types.LogTypeTail,
	}

	if fnVersion != "" {
		input.Qualifier = &fnVersion
	}

	out, err := a.lambdaClient.Invoke(context.TODO(), input)

	if err != nil {
		return nil, err
	}

	if out == nil {
		return nil, nil
	}

	// #########
	response := FunctionResponse{}

	if err := json.Unmarshal(out.Payload, &response); err != nil {
		return nil, err
	}

	body := utils.GetString(response.Buffer, response.Body)

	invokeResult := &InvokeResult{
		Logs:         response.Logs,
		Body:         []byte(body),
		Headers:      parseHeaders(response.Headers),
		StatusCode:   utils.GetInt(response.Status, response.StatusCode, http.StatusOK),
		ErrorMessage: response.ErrorMessage,
		ErrorStack:   response.ErrorStack,
	}

	// See if this is a base64 encoded string. Must stay StdEncoding: Lambda
	// emits standard base64, so the URL-safe helpers in lib/utils would fail to
	// decode any body containing '+' or '/' — and silently, since a decode
	// error here just leaves the body as-is.
	if decoded, err := base64.StdEncoding.DecodeString(body); err == nil {
		invokeResult.Body = decoded
	}

	return invokeResult, nil
}

func (a *AWSClient) uploadToLambda(args UploadArgs) (UploadOverview, error) {
	result := UploadOverview{}

	if args.zip == "" {
		return result, nil
	}

	fstat, err := os.Stat(args.zip)

	// Nothing to deploy
	if err != nil || fstat == nil || file.IsZipEmpty(args.zip) {
		return result, nil
	}

	arn := BuildFunctionARN(BuildFunctionARNProps{
		EnvID:    args.EnvID,
		AppID:    args.AppID,
		FuncType: args.funcType,
	})

	fileContent, err := os.ReadFile(args.zip)

	if err != nil {
		return result, err
	}

	s3args := S3Args{
		BucketName: a.bucketName(args),
		KeyPrefix:  fmt.Sprintf("%d/%d", args.AppID, args.DeploymentID),
	}

	uploadFile := File{
		Size:         fstat.Size(),
		Content:      fileContent,
		RelativePath: path.Base(args.zip),
		ContentType:  DetectContentType(args.zip, fileContent),
	}

	// Always upload deployment package to s3 to take a backup.
	if err = a.UploadFile(uploadFile, s3args); err != nil {
		return result, err
	}

	version, err := a.createVersion(awsCreateFunctionArgs{
		HandlerName:  lambdaHandler(args.handler),
		FunctionName: arn,
		S3BucketName: s3args.BucketName,
		S3KeyPrefix:  path.Join(s3args.KeyPrefix, uploadFile.RelativePath),
	})

	if err != nil {
		var rnf *types.ResourceNotFoundException

		if errors.As(err, &rnf) {
			version, err = a.createFunction(awsCreateFunctionArgs{
				AppID:        args.AppID,
				FunctionName: arn,
				HandlerName:  a.normalizedAWSFunctionHandler(args.handler),
				RoleName:     config.Get().AWS.LambdaRoleName,
				Runtime:      types.Runtime(a.validRuntime(args.Runtime)),
				S3BucketName: s3args.BucketName,
				S3KeyPrefix:  path.Join(s3args.KeyPrefix, uploadFile.RelativePath),
				EnvVars:      args.EnvVars,
			})

			if err != nil {
				return result, err
			}
		} else {
			return result, err
		}
	}

	// Upload also en example of env vars to make sure everything is recoverable.
	if config.IsStormkitCloud() {
		content := []string{}

		for k, v := range args.EnvVars {
			content = append(content, fmt.Sprintf("%s=%s", k, v))
		}

		envFile := strings.Join(content, "\n")

		err = a.UploadFile(File{
			Size:         int64(len(envFile)),
			Content:      []byte(envFile),
			ContentType:  "text/plain",
			RelativePath: ".env.backup",
		}, s3args)

		// Do nothing because this is not a blocker.
		if err != nil {
			slog.Errorf("error while backing up env var: %s", err.Error())
		}
	}

	if version != nil {
		result.BytesUploaded = fstat.Size()
		result.Location = fmt.Sprintf("%s:%s/%s", a.providerPrefix, arn, *version)
	}

	return result, nil
}

// lambdaHandler turns a handler such as server/index.mjs:handler into the
// one Lambda expects, server/index.handler.
func lambdaHandler(handler string) string {
	file, exported, ok := strings.Cut(handler, ":")

	if !ok || exported == "" {
		exported = "handler"
	}

	return fmt.Sprintf("%s.%s", strings.TrimSuffix(file, filepath.Ext(file)), exported)
}

type prepareFunctionParams struct {
	arn     string
	handler string
	runtime string
	envVars map[string]string
}

// prepareFunction makes sure that the function exists and runs the given
// handler, so that publishing to it only needs to update its code. A new
// function starts with placeholder code.
func (a *AWSClient) prepareFunction(ctx context.Context, p prepareFunctionParams) error {
	args := awsCreateFunctionArgs{
		FunctionName: p.arn,
		HandlerName:  lambdaHandler(p.handler),
	}

	_, err := a.lambdaClient.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{FunctionName: &p.arn})

	var rnf *types.ResourceNotFoundException

	if !errors.As(err, &rnf) {
		if err != nil {
			return err
		}

		return a.syncHandler(args)
	}

	placeholder, err := placeholderZip()

	if err != nil {
		return err
	}

	args.RoleName = config.Get().AWS.LambdaRoleName
	args.Runtime = types.Runtime(a.validRuntime(p.runtime))
	args.EnvVars = p.envVars
	args.ZipFile = placeholder

	if _, err := a.createFunction(args); err != nil {
		return err
	}

	// Code updates are refused until the new function is active.
	return lambda.NewFunctionActiveV2Waiter(a.lambdaClient).Wait(ctx, &lambda.GetFunctionInput{FunctionName: &p.arn}, 2*time.Minute)
}

// placeholderZip returns the code of a function that has not been published
// to yet.
func placeholderZip() ([]byte, error) {
	var buf bytes.Buffer

	w := zip.NewWriter(&buf)
	f, err := w.Create("index.js")

	if err != nil {
		return nil, err
	}

	if _, err := f.Write([]byte(`exports.handler = async () => ({ statusCode: 503 });`)); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type awsCreateFunctionArgs struct {
	AppID         sktypes.ID
	FunctionName  string
	HandlerName   string
	S3BucketName  string
	S3KeyPrefix   string
	RoleName      string
	Runtime       types.Runtime
	Timeout       int32
	MemorySize    int32
	EnvVars       map[string]string
	ZipFile       []byte // Code to create the function with instead of S3
	resourceRetry int
}

func (a *AWSClient) createFunction(opts awsCreateFunctionArgs) (*string, error) {
	if opts.MemorySize == 0 {
		opts.MemorySize = 512
	}

	if opts.Timeout == 0 {
		opts.Timeout = 15
	}

	code := &types.FunctionCode{
		S3Bucket: &opts.S3BucketName,
		S3Key:    &opts.S3KeyPrefix,
	}

	if opts.ZipFile != nil {
		code = &types.FunctionCode{ZipFile: opts.ZipFile}
	}

	out, err := a.lambdaClient.CreateFunction(context.Background(), &lambda.CreateFunctionInput{
		FunctionName: &opts.FunctionName,
		Handler:      &opts.HandlerName,
		MemorySize:   &opts.MemorySize,
		Role:         &opts.RoleName,
		Runtime:      opts.Runtime,
		Timeout:      &opts.Timeout,
		Publish:      true,
		Code:         code,
		Environment: &types.Environment{
			Variables: opts.EnvVars,
		},
	})

	if err != nil || out == nil || out.Version == nil {
		return nil, err
	}

	return out.Version, nil
}

// validRuntime checks if the provided runtime is supported.
// If not, it returns the default runtime.
func (a *AWSClient) validRuntime(runtime string) string {
	if runtime == "" {
		return config.DefaultNodeRuntime
	}

	for _, r := range AllowedRuntimes {
		if r == runtime {
			return runtime
		}
	}

	return config.DefaultNodeRuntime
}

func (a *AWSClient) createVersion(opts awsCreateFunctionArgs) (*string, error) {
	if err := a.syncHandler(opts); err != nil {
		return nil, err
	}

	out, err := a.lambdaClient.UpdateFunctionCode(context.Background(), &lambda.UpdateFunctionCodeInput{
		S3Bucket:     &opts.S3BucketName,
		S3Key:        &opts.S3KeyPrefix,
		FunctionName: &opts.FunctionName,
		Publish:      true,
	})

	if err != nil {
		var rce *types.ResourceConflictException

		// retry 5 times if there is a resource conflict exception
		// it usually occurs when the function is still in pending state
		// and we're trying to access it
		if errors.As(err, &rce) && opts.resourceRetry < 5 {
			opts.resourceRetry = opts.resourceRetry + 1
			return a.createVersion(opts)
		}
	}

	if err != nil || out == nil || out.Version == nil {
		return nil, err
	}

	return out.Version, nil
}

// syncHandler makes sure that the handler name of the function is equal
// to build manifests handler name.
func (a *AWSClient) syncHandler(args awsCreateFunctionArgs) error {
	if args.FunctionName == "" {
		return nil
	}

	ctx := context.Background()
	out, err := a.lambdaClient.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
		FunctionName: &args.FunctionName,
	})

	if err != nil {
		return err
	}

	// Wait until function is ready in case it's being updated by another deployment.
	if out != nil {
		if err := a.waitFunctionConfig(args.FunctionName, out.LastUpdateStatus); err != nil {
			return err
		}
	}

	// Make sure handler name matches manifest
	if out.Handler == nil || *out.Handler != args.HandlerName {
		out, err := a.lambdaClient.UpdateFunctionConfiguration(ctx, &lambda.UpdateFunctionConfigurationInput{
			FunctionName: &args.FunctionName,
			Handler:      &args.HandlerName,
		})

		if err != nil {
			return err
		}

		if out != nil {
			if err := a.waitFunctionConfig(args.FunctionName, out.LastUpdateStatus); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *AWSClient) waitFunctionConfig(functionName string, lastUpdateStatus types.LastUpdateStatus) error {
	// Wait until function is ready in case it's being updated by another deployment.
	if lastUpdateStatus == types.LastUpdateStatusInProgress {
		waiter := lambda.NewFunctionUpdatedV2Waiter(a.lambdaClient)
		_, err := waiter.WaitForOutput(context.Background(), &lambda.GetFunctionInput{
			FunctionName: &functionName,
		}, 5*time.Minute)
		return err
	}

	return nil
}

// normalizeAWSFunctionHandler modifies the function handler string.
// It reads a function handler in the following format `index.js:handler`
// and transforms it into `index.handler`.
func (a *AWSClient) normalizedAWSFunctionHandler(handler string) string {
	defaultHandler := "handler"
	pieces := strings.Split(handler, ".")
	fileName := pieces[0]

	if len(pieces) == 1 {
		return fmt.Sprintf("%s.%s", fileName, defaultHandler)
	}

	pieces = strings.Split(pieces[1], ":")

	if len(pieces) == 1 {
		return fmt.Sprintf("%s.%s", fileName, defaultHandler)
	}

	return fmt.Sprintf("%s.%s", fileName, pieces[1])
}

func (a *AWSClient) parseFunctionLocation(location string) (string, string) {
	pieces := strings.Split(location[len(a.providerPrefix)+1:], "/")

	if len(pieces) < 2 {
		slog.Errorf("invalid function location format: %s", location)
		return "", ""
	}

	arn := pieces[0]
	version := strings.Join(pieces[1:], "/")

	return arn, version
}
