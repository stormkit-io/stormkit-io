package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// FunctionCredentialsTTL is how long the credentials that publish a
// deployment's functions stay valid. They are issued right before the upload,
// and publishing waits up to five minutes for Lambda to finish an update.
const FunctionCredentialsTTL = 30 * 60 // seconds

// FunctionCredentials are short-lived AWS credentials that can only publish
// new code to the functions of one deployment's environment. A build gets
// them together with its upload links, after its commands ran.
type FunctionCredentials struct {
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken"`
	Region       string `json:"region"`
	AccountID    string `json:"accountId"`
	Bucket       string `json:"bucket"`
	LambdaRole   string `json:"lambdaRole"`
}

// FunctionHandlers name the functions a build is about to publish, such as
// server.mjs:handler, and the Lambda runtime they need.
type FunctionHandlers struct {
	Server  string `json:"server,omitempty"`
	API     string `json:"api,omitempty"`
	Runtime string `json:"runtime,omitempty"`
}

// publishesToLambda reports whether functions run on Lambda while the rest of
// a deployment is stored on HCloud. Only Stormkit Cloud does this; on other
// instances functions run from HCloud on the hosting servers, whatever their
// AWS settings.
func (c *HCloudClient) publishesToLambda() bool {
	conf := config.Get().AWS

	return config.IsStormkitCloud() && conf != nil && conf.LambdaRoleName != ""
}

// instanceAWS returns an AWS client with the instance's own credentials. The
// shared AWS client may have been created with a volume's keys instead.
func (c *HCloudClient) instanceAWS() (*AWSClient, error) {
	c.instanceMux.Lock()
	defer c.instanceMux.Unlock()

	if c.instance != nil {
		return c.instance, nil
	}

	client, err := newAWSClient(ClientArgs{Middlewares: lambdaMiddlewares}, nil)

	if err != nil {
		return nil, err
	}

	c.instance = client

	return client, nil
}

// functionCredentials prepares the functions the build is about to publish
// and assumes the runner role with a policy that only lets the credentials
// publish new code to them.
func (c *HCloudClient) functionCredentials(ctx context.Context, p SignUploadsParams) (*FunctionCredentials, error) {
	conf := config.Get().AWS

	if conf.RunnerRoleARN == "" {
		return nil, errors.New("publishing functions to lambda needs AWS_RUNNER_ROLE_ARN")
	}

	client, err := c.instanceAWS()

	if err != nil {
		return nil, err
	}

	if err := c.prepareFunctions(ctx, prepareFunctionsParams{client: client, deploy: p}); err != nil {
		return nil, err
	}

	policy, err := (functionPolicy{appID: p.AppID, envID: p.EnvID, deploymentID: p.DeploymentID}).json()

	if err != nil {
		return nil, err
	}

	out, err := client.stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         utils.Ptr(conf.RunnerRoleARN),
		RoleSessionName: utils.Ptr(fmt.Sprintf("deployment-%d", p.DeploymentID)),
		Policy:          utils.Ptr(policy),
		DurationSeconds: utils.Ptr(int32(FunctionCredentialsTTL)),
	})

	if err != nil {
		return nil, fmt.Errorf("cannot issue function credentials: %w", err)
	}

	if out.Credentials == nil {
		return nil, errors.New("cannot issue function credentials: no credentials returned")
	}

	return &FunctionCredentials{
		AccessKey:    aws.ToString(out.Credentials.AccessKeyId),
		SecretKey:    aws.ToString(out.Credentials.SecretAccessKey),
		SessionToken: aws.ToString(out.Credentials.SessionToken),
		Region:       conf.Region,
		AccountID:    conf.AccountID,
		Bucket:       conf.StorageBucket,
		LambdaRole:   conf.LambdaRoleName,
	}, nil
}

type prepareFunctionsParams struct {
	client *AWSClient
	deploy SignUploadsParams
}

// prepareFunctions creates the functions the build is about to publish and
// sets their handlers, so that the build's credentials never need to change
// a function's configuration.
func (c *HCloudClient) prepareFunctions(ctx context.Context, p prepareFunctionsParams) error {
	handlers := p.deploy.Functions

	if handlers == nil {
		return nil
	}

	arns := (functionPolicy{appID: p.deploy.AppID, envID: p.deploy.EnvID}).functionARNs()

	functions := []struct {
		arn     string
		handler string
	}{
		{arn: arns[0], handler: handlers.Server},
		{arn: arns[1], handler: handlers.API},
	}

	for _, fn := range functions {
		if fn.handler == "" {
			continue
		}

		err := p.client.prepareFunction(ctx, prepareFunctionParams{
			arn:     fn.arn,
			handler: fn.handler,
			runtime: handlers.Runtime,
			envVars: p.deploy.EnvVars,
		})

		if err != nil {
			return fmt.Errorf("cannot prepare %s: %w", fn.arn, err)
		}
	}

	return nil
}

