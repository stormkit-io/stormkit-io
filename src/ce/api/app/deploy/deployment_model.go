package deploy

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

const ExitCodeSuccess = int64(0)
const ExitCodeStopped = int64(-1)
const ExitCodeTimeout = int64(-2)
const ExitCodeFailed = int64(1)
const ExitCodeMigrationsFailed = int64(2)

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

type UploadResult struct {
	ClientBytes        int64  `json:"clientBytes"`
	ServerBytes        int64  `json:"serverBytes"`
	MigrationsBytes    int64  `json:"migrationsBytes"`
	ServerlessBytes    int64  `json:"serverlessBytes"`    // this is used to be called apiPackage
	ServerlessLocation string `json:"serverlessLocation"` // e.g., aws:<function-arn>/<version>
	ServerLocation     string `json:"serverLocation"`     // e.g., aws:<function-arn>/<version> or path to server zip
	ClientLocation     string `json:"clientLocation"`     // e.g., s3://bucket/path/to/client/files
	MigrationsLocation string `json:"migrationsLocation"` // e.g., s3://bucket/path/to/migrations/files
}

// Scan implements the sql.Scanner interface
func (ur *UploadResult) Scan(value any) error {
	if value != nil {
		return json.Unmarshal(value.([]byte), &ur)
	}

	return nil
}

// Value implements the driver.Valuer interface
func (ur *UploadResult) Value() (driver.Value, error) {
	if ur == nil {
		return nil, nil
	}

	return json.Marshal(ur)
}

// Deployment represents a deployment.
type Deployment struct {
	ID                 types.ID       `json:"id,string,omitempty"`
	AppID              types.ID       `json:"appId,string,omitempty"`
	EnvID              types.ID       `json:"envId,string,omitempty"`
	Env                string         `json:"env,omitempty"` // @deprecated: Env is the environment name used for this deployment. Will be replaced with EnvID
	Branch             string         `json:"branch"`        // Branch is name of the branch that was deployed.
	ConfigCopy         []byte         `json:"-"`
	configCopyCached   map[string]any `json:"-"`
	ExitCode           null.Int       `json:"exit"`
	Logs               null.String    `json:"-"`
	PullRequestNumber  null.Int       `json:"pullRequestNumber"`
	IsFork             bool           `json:"isFork"`
	CheckoutRepo       string         `json:"checkoutRepo"` // CheckoutRepo is the repository used to check out while deploying the application.
	BuildManifest      *BuildManifest `json:"buildManifest,omitempty"`
	ShouldPublish      bool           `json:"shouldPublish"` // ShouldPublish is boolean value which stores an overwrite for the AutoPublish field of an environment.
	IsAutoDeploy       bool           `json:"isAutoDeploy"`
	IsImmutable        null.Bool      `json:"-"`
	StatusChecks       null.String    `json:"-"`
	StatusChecksPassed null.Bool      `json:"statusChecksPassed,omitempty"`
	CreatedAt          utils.Unix     `json:"createdAt,omitempty"`
	StoppedAt          utils.Unix     `json:"stoppedAt,omitempty"`
	DeletedAt          utils.Unix     `json:"deletedAt,omitempty"`
	Commit             CommitInfo     `json:"commit"`
	Error              null.String    `json:"-"` // Error represents the deployment error. It's for internal use only.
	APIPathPrefix      null.String    `json:"apiPathPrefix,omitempty"`
	WebhookEvent       any            `json:"-"` // The webhook event that triggers the deployment
	MigrationsFolder   null.String    `json:"migrationsFolder,omitempty"`
	UploadResult       *UploadResult  `json:"uploadResult,omitempty"`
	IsPriority         bool           `json:"isPriority,omitempty"`

	// GithubRunID is the associated run id with the deployment.
	// It is obtained by printing $GITHUB_RUN_ID in GitHub actions.
	// This value is used to retrieve the jobs and then the logs.
	GithubRunID null.Int `json:"-"`

	// Published represents the publish information.
	// It's a json string fetched from the database that contains
	// the environment id and the released percentage.
	Published PublishedInfo `json:"-"`

	BuildConfig *buildconf.BuildConf `json:"-"` // ConfigCopy is the snapshot of the environment used during the deployment.
	DisplayName string               `json:"-"` // DisplayName is the name of the app. It is injected to the deployment object.
	IsRestart   bool                 `json:"-"`
}

