package deployhandlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhooks"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"gopkg.in/guregu/null.v3"
)

const OutcomeSuccess = "success"
const OutcomeFailed = "failed"
const OutcomeSkipped = "skipped"
const OutcomeCancelled = "cancelled"

type deployCallbackRequest struct {
	DeployID string `json:"deployId"`

	// Commit related information
	RunID  string            `json:"runId"`
	Commit deploy.CommitInfo `json:"commit"`

	// Logs related information
	Logs string `json:"logs"`

	// Deployment result related information
	Result          integrations.UploadResult `json:"result"`  // The upload result
	UploadError     string                    `json:"error"`   // This field is generated when an error occurs during the upload
	Outcome         string                    `json:"outcome"` // Possible values: success | failure | cancelled | skipped
	Manifest        *deploy.BuildManifest     `json:"manifest"`
	HasStatusChecks bool                      `json:"hasStatusChecks"`

	// Final call
	Lock bool `json:"lock"`

	deployment *deploy.Deployment
}

// NewDeployCallbackRequest creates a new DeployCallbackRequest instance.
// Used mainly for testing purposes.
func NewDeployCallbackRequest(deployment *deploy.Deployment) deployCallbackRequest {
	return deployCallbackRequest{
		deployment: deployment,
	}
}

// handlerDeployCallback updates the deployment based on the request.
func handlerDeployCallback(req *shttp.RequestContext) *shttp.Response {
	data := deployCallbackRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	deployID, err := utils.DecryptID(data.DeployID)

	if err != nil {
		return shttp.NotAllowed()
	}

	depl, err := deploy.NewStore().MyDeployment(req.Context(), &deploy.DeploymentsQueryFilters{
		DeploymentID: deployID,
		IncludeLogs:  aws.Bool(false),
	})

	if err != nil {
		return shttp.Error(err)
	}

	if depl == nil {
		return shttp.NotFound()
	}

	data.deployment = depl

	if data.deployment.Status() != "running" {
		return &shttp.Response{
			Status: http.StatusConflict,
		}
	}

	if data.deployment.IsLocked() {
		return shttp.NoContent()
	}

	switch {
	case data.Lock:
		return LockDeployment(req, data)
	case data.Commit.ID.ValueOrZero() != "" || data.RunID != "":
		return UpdateCommit(req, data)
	case data.Logs != "" && data.deployment.ExitCode.Valid:
		return UpdateStatusCheckLogs(req, data)
	case data.Logs != "":
		return UpdateLogs(req, data)
	case data.Outcome != "":
		return UpdateExit(req, data)
	default:
		return shttp.BadRequest()
	}
}

// UpdateLogs updates the deployment logs.
func UpdateLogs(req *shttp.RequestContext, data deployCallbackRequest) *shttp.Response {
	store := deploy.NewStore()
	ctx := req.Context()

	if err := store.UpdateLogs(ctx, data.deployment.ID, data.Logs); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}

// UpdateStatusCheckLogs updates the status check logs for the deployment.
func UpdateStatusCheckLogs(req *shttp.RequestContext, data deployCallbackRequest) *shttp.Response {
	store := deploy.NewStore()
	ctx := req.Context()

	if err := store.UpdateStatusChecks(ctx, data.deployment.ID, data.Logs); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}

// UpdateCommit updates the commit information for the deployment.
func UpdateCommit(req *shttp.RequestContext, data deployCallbackRequest) *shttp.Response {
	store := deploy.NewStore()
	ctx := req.Context()

	if data.Commit.ID.ValueOrZero() != "" {
		if err := store.UpdateCommitInfo(ctx, data.deployment.ID, data.Commit); err != nil {
			return shttp.Error(err)
		}
	}

	if data.RunID != "" {
		if err := store.UpdateGithubRunID(ctx, data.deployment.ID, utils.StringToID(data.RunID)); err != nil {
			return shttp.Error(err)
		}
	}

	return shttp.OK()
}