type policyStatement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource []string `json:"Resource"`
}

// functionPolicy is the session policy of a deployment's function
// credentials. It narrows the runner role down to publishing new code to the
// environment's two functions from the deployment's folder in the bucket.
type functionPolicy struct {
	appID        types.ID
	envID        types.ID
	deploymentID types.ID
}

func (f functionPolicy) json() (string, error) {
	conf := config.Get().AWS

	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []policyStatement{
			{
				Effect: "Allow",
				// Lambda reads the code with the caller's permissions.
				Action:   []string{"s3:PutObject", "s3:AbortMultipartUpload", "s3:GetObject", "s3:GetObjectVersion"},
				Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/%s*", conf.StorageBucket, artifactKey(f.appID, f.deploymentID, ""))},
			},
			{
				Effect: "Allow",
				Action: []string{
					"lambda:GetFunction",
					"lambda:GetFunctionConfiguration",
					"lambda:PublishVersion",
					"lambda:UpdateFunctionCode",
				},
				Resource: f.functionARNs(),
			},
		},
	}

	data, err := json.Marshal(policy)

	return string(data), err
}

// functionARNs returns the ARNs of the environment's server and API functions.
func (f functionPolicy) functionARNs() []string {
	return []string{
		BuildFunctionARN(BuildFunctionARNProps{AppID: f.appID, EnvID: f.envID, FuncType: FuncTypeRenderer}),
		BuildFunctionARN(BuildFunctionARNProps{AppID: f.appID, EnvID: f.envID, FuncType: FuncTypeAPI}),
	}
}

// verifyFunctions returns an error when a function location in the result is
// not a version of the deployment environment's own functions on Lambda.
// Other locations are left to the bucket's verification and removed from the
// returned copy.
func (c *HCloudClient) verifyFunctions(p VerifyUploadsParams) (VerifyUploadsParams, error) {
	if p.Result == nil || !c.publishesToLambda() {
		return p, nil
	}

	result := *p.Result
	arns := (functionPolicy{appID: p.AppID, envID: p.EnvID}).functionARNs()

	functions := []struct {
		overview *UploadOverview
		arn      string
	}{
		{overview: &result.Server, arn: arns[0]},
		{overview: &result.API, arn: arns[1]},
	}

	for _, fn := range functions {
		if !strings.HasPrefix(fn.overview.Location, "aws:") {
			continue
		}

		version, ok := strings.CutPrefix(fn.overview.Location, "aws:"+fn.arn+"/")

		if !ok || version == "" || strings.Contains(version, "/") {
			return p, fmt.Errorf("%s is not a function of this deployment", fn.overview.Location)
		}

		fn.overview.Location = ""
	}

	p.Result = &result

	return p, nil
}

// deleteLambdaFunctions deletes the functions of a deployment published to
// Lambda, together with the copies of their zips in the AWS bucket, and
// clears their locations from args so that only the other ones are left.
func (c *HCloudClient) deleteLambdaFunctions(ctx context.Context, args *DeleteArtifactsArgs) error {
	functions := DeleteArtifactsArgs{}

	if strings.HasPrefix(args.FunctionLocation, "aws:") {
		functions.FunctionLocation = args.FunctionLocation
		args.FunctionLocation = ""
	}

	if strings.HasPrefix(args.APILocation, "aws:") {
		functions.APILocation = args.APILocation
		args.APILocation = ""
	}

	if functions.FunctionLocation == "" && functions.APILocation == "" {
		return nil
	}

	if config.Get().AWS == nil {
		return errors.New("the functions are on lambda, but aws is not configured")
	}

	client, err := c.instanceAWS()

	if err != nil {
		return err
	}

	// The zips are stored under <app-id>/<deployment-id>, the same prefix
	// as in the HCloud bucket.
	if bucket := config.Get().AWS.StorageBucket; bucket != "" && args.AppID != 0 && args.DeploymentID != 0 {
		functions.StorageLocation = "aws:" + path.Join(bucket, args.AppID.String(), args.DeploymentID.String())
	}

	return client.DeleteArtifacts(ctx, functions)
}
