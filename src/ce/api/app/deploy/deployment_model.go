package deploy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

const ExitCodeSuccess = int(0)
const ExitCodeStopped = int(-1)
const ExitCodeFailed = int(1)

type Log struct {
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Status    bool                   `json:"status"`
	Payload   map[string]interface{} `json:"payload"`
	Duration  int64                  `json:"duration"` // in seconds
	timestamp int64                  `json:"-"`
	systemLog string                 `json:"-"`
}

type CommitInfo struct {
	Author  null.String `json:"author"`
	Message null.String `json:"message"`
	ID      null.String `json:"sha"`
}

// APIDeployResponse represents the data that we send to the frontend client.
type APIDeployResponse struct {
	ID                string           `json:"id,omitempty"`
	AppID             string           `json:"appId,omitempty"`
	EnvID             string           `json:"envId,omitempty"`
	ExitCode          *int             `json:"exit"`
	NumberOfFiles     int              `json:"numberOfFiles"`
	IsRunning         bool             `json:"isRunning"`
	Preview           string           `json:"preview,omitempty"`
	TotalSizeInBytes  int              `json:"totalSizeInBytes"`
	ServerPackageSize int              `json:"serverPackageSize"`
	CreatedAt         utils.Unix       `json:"createdAt"`
	StoppedAt         utils.Unix       `json:"stoppedAt"`
	Commit            CommitInfo       `json:"commit"`
	Config            map[string]any   `json:"config,omitempty"`
	Published         []map[string]any `json:"published,omitempty"`
	Branch            string           `json:"branch"`
	Logs              []*Log           `json:"logs"`
}

// Deployment represents a deployment.
type Deployment struct {
	ID                 types.ID       `json:"id,string,omitempty" db:"deployment_id"`
	AppID              types.ID       `json:"appId,string,omitempty" db:"app_id"`
	EnvID              types.ID       `json:"envId,string,omitempty" db:"env_id"`
	Env                string         `json:"env,omitempty" db:"env_name"` // @deprecated: Env is the environment name used for this deployment. Will be replaced with EnvID
	Branch             string         `json:"branch" db:"branch"`          // Branch is name of the branch that was deployed.
	ConfigCopy         []byte         `json:"-" db:"config_snapshot"`
	configCopyCached   map[string]any `json:"-"`
	S3NumberOfFiles    null.Int       `json:"numberOfFiles" db:"s3_number_of_files"`
	S3TotalSizeInBytes null.Int       `json:"totalSizeInBytes"`
	ServerPackageSize  null.Int       `json:"serverPackageSize,omitempty" db:"server_package_size"`
	ExitCode           null.Int       `json:"exit" db:"exit_code"`
	Logs               null.String    `json:"-" db:"logs"`
	PullRequestNumber  null.Int       `json:"pullRequestNumber" db:"pull_request_number"`
	IsFork             bool           `json:"isFork" db:"is_fork"`
	CheckoutRepo       string         `json:"checkoutRepo" db:"checkout_repo"` // CheckoutRepo is the repository used to check out while deploying the application.
	BuildManifest      *BuildManifest `json:"buildManifest,omitempty" db:"build_manifest"`
	ShouldPublish      bool           `json:"shouldPublish" db:"auto_publish"` // ShouldPublish is boolean value which stores an overwrite for the AutoPublish field of an environment.
	IsAutoDeploy       bool           `json:"isAutoDeploy" db:"auto_deploy"`
	IsImmutable        null.Bool      `json:"-"`
	StatusChecks       null.String    `json:"-"`
	StatusChecksPassed null.Bool      `json:"statusChecksPassed,omitempty"`
	CreatedAt          utils.Unix     `json:"createdAt,omitempty" db:"created_at"`
	StoppedAt          utils.Unix     `json:"stoppedAt,omitempty" db:"stopped_at"`
	DeletedAt          utils.Unix     `json:"deletedAt,omitempty" db:"deleted_at"`
	Commit             CommitInfo     `json:"commit" db:"commit_id, commit_author, commit_message"`
	Error              null.String    `json:"-" db:"error"` // Error represents the deployment error. It's for internal use only.
	StorageLocation    null.String    `json:"storageLocation,omitempty" db:"storage_location"`
	FunctionLocation   null.String    `json:"functionLocation,omitempty" db:"function_location"` // aws:<function-arn>/<version>
	APILocation        null.String    `json:"apiLocation,omitempty" db:"api_location"`           // aws:<function-arn>/<version>
	APIPathPrefix      null.String    `json:"apiPathPrefix,omitempty"`
	APIPackageSize     null.Int       `json:"apiPackageSize,omitempty" db:"api_package_size"`
	WebhookEvent       any            `json:"-"` // The webhook event that triggers the deployment

	// GithubRunID is the associated run id with the deployment.
	// It is obtained by printing $GITHUB_RUN_ID in GitHub actions.
	// This value is used to retrieve the jobs and then the logs.
	GithubRunID null.Int `json:"-" db:"github_run_id"`

	// Published represents the publish information.
	// It's a json string fetched from the database that contains
	// the environment id and the released percentage.
	PublishedV2 PublishedInfoV2 `json:"-"`

	User          *user.User           `json:"-"`
	BuildConfig   *buildconf.BuildConf `json:"-"` // ConfigCopy is the snapshot of the environment used during the deployment.
	EnvBranchName string               `json:"-"` // EnvBranchName represents the branch name that is associated with the given environment.
	DisplayName   string               `json:"-"` // DisplayName is the name of the app. It is injected to the deployment object.
	IsRestart     bool                 `json:"-"`
}

