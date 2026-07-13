package runner

import (
	"errors"
	"os"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var DefaultUploader RunnerUploaderInterface
var Stat = os.Stat

type RunnerUploaderInterface interface {
	Upload(UploadArgs) (*integrations.UploadResult, error)
}

type Uploader struct {
	conf *config.RunnerConfig
}

type UploadArgs struct {
	MigrationsZip string
	ClientZip     string
	ServerZip     string
	ServerHandler string
	ApiZip        string
	ApiHandler    string
	EnvVars       map[string]string
	EnvID         types.ID
	AppID         types.ID
	DeploymentID  types.ID
	BucketName    string
	Region        string
	Runtime       string
}

func NewUploader(opts *config.RunnerConfig) RunnerUploaderInterface {
	if DefaultUploader != nil {
		return DefaultUploader
	}

	if opts == nil {
		opts = &config.RunnerConfig{}
	}

	return &Uploader{
		conf: opts,
	}
}

func (u *Uploader) isDeploymentSizeWithinLimits(args UploadArgs) bool {
	zips := map[string]int64{}
	zips[args.ClientZip] = 50<<20 + 1024  // 50MB
	zips[args.ServerZip] = 100<<20 + 1024 // 100MB
	zips[args.ApiZip] = 100<<20 + 1024    // 100 MB

	for zipFile, maxSize := range zips {
		if zipFile == "" {
			continue
		}

		info, err := Stat(zipFile)

		if err != nil {
			slog.Errorf("cannot check file info: %s", err.Error())
			return false
		}

		if info == nil {
			// no file to upload
			continue
		}

		if info.Size() > maxSize {
			return false
		}
	}

	return true
}

func (u *Uploader) Upload(args UploadArgs) (*integrations.UploadResult, error) {
	conf := config.Get()

	// We need to configure the config at this point, manually, because
	// many environment variables will be missing in the runner environment
	switch u.conf.Provider {
	case config.ProviderAWS:
		if conf.AWS == nil {
			conf.AWS = &config.AwsConfig{
				AccountID:      u.conf.AccountID,
				Region:         u.conf.Region,
				LambdaRoleName: u.conf.LambdaRole,
				StorageBucket:  args.BucketName,
			}
		}

	case config.ProviderAlibaba:
		if conf.Alibaba == nil {
			conf.Alibaba = &config.AlibabaConfig{
				Region:        u.conf.Region,
				AccountID:     u.conf.AccountID,
				StorageBucket: args.BucketName,
			}
		}
	}

	if strings.HasPrefix(args.Runtime, "bun") {
		args.Runtime = config.NodeRuntime18
	}

	if config.IsStormkitCloud() && !u.isDeploymentSizeWithinLimits(args) {
		msg := "Deployment size is larger than allowed.\n" +
			"For client-side applications, the limit is 50MB and " +
			"for serverless applications 100MB."

		//lint:ignore ST1005 This message is being consumed by the frontend
		return nil, errors.New(msg)
	}

	return integrations.
		Client(integrations.ClientArgs{
			Provider:  u.conf.Provider,
			AccessKey: u.conf.AccessKey,
			SecretKey: u.conf.SecretKey,
			Region:    utils.GetString(args.Region, u.conf.Region),
		}).
		Upload(integrations.UploadArgs{
			MigrationsZip: args.MigrationsZip,
			ClientZip:     args.ClientZip,
			ServerZip:     args.ServerZip,
			ServerHandler: args.ServerHandler,
			APIZip:        args.ApiZip,
			APIHandler:    args.ApiHandler,
			EnvVars:       args.EnvVars,
			EnvID:         args.EnvID,
			AppID:         args.AppID,
			DeploymentID:  args.DeploymentID,
			Runtime:       args.Runtime,
			BucketName:    utils.GetString(args.BucketName, u.conf.BucketName),
		})
}