// UpdateExit updates the deployment exit code and handles post-deployment tasks.
func UpdateExit(req *shttp.RequestContext, data deployCallbackRequest) *shttp.Response {
	// TODO: Retry job perhaps?
	if data.UploadError != "" {
		data.deployment.Error = null.NewString(
			fmt.Sprintf("Error: %s", data.UploadError),
			true,
		)
	}

	data.deployment.BuildManifest = data.Manifest

	switch data.Outcome {
	case OutcomeSuccess:
		data.deployment.ExitCode = null.NewInt(0, true)
	case OutcomeCancelled, OutcomeSkipped:
		data.deployment.ExitCode = null.NewInt(-1, true)
	default:
		data.deployment.ExitCode = null.NewInt(1, true)
	}

	// If the deployment branch == environment branch, it means we need to migrate the environment
	if data.Result.Migrations.Location != "" {
		logs := []string{}
		logs = append(logs, deploy.LogStep("database migrations"))

		results, err := RunMigrations(req.Context(), RunMigrationsArgs{
			DeploymentID:   data.deployment.ID,
			EnvID:          data.deployment.EnvID,
			Branch:         data.deployment.Branch,
			MigrationsFile: data.Result.Migrations.Location,
		})

		// If there was an error and no migrations were applied, set the deployment error
		if err != nil && len(results) == 0 {
			data.deployment.Error = null.StringFrom(err.Error())
		}

		for _, result := range results {
			if result == nil {
				continue
			}

			errorMsg := ""
			tickOrCross := "✓"

			if result.Error != "" {
				tickOrCross = "✗"
				errorMsg = " - Error: " + result.Error
				data.deployment.ExitCode = null.IntFrom(deploy.ExitCodeMigrationsFailed)
			}

			logs = append(logs, fmt.Sprintf("%s (%dms) %s%s", result.FileName, result.Duration.Milliseconds(), tickOrCross, errorMsg))
		}

		if len(results) == 0 {
			logs = append(logs, "No new migrations to apply.")
		}

		data.deployment.AddLogs(logs)
	}

	if err := deploy.NewStore().UpdateDeploymentResult(req.Context(), data.deployment, data.Result); err != nil {
		return shttp.BadRequest(map[string]any{
			"error": fmt.Sprintf("error while updating deployment result: %s", err.Error()),
		})
	}

	if config.IsSelfHosted() && data.deployment.BuildManifest != nil && len(data.deployment.BuildManifest.Runtimes) > 0 {
		cnf := admin.MustConfig()

		slog.Debug(slog.LogOpts{
			Msg:   fmt.Sprintf("auto install runtimes after deployment: %v", cnf.SystemConfig == nil || cnf.SystemConfig.AutoInstall),
			Level: slog.DL2,
		})

		if cnf.SystemConfig == nil || cnf.SystemConfig.AutoInstall {
			if err := admin.AddRuntimes(context.Background(), data.deployment.BuildManifest.Runtimes); err != nil {
				return shttp.Error(err, fmt.Sprintf("error while adding runtime configs to the instance: %s", err.Error()))
			}
		}
	}

	deployhooks.Exec(req.Context(), data.deployment)

	if !data.HasStatusChecks {
		return LockDeployment(req, data)
	}

	return shttp.OK()
}

type RunMigrationsArgs struct {
	DeploymentID   types.ID
	EnvID          types.ID
	Branch         string
	MigrationsFile string
}

// RunMigrations runs database migrations for the given environment.
func RunMigrations(ctx context.Context, args RunMigrationsArgs) ([]*buildconf.MigrationResult, error) {
	env, err := buildconf.NewStore().EnvironmentByID(ctx, args.EnvID)

	if err != nil {
		return nil, err
	}

	if env == nil {
		return nil, fmt.Errorf("deployment environment not found: %d", args.EnvID)
	}

	// Migrations not enabled
	if env.SchemaConf == nil || !env.SchemaConf.MigrationsEnabled {
		return nil, nil
	}

	// Skip migrations as we're not on the default branch
	if env.Branch != args.Branch {
		return nil, nil
	}

	migrationsZip, err := integrations.Client().GetFile(integrations.GetFileArgs{
		Location: args.MigrationsFile,
	})

	if err != nil {
		return nil, fmt.Errorf("error while fetching migrations file: %s", err.Error())
	}

	if migrationsZip == nil {
		return nil, fmt.Errorf("migrations file not found at location: %s", args.MigrationsFile)
	}

	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeMigrations)

	if err != nil {
		return nil, err
	}

	defer store.Close()

	if err := store.EnsureMigrationsTable(); err != nil {
		return nil, err
	}

	if err := store.AdvisoryLock(int64(env.ID)); err != nil {
		return nil, errors.New("failed to acquire advisory lock for running migrations; this may be caused by simultaneous deployments")
	}

	defer store.AdvisoryUnlock(int64(env.ID))

	migrations, err := store.Migrations(ctx)

	if err != nil {
		return nil, err
	}

	results := []*buildconf.MigrationResult{}

	err = file.ZipIterator(migrationsZip.Content, func(fileName string, content []byte) error {
		hash := utils.Hash(content)

		// Skip migrations that have already been applied successfully
		for _, migration := range migrations {
			if migration.ContentHash == hash && migration.ErrorMsg.ValueOrZero() == "" {
				return nil
			}
		}

		// Skip down migrations
		if strings.Contains(fileName, ".down.") {
			return nil
		}

		result, err := store.ApplyMigration(ctx, buildconf.ApplyMigrationArgs{
			MigrationName: fileName,
			Content:       content,
			SHA:           hash,
			DeploymentID:  args.DeploymentID,
		})

		results = append(results, result)

		if err != nil {
			return err
		}

		return nil
	})

	return results, err
}

// LockDeployment is called when the deployment is complete. A deployment is complete
// when all status checks have completed. If there are no status checks, it is complete
// when the main deployment process is complete.
//
// @since Runner v1.6.17
func LockDeployment(req *shttp.RequestContext, d deployCallbackRequest) *shttp.Response {
	isSuccess := d.Outcome == OutcomeSuccess

	if isSuccess {
		if err := deploy.AutoPublishIfNecessary(req.Context(), d.deployment); err != nil {
			return shttp.Error(err)
		}
	}

	var statusChecksPassed null.Bool

	if d.HasStatusChecks {
		statusChecksPassed = null.BoolFrom(isSuccess)
	}

	if err := deploy.NewStore().LockDeployment(req.Context(), d.deployment.ID, statusChecksPassed); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
