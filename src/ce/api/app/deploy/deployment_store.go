package deploy

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"text/template"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// Store represents a store for the deployments and deployment logs.
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{database.NewStore()}
}

func (s *Store) ManifestByDeploymentID(ctx context.Context, deploymentID, appID types.ID) (*Deployment, error) {
	d := &Deployment{}
	row, err := s.QueryRow(ctx, stmt.selectBuildManifest, deploymentID, appID)

	if err != nil {
		return nil, err
	}

	err = row.Scan(&d.BuildManifest)
	return d, err
}

type ConfigSnapshot struct {
	BuildConfig *buildconf.BuildConf `json:"build"`
	EnvName     string               `json:"env"`
	EnvID       string               `json:"envId"`
}

// DeploymentByID returns a deployment (alias for MyDeployment with DeploymentID filter).
func (s *Store) DeploymentByID(ctx context.Context, id types.ID) (*Deployment, error) {
	return s.MyDeployment(ctx, &DeploymentsQueryFilters{
		DeploymentID: id,
	})
}

// DeploymentsQueryFilters defines the filters that are
// accepted for the Deployments query.
type DeploymentsQueryFilters struct {
	AppID          types.ID `json:"-"`
	UserID         types.ID `json:"-"`
	TeamID         types.ID `json:"teamId,string"`
	EnvID          types.ID `json:"envId,string"`
	DeploymentID   types.ID `json:"deploymentId,string"`
	AppDisplayName string   `json:"-"`

	// Limit specifies the number of deployments that can be
	// returned by a single query. The maximum is 50.
	Limit int `json:"limit"`

	// From is used for pagination. The deployments are sorted
	// by date descending and this variable specifies from which
	// row to start the query.
	From int `json:"from"`

	// Branch filters the list by branch name.
	Branch string `json:"branch"`

	// Status is a boolean value. If true, only successful deployments
	// will be returned. If false only failed deployments.
	Status *bool `json:"status"`

	// Published is false by default. If true, only currently b
	Published *bool `json:"published"`

	IncludeLogs *bool `json:"-"`
}

func (s *Store) prepareSelectDeploymentsQuery(data map[string]any) (string, error) {
	tmpl, err := template.New("deployments").Parse(stmt.selectDeploymentsV2)

	if err != nil {
		slog.Errorf("error while preparing deployments query: %s", err.Error())
		return "", err
	}

	var wr bytes.Buffer
	err = tmpl.Execute(&wr, map[string]any{
		"where":  data["where"],
		"limit":  data["limit"],
		"offset": data["offset"],
		"logs":   data["logs"],
		"joins":  data["joins"],
	})

	if err != nil {
		slog.Errorf("error while executing deployments query: %s", err.Error())
		return "", err
	}

	return wr.String(), nil
}

func (s *Store) scanRows(rows *sql.Rows, err error) ([]*Deployment, error) {
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	deployments := []*Deployment{}

	for rows.Next() {
		d := &Deployment{}

		err := rows.Scan(
			&d.ID, &d.AppID, &d.EnvID, &d.Env,
			&d.Branch, &d.CreatedAt, &d.StoppedAt,
			&d.ExitCode, &d.ConfigCopy,
			&d.Commit.ID, &d.Commit.Author, &d.Commit.Message,
			&d.GithubRunID, &d.Error, &d.IsAutoDeploy,
			&d.ShouldPublish, &d.PullRequestNumber, &d.BuildManifest,
			&d.APIPathPrefix, &d.IsImmutable, &d.UploadResult, &d.MigrationsPath,
			&d.StatusChecksPassed, &d.StatusChecks, &d.Logs,
			&d.DisplayName, &d.CheckoutRepo, &d.Published,
		)

		if err != nil {
			slog.Errorf("error while scanning deployment: %v", err)
			return nil, err
		}

		deployments = append(deployments, d)
	}

	return deployments, nil
}

// MyDeployment returns a deployment based on the given filters.
func (s *Store) MyDeployment(ctx context.Context, filters *DeploymentsQueryFilters) (*Deployment, error) {
	ds, err := s.MyDeployments(ctx, filters)

	if len(ds) == 1 {
		return ds[0], nil
	}

	return nil, err
}