// PublishedInfo represents information on the publish details
// for the given deployment.
type PublishedInfoV2 []struct {
	EnvID      types.ID `json:"envId"`
	Percentage float64  `json:"percentage"`
}

func (pi *PublishedInfoV2) Scan(value any) error {
	if value != nil {
		return json.Unmarshal(value.([]byte), &pi)
	}

	return nil
}

func (pi PublishedInfoV2) MarshalJSON() ([]byte, error) {
	hash := []map[string]any{}

	for _, element := range pi {
		hash = append(hash, map[string]any{
			"envId":      types.ID(element.EnvID).String(),
			"percentage": element.Percentage,
		})
	}

	return json.Marshal(hash)
}

// Env represents an environment with deployments.
type Env struct {
	buildconf.Env

	Deployments []*Deployment `json:"deployments"`
}

// MarshalJSON implements the json marshaler interface.
func (e *Env) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Environment buildconf.Env `json:"env"`
		Deployments []*Deployment `json:"deployments"`
	}{
		Environment: e.Env,
		Deployments: e.Deployments,
	})
}

// RequestData represents the data which can be overwritten by a request.
type RequestData struct {
	// Publish indicates whether this deployment should be
	// published or not.
	Publish bool `json:"publish"`

	// Branch is the branch to deploy.
	Branch string `json:"branch"`

	// The distribution folder.
	DistFolder string `json:"distFolder"`

	// Cmd is the command to run
	BuildCmd string `json:"buildCmd"`
}

// New returns a new deployment instance.
func New(appID types.ID) *Deployment {
	d := &Deployment{
		AppID: appID,
	}

	return d
}

// MarshalJSON implements the json marshaler interface.
func (d *Deployment) MarshalJSON() ([]byte, error) {
	exitCode := int(d.ExitCode.ValueOrZero())

	apiDeployModel := &APIDeployResponse{
		ID:                d.ID.String(),
		EnvID:             d.EnvID.String(),
		AppID:             d.AppID.String(),
		Branch:            d.Branch,
		Logs:              d.PrepareLogs(d.Logs.ValueOrZero(), false),
		Commit:            d.Commit,
		CreatedAt:         d.CreatedAt,
		StoppedAt:         d.StoppedAt,
		TotalSizeInBytes:  int(d.S3TotalSizeInBytes.ValueOrZero()),
		ServerPackageSize: int(d.ServerPackageSize.ValueOrZero()),
		IsRunning:         exitCode == 0 && !d.ExitCode.Valid,
	}

	if d.ExitCode.Valid {
		apiDeployModel.ExitCode = &exitCode

		if exitCode == 0 {
			apiDeployModel.Preview = admin.MustConfig().PreviewURL(d.DisplayName, d.ID.String())
		}
	}

	_ = json.Unmarshal(d.ConfigCopy, &apiDeployModel.Config)
	return json.Marshal(apiDeployModel)
}

// IsLocked returns true when a deployment is locked.
func (d *Deployment) IsLocked() bool {
	return d.IsImmutable.Valid && d.IsImmutable.ValueOrZero()
}

// Snapshot returns the deployment Config Copy in a
// map format or it returns nil.
func (d *Deployment) Snapshot() map[string]any {
	if d.configCopyCached != nil {
		return d.configCopyCached
	}

	if d.ConfigCopy == nil {
		return nil
	}

	d.configCopyCached = map[string]any{}
	_ = json.Unmarshal(d.ConfigCopy, &d.configCopyCached)
	return d.configCopyCached
}