// PublishedInfo represents information on the publish details
// for the given deployment.
type PublishedInfo []struct {
	EnvID      types.ID `json:"envId"`
	Percentage float64  `json:"percentage"`
}

func (pi *PublishedInfo) Scan(value any) error {
	if value != nil {
		return json.Unmarshal(value.([]byte), &pi)
	}

	return nil
}

// RequestData represents the data which can be overwritten by a request.
type RequestData struct {
	// Publish indicates whether this deployment should be
	// published or not.
	Publish bool `json:"publish"`

	// Branch is the branch to deploy.
	Branch string `json:"branch"`
}

// New returns a new deployment instance.
func New(a *app.App) *Deployment {
	d := &Deployment{
		AppID:        a.ID,
		CheckoutRepo: a.Repo,
		Branch:       a.DefaultBranch(),
		DisplayName:  a.DisplayName,
	}

	return d
}

// includeMailerVars includes the mailer variables into the deployment build config.
func (d *Deployment) includeMailerVars(mailer *buildconf.MailerConf) {
	if mailer == nil {
		return
	}

	if d.BuildConfig == nil {
		d.BuildConfig = &buildconf.BuildConf{}
	}

	if d.BuildConfig.Vars == nil {
		d.BuildConfig.Vars = make(map[string]string)
	}

	if d.BuildConfig.Vars["MAILER_URL"] == "" {
		d.BuildConfig.Vars["MAILER_URL"] = mailer.String()
	}
}

// includeSchemaVars includes the database schema variables into the deployment build config.
func (d *Deployment) includeSchemaVars(schema *buildconf.SchemaConf) {
	if d.BuildConfig == nil {
		d.BuildConfig = &buildconf.BuildConf{}
	}

	if d.BuildConfig.Vars == nil {
		d.BuildConfig.Vars = make(map[string]string)
	}

	if d.BuildConfig.Vars["POSTGRES_HOST"] == "" {
		d.BuildConfig.Vars["POSTGRES_HOST"] = schema.Host
	}

	if d.BuildConfig.Vars["POSTGRES_PORT"] == "" {
		d.BuildConfig.Vars["POSTGRES_PORT"] = schema.Port
	}

	if d.BuildConfig.Vars["POSTGRES_DB"] == "" {
		d.BuildConfig.Vars["POSTGRES_DB"] = schema.DBName
	}

	if d.BuildConfig.Vars["POSTGRES_SCHEMA"] == "" {
		d.BuildConfig.Vars["POSTGRES_SCHEMA"] = schema.SchemaName
	}

	if d.BuildConfig.Vars["POSTGRES_USER"] == "" {
		d.BuildConfig.Vars["POSTGRES_USER"] = schema.AppUserName
	}

	if d.BuildConfig.Vars["POSTGRES_PASSWORD"] == "" {
		d.BuildConfig.Vars["POSTGRES_PASSWORD"] = schema.AppPassword
	}

	if d.BuildConfig.Vars["DATABASE_URL"] == "" {
		d.BuildConfig.Vars["DATABASE_URL"] = schema.URL()
	}
}

// PopulateFromEnv populates the deployment from the given environment.
func (d *Deployment) PopulateFromEnv(env *buildconf.Env) {
	d.Env = env.Name
	d.EnvID = env.ID
	d.Branch = env.Branch
	d.BuildConfig = env.Data
	d.ShouldPublish = env.AutoPublish

	if env.SchemaConf != nil {
		if env.SchemaConf.MigrationsEnabled {
			d.MigrationsFolder = null.StringFrom(env.SchemaConf.MigrationsFolder)
		}

		if env.SchemaConf.InjectEnvVars {
			d.includeSchemaVars(env.SchemaConf)
		}
	}

	if env.MailerConf != nil {
		d.includeMailerVars(env.MailerConf)
	}
}