// MyDeployments returns deployments based on the given filters.
func (s *Store) MyDeployments(ctx context.Context, filters *DeploymentsQueryFilters) ([]*Deployment, error) {
	params := []any{}
	where := []string{}
	joins := []string{}
	joinTeams := "LEFT JOIN teams t ON t.team_id = a.team_id"
	joinTeamMembers := "LEFT JOIN team_members tm ON t.team_id = tm.team_id"

	if filters.UserID != 0 {
		params = append(params, filters.UserID)
		where = append(where, "tm.user_id = $1")
		joins = append(joins, joinTeams, joinTeamMembers)
	}

	if filters.EnvID != 0 {
		params = append(params, filters.EnvID)
		where = append(where, fmt.Sprintf("d.env_id = $%d", len(params)))
	}

	if filters.TeamID != 0 {
		params = append(params, filters.TeamID)
		where = append(where, fmt.Sprintf("tm.team_id = $%d", len(params)))

		// Make sure we don't join the same table twice (also included when UserID != 0)
		if filters.UserID == 0 {
			joins = append(joins, joinTeams, joinTeamMembers)
		}
	}

	if filters.DeploymentID != 0 {
		params = append(params, filters.DeploymentID)
		where = append(where, fmt.Sprintf("d.deployment_id = $%d", len(params)))
	}

	if filters.Published != nil {
		where = append(where, "dp.percentage_released > 0")
		joins = append(joins, "LEFT JOIN deployments_published dp ON dp.deployment_id = d.deployment_id")
	}

	data := map[string]any{
		"where": strings.Join(where, " AND "),
		"joins": strings.Join(joins, " "),
	}

	if filters.IncludeLogs != nil && *filters.IncludeLogs {
		data["logs"] = true
	}

	query, err := s.prepareSelectDeploymentsQuery(data)

	if err != nil {
		return nil, err
	}

	return s.scanRows(s.Query(ctx, query, params...))
}

// InsertDeployment inserts a new deployment.
func (s *Store) InsertDeployment(ctx context.Context, d *Deployment) error {
	var webhookEvent any

	branch := null.NewString(d.Branch, d.Branch != "")
	repo := null.NewString(d.CheckoutRepo, d.CheckoutRepo != "")

	if d.WebhookEvent != nil {
		webhookEvent, _ = json.Marshal(d.WebhookEvent)
	} else {
		webhookEvent = nil
	}

	params := []any{
		d.AppID, d.ConfigCopy, branch, d.Env, d.EnvID,
		d.IsAutoDeploy, d.PullRequestNumber,
		d.Commit.ID, d.IsFork, d.ShouldPublish, repo,
		d.APIPathPrefix, webhookEvent, d.Commit.Author,
		d.MigrationsPath,
	}

	row, err := s.QueryRow(ctx, stmt.insertDeployment, params...)

	if err != nil {
		return err
	}

	return row.Scan(&d.ID, &d.CreatedAt)
}

// UpdateLogs will update the deployment logs for the given deployment id.
func (s *Store) UpdateLogs(ctx context.Context, did types.ID, logs string) error {
	if logs == "" {
		return nil
	}

	_, err := s.Exec(ctx, stmt.updateLogs, logs, did)
	return err
}

func (s *Store) UpdateStatusChecks(ctx context.Context, did types.ID, logs string) error {
	if logs == "" {
		return nil
	}

	_, err := s.Exec(ctx, stmt.updateStatusChecks, logs, did)
	return err
}

// UpdateExitCode updates the exit code of the deployment if it's not updated already.
func (s *Store) UpdateExitCode(ctx context.Context, deploymentID types.ID, exitCode int) error {
	_, err := s.Exec(ctx, stmt.updateExitCode, exitCode, deploymentID)
	return err
}

// UpdateCommitID updates the commit ID of the deployment.
func (s *Store) UpdateCommitInfo(ctx context.Context, deploymentID types.ID, info CommitInfo) error {
	_, err := s.Exec(ctx, stmt.updateCommitInfo, info.ID, info.Author, info.Message, deploymentID)
	return err
}

// MarkDeploymentsAsDeleted marks the deployment as deleted.
// This function also DELETEs all the logs to free up some space.
func (s *Store) MarkDeploymentsAsDeleted(ctx context.Context, ids []types.ID) error {
	_, err := s.Exec(ctx, stmt.markDeploymentsAsDeleted, pq.Array(ids))
	return err
}

type DeploymentStats struct {
	ActiveDeployments             int `json:"activeDeployments"`
	NumberOfDeploymentsThisMonth  int `json:"numberOfDeploymentsThisMonth"`
	RemainingDeploymentsThisMonth int `json:"remainingDeploymentsThisMonth"`
}

// IsDeploymentAlreadyBuilt checks if the deployment has been already built or not.
func (s *Store) IsDeploymentAlreadyBuilt(ctx context.Context, commitID string) (bool, error) {
	var count int

	row, err := s.QueryRow(ctx, stmt.isDeploymentAlreadyBuilt, commitID)

	if err != nil {
		return false, err
	}

	err = row.Scan(&count)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return count > 0, nil
}

// StopDeployment stops a deployment by updating the stopped_at field
// and setting the exit_code to -1.
func (s *Store) StopDeployment(ctx context.Context, deploymentID types.ID) error {
	_, err := s.Exec(ctx, stmt.stopDeployment, deploymentID)
	return err
}