// PrepareConfigSnapshot prepares the deployment config snapshot.
func (d *Deployment) MarshalConfigSnapshot() ([]byte, error) {
	return json.Marshal(map[string]any{
		"build": d.BuildConfig,
		"env":   d.Env,
		"envId": d.EnvID.String(),
	})
}

// HasStatusChecks checks the deployment snapshot to determine if it had
// status checks on when executed.
func (d *Deployment) HasStatusChecks() bool {
	// Check if we have status checks
	snapshot := d.Snapshot()

	if _, ok := snapshot["build"]; !ok {
		return false
	}

	if _, ok := snapshot["build"].(map[string]any)["statusChecks"]; !ok {
		return false
	}

	return true
}

// Status returns the deployment status based on the exit code.
// Possible values are: running | success | failed
func (d *Deployment) Status() string {
	running := "running"
	success := "success"
	failed := "failed"

	if !d.ExitCode.Valid {
		return running
	}

	if d.ExitCode.ValueOrZero() == 0 {
		if d.HasStatusChecks() {
			if !d.StatusChecksPassed.Valid {
				return running
			}

			if d.StatusChecksPassed.ValueOrZero() {
				return success
			}

			return failed
		}

		return success
	}

	return failed
}

// RepoCloneURL returns the fully qualified repository name to clone the repository.
func (d *Deployment) RepoCloneURL() string {
	pieces := strings.Split(d.CheckoutRepo, "/")

	if len(pieces) >= 3 {
		provider, owner, slug := pieces[0], pieces[1], strings.TrimSuffix(strings.Join(pieces[2:], "/"), ".git")

		if provider == "github" {
			return fmt.Sprintf("https://github.com/%s/%s.git", owner, slug)
		} else if provider == "bitbucket" {
			return fmt.Sprintf("git@bitbucket.org:%s/%s.git", owner, slug)
		} else if provider == "gitlab" {
			return fmt.Sprintf("https://gitlab.com/%s/%s.git", owner, slug)
		} else {
			panic("Unknown provider")
		}
	}

	return ""
}

// RepoSlug returns the owner and slug of the repository.
// For instance: stormkit-io/app-stormkit-io
func (d *Deployment) RepoSlug() string {
	pieces := strings.Split(d.CheckoutRepo, "/")

	if len(pieces) >= 3 {
		owner, slug := pieces[1], strings.TrimSuffix(strings.Join(pieces[2:], "/"), ".git")
		return fmt.Sprintf("%s/%s", owner, slug)
	}

	return ""
}

// PrepareLogs prepares the deployment logs and returns an array of log objects.
func (d *Deployment) PrepareLogs(rawLogs string, isStatusChecks bool) []*Log {
	if rawLogs == "" &&
		d.S3NumberOfFiles.ValueOrZero() == 0 &&
		d.ServerPackageSize.ValueOrZero() == 0 &&
		d.APIPackageSize.ValueOrZero() == 0 {
		return nil

	}

	logs := []*Log{}

	// This is a special case for old-style deployments
	if err := json.Unmarshal([]byte(rawLogs), &logs); err == nil {
		return logs
	}

	lines := strings.Split(rawLogs, "\n")

	var currentBatch *Log
	var lastStep *Log

	lastStepTimestamp := d.CreatedAt.Unix()
	buildingFinished := len(lines) == 0 && d.ExitCode.Valid
	deploymentComplete := false

	for _, line := range lines {
		isHeader := strings.HasPrefix(line, "[sk-step] ")
		timestamp := int64(0)

		if isHeader {
			pieces := strings.Split(line, " [ts:")
			line = pieces[0]

			if len(pieces) > 1 {
				timestamp = utils.StringToInt64(strings.Replace(pieces[1], "]", "", 1))
			}
		}

		log := &Log{
			Status:    false,
			timestamp: timestamp,
		}

		// Ignore system messages
		// This one comes from GitHub builds and is the last step.
		// So we can add our build details.
		if isHeader && strings.HasPrefix(line, "[sk-step] [system] ") {
			log.systemLog = line
			logs = append(logs, log)

			if strings.Contains(line, "building finished") {
				buildingFinished = true
				lastStepTimestamp = timestamp
			} else if strings.Contains(line, "deployment finished") {
				deploymentComplete = true
			}

			continue
		}

		// We are going to batch messages. To do this, we walk line by line
		// and detect anything that starts with [sk-step]. It means we're
		// reading a title.
		if isHeader {
			log.Status = true
			log.Title = line[10:]
			currentBatch = log
			logs = append(logs, currentBatch)
			lastStep = log
			continue
		}

		if currentBatch == nil {
			continue
		}

		if config.IsStormkitCloud() {
			line = strings.ReplaceAll(line, "/home/runner/work/deployer-service/deployer-service/repo/", "/stormkit/app")
		}

		currentBatch.Message = currentBatch.Message + line + "\n"
	}

	if !isStatusChecks {
		isSuccess := d.ExitCode.ValueOrZero() == 0 && d.ExitCode.Valid

		if buildingFinished || isSuccess {
			logs = append(logs, d.deploymentsResult(lastStepTimestamp))
		} else if !deploymentComplete {
			// Let's sync the last step
			lastStep.Status = isSuccess
		}
	}

	// Next bit of code calculates the durations of each step by
	// subracting the timestamp of the next step from the current step.
	//
	// For instance:
	//
	// 1. npm ci [timestamp: 10]
	// 2. npm run build [timestamp: 12]
	//
	// 1. npm ci [duration = 2s]
	var prevStep *Log

	filteredLogs := []*Log{}

	for _, step := range logs {
		if prevStep != nil {
			prevStep.Duration = step.timestamp - prevStep.timestamp
		} else if d.CreatedAt.Valid {
			step.Duration = step.timestamp - d.CreatedAt.Unix()
		}

		prevStep = step

		// We store system logs to calculate the end time. Now filter them out.
		if step.Title != "" {
			filteredLogs = append(filteredLogs, step)
		}
	}

	// Calculate the duration of the last step
	if filteredLogsLen := len(filteredLogs); filteredLogsLen > 0 {
		filteredLogs[filteredLogsLen-1].Duration = d.StoppedAt.Unix() - filteredLogs[filteredLogsLen-1].timestamp

		if isStatusChecks {
			filteredLogs[filteredLogsLen-1].Status = d.StatusChecksPassed.ValueOrZero()
		}
	}

	return filteredLogs
}