type DeployCandidatePayload struct {
	Branch            string
	CommitSha         string
	CheckoutRepo      string
	IsFork            bool
	PullRequestNumber int64
	WebhookEvent      any
}

// PopulateFromDeployCandidate populates the deployment from the given deploy candidate.
func (d *Deployment) PopulateFromDeployCandidate(a *app.DeployCandidate, p DeployCandidatePayload) {
	// Info coming from environment
	d.Env = a.EnvName
	d.EnvID = a.EnvID
	d.IsAutoDeploy = true
	d.BuildConfig = a.BuildConfig
	d.ShouldPublish = a.ShouldPublish
	d.WebhookEvent = p.WebhookEvent

	// Info coming from payload
	d.Branch = p.Branch
	d.Commit.ID = null.NewString(p.CommitSha, p.CommitSha != "")
	d.CheckoutRepo = p.CheckoutRepo
	d.IsFork = p.IsFork
	d.PullRequestNumber = null.NewInt(p.PullRequestNumber, p.PullRequestNumber != 0)

	if a.SchemaConf != nil {
		if a.SchemaConf.MigrationsEnabled {
			d.MigrationsFolder = null.StringFrom(a.SchemaConf.MigrationsFolder)
		}

		if a.SchemaConf.InjectEnvVars {
			d.includeSchemaVars(a.SchemaConf)
		}
	}

	if a.MailerConf != nil {
		d.includeMailerVars(a.MailerConf)
	}
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

	// The snapshot is a frozen copy of the deployed config; mask its env-var
	// values so the deployment endpoints don't expose secrets in plaintext.
	if build, ok := d.configCopyCached["build"].(map[string]any); ok {
		if rawVars, ok := build["vars"].(map[string]any); ok {
			vars := make(map[string]string, len(rawVars))

			for key, value := range rawVars {
				if s, ok := value.(string); ok {
					vars[key] = s
				}
			}

			build["vars"] = buildconf.MaskVars(vars)
		}
	}

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

// maxFailureSummaryLines caps the log tail surfaced inline on a failed
// deployment, so the reason is available without fetching the full logs.
const maxFailureSummaryLines = 20

// FailureSummary returns a short, human-readable reason a deployment failed,
// for callers that have the deployment object but not the full logs. It reads
// the tail of the failed step's captured output -- where a build-command
// failure surfaces -- preferring the build logs and falling back to the
// status-check logs, then to any explicitly recorded error. It returns an
// empty string for a deployment that did not fail. The logs must be loaded on
// the deployment for a build-log summary to be produced.
func (d *Deployment) FailureSummary() string {
	if d.Status() != "failed" {
		return ""
	}

	if s := failedStepMessage(d.PrepareLogs(d.Logs.ValueOrZero(), false)); s != "" {
		return lastLines(s, maxFailureSummaryLines)
	}

	if s := failedStepMessage(d.PrepareLogs(d.StatusChecks.ValueOrZero(), true)); s != "" {
		return lastLines(s, maxFailureSummaryLines)
	}

	return strings.TrimSpace(d.Error.ValueOrZero())
}

// failedStepMessage returns the message of the last failed step that carries
// one, which for a build-command failure is the step whose output ends in the
// error.
func failedStepMessage(logs []*Log) string {
	for i := len(logs) - 1; i >= 0; i-- {
		if !logs[i].Status {
			if msg := strings.TrimSpace(logs[i].Message); msg != "" {
				return msg
			}
		}
	}

	return ""
}

// lastLines returns at most maxLines of the trailing, non-empty lines of s.
func lastLines(s string, maxLines int) string {
	lines := []string{}

	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimRight(line, "\r"))
		}
	}

	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}

	return strings.Join(lines, "\n")
}