// StopStatusChecks sets the status_checks_passed column to false.
func (s *Store) StopStatusChecks(ctx context.Context, deploymentID types.ID) error {
	_, err := s.Exec(ctx, stmt.stopStatusChecks, deploymentID)
	return err
}

// IsDeploymentStopped checks whether a deployment is already stopped or not.
func (s *Store) IsDeploymentStopped(ctx context.Context, deploymentID types.ID) (bool, error) {
	var exitCode null.Int

	row, err := s.QueryRow(ctx, stmt.selectExitCode, deploymentID)

	if err != nil {
		return false, err
	}

	err = row.Scan(&exitCode)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return exitCode.ValueOrZero() == int64(ExitCodeStopped), nil
}

// UpdateGithubRunID updates the run id associated with the deployment. This value
// is the $GITHUB_RUN_ID available in github actions.
func (s *Store) UpdateGithubRunID(ctx context.Context, deploymentID types.ID, runID types.ID) error {
	_, err := s.Exec(ctx, stmt.updateGithubRunID, runID, deploymentID)
	return err
}

func (s *Store) UpdateDeploymentResult(ctx context.Context, d *Deployment, result integrations.UploadResult) error {
	// these values are int32 in db but int64 in code
	// sometimes these value is more than int32 and
	// it causes error, this is workaround for that
	if result.Server.BytesUploaded > math.MaxInt32 {
		result.Server.BytesUploaded = math.MaxInt32
	}

	if result.Client.BytesUploaded > math.MaxInt32 {
		result.Client.BytesUploaded = math.MaxInt32
	}

	d.UploadResult = &UploadResult{
		ClientBytes:        result.Client.BytesUploaded,
		ClientLocation:     result.Client.Location,
		ServerBytes:        result.Server.BytesUploaded,
		ServerLocation:     result.Server.Location,
		ServerlessBytes:    result.API.BytesUploaded,
		ServerlessLocation: result.API.Location,
		// MigrationsBytes:    result.Migrations.BytesUploaded,
		// MigrationsLocation: result.Migrations.Location,
	}

	if d.Error.ValueOrZero() != "" {
		d.ExitCode = null.NewInt(1, true)
	}

	row, err := s.QueryRow(
		ctx,
		stmt.updateDeploymentResult,
		d.UploadResult,
		d.Error,
		d.ExitCode,
		d.BuildManifest,
		d.ID,
	)

	if err != nil {
		slog.Errorf("error while updating upload result: %v", err)
		return err
	}

	return row.Scan(&d.StoppedAt)
}

// MarkArtifactsAsDeleted marks artifacts as deleted.
func (s *Store) MarkArtifactsAsDeleted(ctx context.Context, ids []types.ID) error {
	_, err := s.Exec(ctx, stmt.markArtifactsAsDeleted, pq.Array(ids))
	return err
}

// LockDeployment locks a deployment so that it becomes immutable and updates the status checks result.
func (s *Store) LockDeployment(ctx context.Context, id types.ID, statusChecksPassed null.Bool) error {
	_, err := s.Exec(ctx, stmt.lockDeployment, statusChecksPassed, id)

	if err != nil {
		return err
	}

	if config.IsStormkitCloud() {
		if _, err := s.Exec(ctx, stmt.updateUserMetrics, id); err != nil {
			return err
		}
	}

	return nil
}

// Restart resets the output of the deployment and prepares the deployment for a restart.
func (s *Store) Restart(ctx context.Context, d *Deployment) error {
	d.IsRestart = true
	d.ExitCode = null.NewInt(0, false)
	d.IsImmutable = null.NewBool(false, false)
	_, err := s.Exec(ctx, stmt.restartDeployment, d.ID)
	return err
}

// Publish publishes the given deployments.
func (s *Store) Publish(ctx context.Context, settings ...*PublishSettings) error {
	if len(settings) == 0 {
		return nil
	}

	checks := map[types.ID]types.ID{}
	envIDs := []types.ID{}

	tmpl, err := template.New("publish").
		Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
		Parse(stmt.publish)

	if err != nil {
		slog.Errorf("error parsing publish query template: %v", err)
		return err
	}

	params := []any{}

	for _, record := range settings {
		if record.Percentage <= 0 {
			continue
		}

		params = append(params, record.EnvID, record.DeploymentID, record.Percentage)
		envIDs = append(envIDs, record.EnvID)
		checks[record.DeploymentID] = record.EnvID
	}

	var qb strings.Builder

	data := map[string]any{
		"envIDsParam": fmt.Sprintf("$%d", len(params)+1),
		"records":     settings,
	}

	if err = tmpl.Execute(&qb, data); err != nil {
		slog.Errorf("Error executing query template: %v", err)
		return err
	}

	params = append(params, pq.Array(envIDs))
	_, err = s.Exec(ctx, qb.String(), params...)
	return err
}