// deploymentResult returns the deployment result log. The startTimestamp
// is calculated based on the timestamp of "building finished" system log.
// deployment.stoppedAt is set when the deployment is complete (before the status checks).
// Therefore, we can substract the two and calculate the duration of the step.
func (d *Deployment) deploymentsResult(startTimestamp int64) *Log {
	log := &Log{
		timestamp: startTimestamp,
		Status:    d.ExitCode.Valid && d.ExitCode.ValueOrZero() == 0,
		Title:     "deploy",
	}

	if d.Error.ValueOrZero() != "" {
		d.ExitCode = null.NewInt(1, true)
		log.Message = d.Error.ValueOrZero()
		return log
	}

	if d.ExitCode.ValueOrZero() == -1 {
		log.Message = "Deployment has been stopped manually."
		return log
	}

	// We're still uploading artifacts
	if d.S3NumberOfFiles.ValueOrZero() == 0 &&
		d.APIPackageSize.ValueOrZero() == 0 &&
		d.ServerPackageSize.ValueOrZero() == 0 {
		if d.ExitCode.Valid {
			switch d.ExitCode.ValueOrZero() {
			case 0:
				log.Message = "Deployment completed successfully with no artifacts."
				return log
			default:
				log.Message = "Deployment failed."
				return log
			}
		}

		d.ExitCode = null.NewInt(0, false)
		log.Status = true
		log.Message = "Deploying your application... This may take a while..."
		return log
	}

	log.Status = true

	if d.S3NumberOfFiles.ValueOrZero() != 0 {
		log.Message = strings.Join([]string{
			log.Message,
			fmt.Sprintf(
				"Successfully deployed client side.\n"+
					"Total bytes uploaded: %s\n\n",
				byteCountDecimal(d.S3TotalSizeInBytes.ValueOrZero()),
			),
		}, "\n")
	}

	if d.ServerPackageSize.ValueOrZero() != 0 {
		log.Message = strings.Join([]string{
			log.Message,
			"Successfully deployed server side.",
			fmt.Sprintf("Package size: %s\n\n", byteCountDecimal(d.ServerPackageSize.ValueOrZero())),
		}, "\n")
	}

	if d.APIPackageSize.ValueOrZero() != 0 {
		log.Message = strings.Join([]string{
			log.Message,
			"Successfully deployed api.",
			fmt.Sprintf("Package size: %s", byteCountDecimal(d.APIPackageSize.ValueOrZero())),
		}, "\n")
	}

	return log
}

// byteCountDecimal converts the given bytes into a human readable format.
func byteCountDecimal(b int64) string {
	const unit = 1000

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "kMGTPE"[exp])
}