// RepoCloneURL returns the fully qualified repository name to clone the repository.
func (d *Deployment) RepoCloneURL() string {
	pieces := strings.Split(d.CheckoutRepo, "/")

	if len(pieces) >= 3 {
		provider, owner, slug := pieces[0], pieces[1], strings.TrimSuffix(strings.Join(pieces[2:], "/"), ".git")

		switch provider {
		case "github":
			return fmt.Sprintf("https://github.com/%s/%s.git", owner, slug)
		case "bitbucket":
			return fmt.Sprintf("git@bitbucket.org:%s/%s.git", owner, slug)
		case "gitlab":
			return fmt.Sprintf("https://gitlab.com/%s/%s.git", owner, slug)
		default:
			slog.Errorf("repo provider is unknown: %s", d.CheckoutRepo)
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

// AddLogs appends the given logs to the deployment logs.
func (d *Deployment) AddLogs(logs []string) {
	if len(logs) == 0 {
		return
	}

	d.Logs = null.StringFrom(d.Logs.ValueOrZero() + "\n" + strings.Join(logs, "\n"))
}

// PrepareLogs prepares the deployment logs and returns an array of log objects.
func (d *Deployment) PrepareLogs(rawLogs string, isStatusChecks bool) []*Log {
	if rawLogs == "" && d.UploadResult == nil {
		return nil

	}

	// Structured logs written by the runner: one step record per line.
	if steps, ok := ParseStepLogs(rawLogs); ok {
		return d.prepareStepLogs(steps, isStatusChecks)
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
	buildingFinishedIdx := -1
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
				buildingFinishedIdx = len(logs) - 1
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
		isSuccess := d.ExitCode.ValueOrZero() == ExitCodeSuccess && d.ExitCode.Valid

		if d.ExitCode.ValueOrZero() == ExitCodeMigrationsFailed && lastStep != nil {
			lastStep.Status = false
		} else if buildingFinished || isSuccess {
			result := d.deploymentsResult(lastStepTimestamp)

			// Steps logged after the "building finished" marker (e.g. database
			// migrations) carry later timestamps than the deploy step, which
			// starts at the marker. Insert the deploy step right after the
			// marker so the duration calculation below sees timestamps in
			// chronological order.
			if buildingFinishedIdx >= 0 && buildingFinishedIdx+1 < len(logs) {
				logs = append(logs[:buildingFinishedIdx+1], append([]*Log{result}, logs[buildingFinishedIdx+1:]...)...)
			} else {
				logs = append(logs, result)
			}
		} else if !deploymentComplete && lastStep != nil {
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

	if d.ExitCode.ValueOrZero() == ExitCodeStopped {
		log.Message = "Deployment has been stopped manually."
		return log
	}

	// We're still uploading artifacts
	if d.UploadResult == nil {
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
	log.Message = d.UploadResult.message()

	return log
}

// message summarizes the upload result for the deploy step's log output.
func (ur *UploadResult) message() string {
	msg := ""

	if ur.ClientBytes != 0 {
		msg = strings.Join([]string{
			msg,
			fmt.Sprintf(
				"Successfully deployed client side.\n"+
					"Total bytes uploaded: %s\n\n",
				byteCountDecimal(ur.ClientBytes),
			),
		}, "\n")
	}

	if ur.ServerBytes != 0 {
		msg = strings.Join([]string{
			msg,
			"Successfully deployed server side.",
			fmt.Sprintf("Package size: %s\n\n", byteCountDecimal(ur.ServerBytes)),
		}, "\n")
	}

	if ur.ServerlessBytes != 0 {
		msg = strings.Join([]string{
			msg,
			"Successfully deployed api.",
			fmt.Sprintf("Package size: %s", byteCountDecimal(ur.ServerlessBytes)),
		}, "\n")
	}

	return msg
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

// LogStep formats a deployment step log.
func LogStep(title string) string {
	return fmt.Sprintf("[sk-step] %s [ts:%d]\n", title, time.Now().Unix())
}

// JSON returns a serialisable map representation of the deployment.
// When withLogs is true, the logs and status-checks fields are populated.
func (d *Deployment) JSON(withLogs bool) map[string]any {
	appID := d.AppID.String()
	envID := d.EnvID.String()
	depID := d.ID.String()

	var deploymentLogs []*Log
	var statusChecksLogs []*Log

	if withLogs {
		deploymentLogs = d.PrepareLogs(d.Logs.ValueOrZero(), false)
		statusChecksLogs = d.PrepareLogs(d.StatusChecks.ValueOrZero(), true)
	}

	var stoppedAt any

	if s := d.StoppedAt.UnixStr(); s != "" {
		stoppedAt = s
	}

	var uploadResult any

	if d.UploadResult != nil {
		uploadResult = map[string]any{
			"clientBytes":     d.UploadResult.ClientBytes,
			"serverBytes":     d.UploadResult.ServerBytes,
			"serverlessBytes": d.UploadResult.ServerlessBytes,
		}
	}

	detailsURL := fmt.Sprintf("/apps/%s/environments/%s/deployments/%s", appID, envID, depID)
	status := d.Status()

	jsonMap := map[string]any{
		"id":                 depID,
		"appId":              appID,
		"envId":              envID,
		"envName":            d.Env,
		"displayName":        d.DisplayName,
		"repo":               d.CheckoutRepo,
		"logs":               deploymentLogs,
		"branch":             d.Branch,
		"createdAt":          d.CreatedAt.UnixStr(),
		"stoppedAt":          stoppedAt,
		"stoppedManually":    d.ExitCode.ValueOrZero() == -1,
		"status":             status,
		"snapshot":           d.Snapshot(),
		"error":              d.Error.ValueOrZero(),
		"isAutoDeploy":       d.IsAutoDeploy,
		"isAutoPublish":      d.ShouldPublish,
		"previewUrl":         admin.MustConfig().PreviewURL(d.DisplayName, d.ID.String()),
		"detailsUrl":         detailsURL,
		"apiPathPrefix":      d.APIPathPrefix.ValueOrZero(),
		"statusChecks":       statusChecksLogs,
		"statusChecksPassed": d.StatusChecksPassed,
		"duration":           calculateDuration(d.CreatedAt, d.StoppedAt),
		"published":          []map[string]any{},
		"uploadResult":       uploadResult,
		"isPriority":         d.IsPriority,
		"commit": map[string]any{
			"sha":     d.Commit.ID.Ptr(),
			"author":  d.Commit.Author.Ptr(),
			"message": d.Commit.Message.Ptr(),
		},
	}

	if d.Published != nil {
		for _, p := range d.Published {
			jsonMap["published"] = append(jsonMap["published"].([]map[string]any), map[string]any{
				"envId":      p.EnvID.String(),
				"percentage": p.Percentage,
			})
		}
	}

	if !withLogs {
		jsonMap["logs"] = nil
		jsonMap["statusChecks"] = nil
	}

	// On failure, surface the reason inline so a caller can triage without a
	// second fetch, plus a link to the full logs for the rest of the detail.
	if status == "failed" {
		if summary := d.FailureSummary(); summary != "" {
			jsonMap["failureSummary"] = summary
		}

		jsonMap["logsUrl"] = dashboardURL(detailsURL)
	}

	return jsonMap
}

// dashboardURL turns a relative dashboard path into a fully-qualified URL using
// the instance's configured app domain, falling back to the relative path when
// no domain is configured.
func dashboardURL(path string) string {
	if dc := admin.MustConfig().DomainConfig; dc != nil && dc.App != "" {
		return strings.TrimRight(dc.App, "/") + path
	}

	return path
}

func calculateDuration(createdAt, stoppedAt utils.Unix) int64 {
	if stoppedAt.IsZero() {
		return 0
	}

	return stoppedAt.Unix() - createdAt.Unix()
}
