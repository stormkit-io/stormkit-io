package runner

import (
	"fmt"
	"os"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
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

type bundleSizeLimit struct {
	name    string
	zipFile string
	maxSize int64
}

// checkDeploymentSize returns an error naming the bundle that busts its size
// limit, together with the actual size, so that the failure is actionable in
// the deployment logs.
func (u *Uploader) checkDeploymentSize(args UploadArgs) error {
	limits := []bundleSizeLimit{
		{name: "client", zipFile: args.ClientZip, maxSize: 50<<20 + 1024},  // 50MB
		{name: "server", zipFile: args.ServerZip, maxSize: 100<<20 + 1024}, // 100MB
		{name: "api", zipFile: args.ApiZip, maxSize: 100<<20 + 1024},       // 100MB
	}

	for _, limit := range limits {
		if limit.zipFile == "" {
			continue
		}

		info, err := Stat(limit.zipFile)

		if err != nil {
			return fmt.Errorf("cannot check the size of the %s bundle: %w", limit.name, err)
		}

		// no file to upload
		if info == nil {
			continue
		}

		if info.Size() > limit.maxSize {
			//lint:ignore ST1005 This message is being consumed by the frontend
			return fmt.Errorf(
				"Deployment size is larger than allowed.\n"+
					"The %s bundle is %s while the limit is %s.\n"+
					"For client-side applications, the limit is 50MB and for serverless applications 100MB.",
				limit.name, megabytes(info.Size()), megabytes(limit.maxSize),
			)
		}
	}

	return nil
}

func megabytes(b int64) string {
	return fmt.Sprintf("%.1fMB", float64(b)/(1<<20))
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

	if config.IsStormkitCloud() {
		if err := u.checkDeploymentSize(args); err != nil {
			return nil, err
		}
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
