package saws

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// NewLambda returns a new Lambda client.
func newLambda(sess *session.Session, cfgs ...*aws.Config) lambdaiface.LambdaAPI {
	return lambdaiface.LambdaAPI(lambda.New(sess, cfgs...))
}

// CreateFunction creates a new lambda function.
func (a *AWS) CreateFunction(input *lambda.CreateFunctionInput) (*lambda.FunctionConfiguration, error) {
	return a.Lambda.CreateFunction(input)
}

// UpdateFunction updates the lambda function.
func (a *AWS) UpdateFunction(input *lambda.UpdateFunctionConfigurationInput) (*lambda.FunctionConfiguration, error) {
	return a.Lambda.UpdateFunctionConfiguration(input)
}

// CreateFunctionAlias creates a new function alias.
func (a *AWS) CreateFunctionAlias(input *lambda.CreateAliasInput) (*lambda.AliasConfiguration, error) {
	return a.Lambda.CreateAlias(input)
}

// DeleteFunctionVersion deletes a specific function version.
func (a *AWS) DeleteFunctionVersion(fnName string, version string) (*lambda.DeleteFunctionOutput, error) {
	if strings.TrimSpace(version) == "" {
		return nil, errors.New("empty function version")
	}

	return a.Lambda.DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: &fnName,
		Qualifier:    &version,
	})
}

// UpdateFunctionAlias updates the alias and points it to the given version.
func (a *AWS) UpdateFunctionAlias(functionName, alias, version, description string) error {
	input := &lambda.UpdateAliasInput{
		Description:     aws.String(description),
		FunctionName:    aws.String(functionName),
		FunctionVersion: aws.String(version),
		Name:            aws.String(alias),
	}

	_, err := a.Lambda.UpdateAlias(input)
	return err
}

// LastFunctionVersion returns the last version of the function.
func (a *AWS) LastFunctionVersion(fn string) (*string, error) {
	input := &lambda.ListVersionsByFunctionInput{
		FunctionName: aws.String(fn),
		MaxItems:     aws.Int64(2),
	}

	output, err := a.Lambda.ListVersionsByFunction(input)

	if err != nil {
		return nil, err
	}

	if len(output.Versions) != 2 {
		return nil, nil
	}

	return output.Versions[1].Version, nil
}

// UpdateFunctionCode updates the given function code.
func (a *AWS) UpdateFunctionCode(arn string, zipFile []byte) (*lambda.FunctionConfiguration, error) {
	input := &lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(arn),
		Publish:      aws.Bool(true),
		ZipFile:      zipFile,
	}

	return a.Lambda.UpdateFunctionCode(input)
}

// Invoke invokes the given function with the given payload.
func (a *AWS) Invoke(arn, version string, payload []byte) (*lambda.InvokeOutput, error) {
	input := &lambda.InvokeInput{
		FunctionName: aws.String(arn),
		Payload:      payload,
		LogType:      aws.String(lambda.LogTypeTail),
	}

	if version != "" {
		input.Qualifier = aws.String(version)
	}

	return a.Lambda.Invoke(input)
}

// FuncTypeAPI  represents api functions, which are used as backend functions.
const FuncTypeAPI = "api"

// FuncTypeRenderer represents serverless functions that render dynamic applications and serve html.
const FuncTypeRenderer = "renderer"

type BuildFunctionARNProps struct {
	FuncType string
	EnvID    types.ID
	AppID    types.ID
}

func BuildFunctionARN(args BuildFunctionARNProps) string {
	lambdaName := utils.Int64ToString(int64(args.AppID))

	if args.EnvID != 0 {
		obfEnvId := utils.Int64ToString(int64(args.EnvID))

		// Fixes a problem that results in duplicate function names
		if !strings.Contains(lambdaName, fmt.Sprintf("-%s", obfEnvId)) {
			lambdaName = fmt.Sprintf(
				"%s-%s",
				lambdaName,
				obfEnvId,
			)
		}
	}

	if args.FuncType != "" && args.FuncType != FuncTypeRenderer {
		lambdaName = fmt.Sprintf("%s-%s", lambdaName, args.FuncType)
	}

	cnf := config.Get()

	return fmt.Sprintf(
		`arn:aws:lambda:%s:%s:function:%s`,
		cnf.AWS.Region,
		cnf.AWS.AccountID,
		lambdaName,
	)
}
