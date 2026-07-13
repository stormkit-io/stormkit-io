//go:build alibaba

package integrations

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/alibabacloud-go/fc-20230330/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

type UpsertFunctionArgs struct {
	FunctionName string
	HandlerName  string
	BucketName   string // Bucket name that contains the server.zip
	ObjectName   string // The full path to server.zip, will be used as key prefix in the bucket.
	Runtime      string
	EnvVars      map[string]*string
}

// Invoke creates a function invocation request and returns the response.
func (a AlibabaClient) Invoke(args InvokeArgs) (*InvokeResult, error) {
	fnName, fnVersion := a.parseFunctionLocation(args.ARN)
	requestPayload, err := json.Marshal(prepareInvokeRequest(args))

	if err != nil {
		return nil, err
	}

	result, err := a.client.InvokeFunction(&fnName, &client.InvokeFunctionRequest{
		Body:      bytes.NewReader(requestPayload),
		Qualifier: &fnVersion,
	})

	if err != nil {
		slog.Errorf("error while invoking function=%s, err=%v", fnName, err)
		return nil, err
	}

	if result == nil {
		slog.Error("invoke result is empty")
		return nil, nil
	}

	headers := map[string]string{}

	for k, v := range result.Headers {
		headers[k] = *v
	}

	payload, err := io.ReadAll(result.Body)

	if err != nil {
		return nil, err
	}

	response := FunctionResponse{}

	if err := json.Unmarshal(payload, &response); err != nil {
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

	// See if this is a base64 encoded string
	if decoded, err := base64.StdEncoding.DecodeString(body); err == nil {
		invokeResult.Body = decoded
	}

	return invokeResult, nil
}

func (a AlibabaClient) parseFunctionLocation(location string) (string, string) {
	pieces := strings.Split(location, "functions/")

	if len(pieces) == 1 {
		return "", ""
	}

	pieces = strings.Split(pieces[1], "/")

	if len(pieces) != 2 {
		return "", ""
	}

	// name, version
	return pieces[0], pieces[1]
}

// Read https://www.alibabacloud.com/help/en/fc/developer-reference/api-fc-open-2021-04-06-createfunction
func (a AlibabaClient) uploadToFunctions(args UploadArgs) (UploadOverview, error) {
	overview := UploadOverview{}

	if args.zip == "" {
		return overview, nil
	}

	fstat, err := os.Stat(args.zip)

	// Nothing to deploy
	if err != nil || fstat == nil || file.IsZipEmpty(args.zip) {
		return overview, nil
	}

	fileContent, err := os.ReadFile(args.zip)

	if err != nil {
		return overview, err
	}

	s3args := S3Args{
		BucketName: args.BucketName,
		KeyPrefix:  fmt.Sprintf("%d/%d", args.AppID, args.DeploymentID),
	}

	uploadFile := File{
		Size:         fstat.Size(),
		Content:      fileContent,
		RelativePath: path.Base(args.zip),
		ContentType:  DetectContentType(args.zip, fileContent),
	}

	if err := a.awsClient.UploadFile(uploadFile, s3args); err != nil {
		return overview, err
	}

	// Alibaba requires function names to start with a letter.
	fnName := fmt.Sprintf("sk-%s-%s", args.AppID.String(), args.EnvID.String())

	if args.funcType == FuncTypeAPI {
		fnName = fmt.Sprintf("%s-api", fnName)
	}

	// In case we're testing locally, add a prefix to differentiate
	if config.IsDevelopment() {
		fnName = fmt.Sprintf("%s-local", fnName)
	}

	fnArgs := UpsertFunctionArgs{
		FunctionName: fnName,
		HandlerName:  a.normalizeHandlerName(args.handler),
		Runtime:      a.normalizeRuntime(args.Runtime),
		BucketName:   args.BucketName,
		ObjectName:   path.Join(s3args.KeyPrefix, uploadFile.RelativePath),
		EnvVars:      map[string]*string{},
	}

	for k, v := range args.EnvVars {
		fnArgs.EnvVars[k] = tea.String(v)
	}

	functionArn := ""

	if result, err := a.createFunctionIfNotExists(fnArgs); err != nil || result != nil {
		if result != nil && result.Body != nil && result.Body.FunctionArn != nil {
			functionArn = *result.Body.FunctionArn
		}
	}

	if functionArn == "" {
		result, err := a.client.UpdateFunction(&fnArgs.FunctionName, &client.UpdateFunctionRequest{
			Body: &client.UpdateFunctionInput{
				Code: &client.InputCodeLocation{
					OssBucketName: &fnArgs.BucketName,
					OssObjectName: &fnArgs.ObjectName,
				},
				Handler:              &fnArgs.HandlerName,
				EnvironmentVariables: fnArgs.EnvVars,
			},
		})

		if err != nil {
			return overview, err
		}

		if result != nil && result.Body != nil && result.Body.FunctionArn != nil {
			functionArn = *result.Body.FunctionArn
		}
	}

	if functionArn != "" {
		publish, err := a.client.PublishFunctionVersion(&fnArgs.FunctionName, &client.PublishFunctionVersionRequest{
			Body: &client.PublishVersionInput{
				Description: tea.String(fmt.Sprintf("Deployment ID: %s", types.ID(args.DeploymentID).String())),
			},
		})

		if err != nil {
			return overview, err
		}

		if publish == nil || publish.Body == nil || publish.Body.VersionId == nil {
			return overview, fmt.Errorf("cannot publish function: %v", fnArgs.FunctionName)
		}

		overview.BytesUploaded = fstat.Size()
		overview.Location = fmt.Sprintf("alibaba:%s/%s", functionArn, *publish.Body.VersionId)
	}

	return overview, nil
}

func (a AlibabaClient) createFunctionIfNotExists(args UpsertFunctionArgs) (*client.CreateFunctionResponse, error) {
	fn, err := a.client.GetFunction(&args.FunctionName, &client.GetFunctionRequest{})

	if err != nil {
		if e, _ := err.(*tea.SDKError); e == nil || *e.StatusCode != http.StatusNotFound {
			slog.Errorf("error while getting function=%v, args=%v, fn=%v", err.Error(), args, fn)
			return nil, err
		}
	}

	if fn == nil || fn.Body == nil {
		result, err := a.client.CreateFunction(&client.CreateFunctionRequest{
			Body: &client.CreateFunctionInput{
				Code: &client.InputCodeLocation{
					OssBucketName: &args.BucketName,
					OssObjectName: &args.ObjectName,
				},
				FunctionName:         &args.FunctionName,
				Handler:              &args.HandlerName, // e.g. index.handler
				MemorySize:           tea.Int32(512),
				Runtime:              &args.Runtime,
				Timeout:              tea.Int32(60), // seconds
				InstanceConcurrency:  tea.Int32(10),
				EnvironmentVariables: args.EnvVars,
			},
		})

		if err != nil {
			slog.Errorf("error while creating function=%v, args=%v, fn=%v", err.Error(), args, fn)
		}

		return result, err
	}

	return nil, nil
}

// Normalize the handler name according to Alibaba specifications.
// index.js:handler => index.handler
func (a AlibabaClient) normalizeHandlerName(s string) string {
	pieces := strings.Split(s, ":")
	handlerFile := strings.TrimSuffix(pieces[0], filepath.Ext(pieces[0]))
	handlerExported := "handler"

	if len(pieces) > 1 {
		handlerExported = pieces[1]
	}

	return fmt.Sprintf("%s.%s", handlerFile, handlerExported)
}

// Normalize the runtime according to Alibaba specifications.
// Note that the Node.js version is 16, therefore anything above
// version 18 will be downgraded automatically to version 16.
// nodejs16.x => nodejs16
func (a AlibabaClient) normalizeRuntime(runtime string) string {
	if !strings.HasPrefix(runtime, "nodejs") {
		return "nodejs16"
	}

	// Remove .x part from the string: nodejs18.x => nodejs18
	runtime = strings.Replace(runtime, ".x", "", 1)
	version := strings.Replace(runtime, "nodejs", "", 1)

	if utils.StringToInt(version) > 16 {
		return "nodejs16"
	}

	return runtime
}
