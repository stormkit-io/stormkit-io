package deployservice

import (
	"context"
	"fmt"
	"regexp"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v3"
)

var ErrBuildMinutesExceeded = fmt.Errorf("build minutes limit exceeded")

type Deployer interface {
	Deploy(context.Context, *app.App, *deploy.Deployment) error
}

type DefaultDeployer struct {
}

var MockDeployer Deployer

func New() Deployer {
	if MockDeployer != nil {
		return MockDeployer
	}

	return &DefaultDeployer{}
}

func (dd *DefaultDeployer) Deploy(ctx context.Context, a *app.App, d *deploy.Deployment) error {
	// Build caching is available to everyone on self-hosted instances. On
	// Stormkit Cloud it is an enterprise feature: premium and ultimate only.
	cacheEnabled := !config.IsStormkitCloud()

	if config.IsStormkitCloud() {
		store := user.NewStore()
		usr, err := store.UserMetrics(ctx, user.UserMetricsArgs{AppID: a.ID})

		if err != nil {
			return err
		}

		if usr != nil && !usr.HasBuildMinutes() {
			return ErrBuildMinutesExceeded
		}

		// UserMetrics already carries the owner's package, but it returns no
		// row until the first usage entry of the month exists; fall back to a
		// dedicated lookup in that case. Build caching is best-effort, so a
		// failed entitlement lookup disables caching instead of failing the
		// deployment.
		tier := ""

		if usr != nil {
			tier = usr.Metadata.PackageName
		} else if t, err := store.PackageNameByAppID(ctx, a.ID); err != nil {
			slog.Errorf("could not resolve package for app %d: %v", a.ID, err)
		} else {
			tier = t
		}

		cacheEnabled = tier == config.PackagePremium || tier == config.PackageUltimate
	}

	// Get git credentials
	gitCreds, err := a.GitCreds(ctx)

	if err != nil {
		return err
	}

	d.ConfigCopy, _ = d.MarshalConfigSnapshot()

	if d.BuildConfig == nil {
		d.BuildConfig = &buildconf.BuildConf{}
	}

	d.APIPathPrefix = null.NewString(
		utils.TrimPath(
			utils.GetString(
				d.BuildConfig.APIPathPrefix,
				d.BuildConfig.APIFolder,
				"api",
			),
		),
		true,
	)

	if !d.IsRestart {
		if d.IsAutoDeploy && d.BuildConfig != nil && d.BuildConfig.PriorityPattern != "" {
			if matched, _ := regexp.MatchString(d.BuildConfig.PriorityPattern, d.Commit.Message.ValueOrZero()); matched {
				d.IsPriority = true
			}
		}

		store := deploy.NewStore()

		// Insert the deployment first, so we can have an ID.
		if err := store.InsertDeployment(ctx, d); err != nil {
			return err
		}
	}

	// When caching is not allowed the runner receives no cache directories,
	// which is how it decides whether to restore and snapshot the cache.
	cacheDirs := d.BuildConfig.CacheDirs

	if !cacheEnabled {
		cacheDirs = nil
	}

	payload := DeploymentMessage{
		Client: ClientConfig{
			Repo:        d.RepoCloneURL(),
			Slug:        d.RepoSlug(),
			AccessToken: gitCreds,
		},

		Build: BuildConfig{
			Env:              d.Env,
			Branch:           d.Branch,
			ShouldPublish:    d.ShouldPublish,
			BuildCmd:         d.BuildConfig.BuildCmd,
			ServerCmd:        d.BuildConfig.ServerCmd,
			InstallCmd:       d.BuildConfig.InstallCmd,
			WorkDir:          d.BuildConfig.WorkDir,
			ServerFolder:     d.BuildConfig.ServerFolder,
			DistFolder:       d.BuildConfig.DistFolder,
			DeploymentID:     d.ID.String(),
			EnvID:            d.EnvID.String(),
			AppID:            d.AppID.String(),
			HeadersFile:      d.BuildConfig.HeadersFile,
			RedirectsFile:    d.BuildConfig.RedirectsFile,
			APIFolder:        utils.GetString(d.BuildConfig.APIFolder, "/api"),
			StatusChecks:     d.BuildConfig.StatusChecks,
			MigrationsFolder: d.MigrationsFolder.ValueOrZero(),
			CacheDirs:        cacheDirs,
			Vars: d.BuildConfig.InterpolatedVars(
				buildconf.InterpolatedVarsOpts{
					DeploymentID: d.ID.String(),
					AppID:        d.AppID.String(),
					EnvID:        d.EnvID.String(),
					Env:          d.Env,
					DisplayName:  a.DisplayName,
				},
			),
		},

		Config: &RunnerSettings{
			RunnerConfig: *config.Get().Runner,
			AutoInstall:  admin.MustConfig().IsAutoInstallEnabled(),
		},
	}

	queue := tasks.QueueDeployService

	if d.IsPriority {
		queue = tasks.QueueDeployServicePriority
	}

	return dd.sendPayloadToRedis(ctx, sendPayloadToRedisParams{message: payload, queue: queue})
}

type sendPayloadToRedisParams struct {
	message DeploymentMessage
	queue   string
}

func (dd *DefaultDeployer) sendPayloadToRedis(ctx context.Context, p sendPayloadToRedisParams) error {
	encrypted, err := p.message.Encrypt()

	if err != nil {
		return err
	}

	if p.queue == "" {
		p.queue = tasks.QueueDeployService
	}

	info, err := tasks.Enqueue(ctx, tasks.DeploymentStart, encrypted, &tasks.EnqueueOptions{
		MaxRetry:  10,
		QueueName: p.queue,
		TaskID:    fmt.Sprintf("deployment-%s", p.message.Build.DeploymentID),
	})

	if err != nil {
		slog.Errorf("could not enqueue task: %v", err)
	}

	if info != nil {
		slog.Debug(slog.LogOpts{
			Msg:   "enqueued task",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("task_id", info.ID),
				zap.String("queue", info.Queue),
			},
		})
	}

	return err
}
