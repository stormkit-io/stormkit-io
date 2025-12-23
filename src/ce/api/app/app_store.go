package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"

	"gopkg.in/guregu/null.v3"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// Store is the store to handle app logic
type Store struct {
	*database.Store
	selectAppTmpl        *template.Template
	markAsDeletedTmpl    *template.Template
	deployCandidatesTmpl *template.Template
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{
		Store:                database.NewStore(),
		markAsDeletedTmpl:    template.Must(template.New("markAppsAsDeleted").Parse(stmt.markAsDeleted)),
		deployCandidatesTmpl: template.Must(template.New("deployCandidates").Parse(stmt.selectDeployCandidates)),
		selectAppTmpl:        template.Must(template.New("selectApp").Parse(stmt.selectApp)),
	}
}

// InsertApp inserts an app into the database.
func (s *Store) InsertApp(ctx context.Context, a *App) (*App, error) {
	tx, err := s.Conn.Begin()

	errFn := func(err error) (*App, error) {
		_ = tx.Rollback()
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	repo := null.NewString(a.Repo, a.Repo != "")

	params := []any{
		repo,
		a.UserID,
		a.ClientID,
		a.ClientSecret,
		a.privateKey,
		a.DisplayName,
		a.IsDefault,
		a.Runtime,
		a.AutoDeploy,
		a.TeamID,
	}

	stmt, err := tx.PrepareContext(ctx, stmt.insertApp)

	if err != nil {
		return errFn(err)
	}

	for {
		if err = stmt.QueryRowContext(ctx, params...).Scan(&a.ID, &a.CreatedAt); err != nil {
			// The display name is already in use, try another one.
			if database.IsDuplicate(err) {
				a.DisplayName = GenerateDisplayName()
				continue
			} else {
				return errFn(err)
			}
		}

		break
	}

	if err = appSetup(ctx, a, tx); err != nil {
		return errFn(err)
	}

	if err := tx.Commit(); err != nil {
		return errFn(err)
	}

	return a, nil
}

// UpdateApp updates the given app.
func (s *Store) UpdateApp(ctx context.Context, a *App) error {
	repo := null.NewString(a.Repo, a.Repo != "")

	_, err := s.Exec(
		ctx,
		stmt.updateApp,
		repo,
		a.DisplayName,
		a.AutoDeploy.Ptr(),
		a.DefaultEnv,
		a.Runtime,
		a.ID,
	)

	return err
}

// DeletedApps will return the requested amount of deleted apps.
func (s *Store) DeletedApps(ctx context.Context, olderThanDays, limit int) ([]*App, error) {
	rows, err := s.Query(
		ctx,
		strings.Replace(stmt.deletedApps, "$0", strconv.Itoa(olderThanDays), 1), // Cannot use $0 as a parameter type
		limit,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	defer rows.Close()
	apps := []*App{}

	for rows.Next() {
		a := &App{}
		err := rows.Scan(&a.ID)

		if err != nil {
			slog.Error(err.Error())
			continue
		}

		apps = append(apps, a)
	}

	return apps, rows.Err()
}

// MarkAsDeleted marks an app as deleted and updates related
// tables (such as unverifying domains and marking envs as deleted).
// It does not remove the application completely from the database.
func (s *Store) MarkAsDeleted(ctx context.Context, appID types.ID) (bool, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"domainsTableName": tableDomains,
		"envsTableName":    tableEnvs,
		"tableName":        tableApps,
	}

	if err := s.markAsDeletedTmpl.Execute(&wr, data); err != nil {
		return false, err
	}

	result, err := s.Exec(ctx, wr.String(), appID)

	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()

	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

// MarkArtifactsAsDeleted updates the database row by setting artifacts
// deleted column to true. This way we know which apps we have deleted,
// and which are still programmed to be deleted. From this stage on,
// we should only have historical data in the database (since the artifacts are deleted).
// This data should never be deleted.
func (s *Store) MarkArtifactsAsDeleted(ctx context.Context, a *App) error {
	_, err := s.Exec(ctx, stmt.markArtifactsAsDeleted, a.ID)
	return err
}

func (s *Store) fetchApp(ctx context.Context, data map[string]any, params ...any) (*App, error) {
	app := &App{}
	env := null.NewString("", false)
	rnt := null.NewString("", false)

	var qb strings.Builder

	if err := s.selectAppTmpl.Execute(&qb, data); err != nil {
		slog.Errorf("error executing query template: %s", err.Error())
		return nil, err
	}

	row, err := s.QueryRow(ctx, qb.String(), params...)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&app.ID, &app.Repo,
		&app.UserID, &app.CreatedAt, &app.ClientID,
		&app.ClientSecret, &app.DisplayName,
		&app.AutoDeploy, &env, &app.TeamID, &rnt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	app.DefaultEnv = env.ValueOrZero()
	app.Runtime = rnt.ValueOrZero()

	if app.DefaultEnv == "" {
		app.DefaultEnv = config.AppDefaultEnvironmentName
	}

	if app.Runtime == "" {
		app.Runtime = config.DefaultNodeRuntime
	}

	return app, err
}

// AppByID returns an app by its id.
func (s *Store) AppByID(ctx context.Context, appID types.ID) (*App, error) {
	return s.fetchApp(ctx, map[string]any{
		"join":  "",
		"where": "a.app_id = $1 AND a.deleted_at IS NULL",
	}, appID)
}

// AppByDisplayName returns an app by it's display name.
func (s *Store) AppByDisplayName(ctx context.Context, displayName string) (*App, error) {
	return s.fetchApp(ctx, map[string]any{
		"join":  "",
		"where": "a.display_name = $1 AND a.deleted_at IS NULL",
	}, displayName)
}

// AppByDomainName returns the app that the domain is associated with.
func (s *Store) AppByDomainName(ctx context.Context, domainName string) (*App, error) {
	return s.fetchApp(ctx, map[string]any{
		"join":  "LEFT JOIN domains d ON d.app_id = a.app_id",
		"where": "d.domain_name = $1 AND a.deleted_at IS NULL",
	}, domainName)
}

// AppByEnvID returns an app by the environment id.
func (s *Store) AppByEnvID(ctx context.Context, envID types.ID) (*App, error) {
	return s.fetchApp(ctx, map[string]any{
		"join":  "LEFT JOIN apps_build_conf e ON e.app_id = a.app_id",
		"where": "e.env_id = $1 AND e.deleted_at IS NULL AND a.deleted_at IS NULL",
	}, envID)
}

// DeployCandidate represents an app, with an environment and branch
// to be deployed.
type DeployCandidate struct {
	*MyApp
	Branch             string
	EnvName            string
	EnvID              types.ID
	EnvDefaultBranch   string
	EnvAutoDeploy      bool
	AutoDeployBranches null.String
	AutoDeployCommits  null.String
	ShouldPublish      bool
	BuildConfig        *buildconf.BuildConf
	SchemaConf         *buildconf.SchemaConf
}

// DeployCandidates returns a list of deploy candidates that belongs to apps matching the repo name.
// Further logic needs to be applied to determine which deployments should be deployed.
func (s *Store) DeployCandidates(ctx context.Context, repo string) ([]*DeployCandidate, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"where": "LOWER(a.repo) = $1",
	}

	if err := s.deployCandidatesTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, wr.String(), strings.ToLower(repo))

	if err != nil {
		return nil, err
	}

	apps := []*DeployCandidate{}
	defer rows.Close()

	for rows.Next() {
		var buildConf []byte
		var defaultEnv null.String

		ma := &DeployCandidate{
			MyApp:       &MyApp{App: &App{}},
			BuildConfig: &buildconf.BuildConf{},
		}

		err := rows.Scan(
			&ma.ID, &ma.Repo, &ma.CreatedAt,
			&ma.DisplayName, &defaultEnv,
			&ma.AutoDeploy, &ma.Runtime,
			&ma.UserID, &ma.TeamID, &ma.EnvName, &ma.EnvID,
			&ma.ShouldPublish, &buildConf, &ma.EnvDefaultBranch,
			&ma.AutoDeployBranches, &ma.AutoDeployCommits, &ma.EnvAutoDeploy,
			&ma.SchemaConf,
		)

		if defaultEnv.ValueOrZero() == "" {
			defaultEnv = null.NewString(config.AppDefaultEnvironmentName, true)
		}

		ma.DefaultEnv = defaultEnv.ValueOrZero()

		if buildConf != nil {
			if err := json.Unmarshal(buildConf, ma.BuildConfig); err != nil {
				return nil, err
			}
		}

		if err != nil {
			slog.Errorf("error while fetching apps for auto deploy: %v", err)
			return nil, err
		}

		apps = append(apps, ma)
	}

	return apps, rows.Err()
}

// privateKey returns the application private key.
func (s *Store) privateKey(ctx context.Context, a *App) error {
	row, err := s.QueryRow(ctx, stmt.selectAppPrivateKey, a.ID)

	if err != nil {
		return err
	}

	return row.Scan(&a.privateKey)
}

// savePrivateKey updates the database row with the new private key.
func (s *Store) savePrivateKey(ctx context.Context, a *App) error {
	_, err := s.Exec(ctx, stmt.updatePrivateKey, a.PrivateKey, a.ID)
	return err
}

// Apps returns a list of apps that belong to the user.
func (s *Store) Apps(ctx context.Context, teamID types.ID, from, limit int, filter ...string) ([]*MyApp, error) {
	filterCondition := ""
	params := []any{teamID, limit, from}

	if len(filter) > 0 {
		filterCondition = `AND (
			LOWER(a.repo) LIKE '%' || $4 || '%' OR
			LOWER(a.display_name) LIKE '%' || $5 || '%'
		)`
		params = append(params, filter[0], filter[0])
	}

	query := strings.Replace(stmt.selectApps, ":filter", filterCondition, 1)
	rows, err := s.Query(ctx, query, params...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	apps := []*MyApp{}

	for rows.Next() {
		env := null.NewString("", false)
		rnt := null.NewString("", false)
		ma := &MyApp{App: &App{}}
		err := rows.Scan(
			&ma.ID, &ma.Repo, &ma.CreatedAt,
			&ma.UserID, &ma.DisplayName, &ma.AutoDeploy,
			&env, &ma.TeamID, &rnt,
		)

		if err != nil {
			slog.Error(err.Error())
			continue
		}

		ma.DefaultEnv = env.ValueOrZero()
		ma.Runtime = rnt.ValueOrZero()

		if ma.DefaultEnv == "" {
			ma.DefaultEnv = config.AppDefaultEnvironmentName
		}

		if ma.Runtime == "" {
			ma.Runtime = config.DefaultNodeRuntime
		}

		apps = append(apps, ma)
	}

	return apps, rows.Err()
}

// Settings returns the application settings, required to be displayed in the app settings page.
func (s *Store) Settings(ctx context.Context, aid types.ID) (*Settings, error) {
	set := &Settings{}
	rnt := null.NewString("", false)
	row, err := s.QueryRow(ctx, stmt.selectAppSettings, aid, aid)

	if err != nil {
		return nil, err
	}

	if err = row.Scan(&set.DeployTrigger, pq.Array(&set.Envs), &rnt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	set.Runtime = rnt.ValueOrZero()

	if set.Runtime == "" {
		set.Runtime = config.DefaultNodeRuntime
	}

	return set, nil
}

// UpdateDeployTrigger updates the deploy trigger hash.
func (s *Store) UpdateDeployTrigger(ctx context.Context, aid types.ID, hash string) error {
	_, err := s.Exec(ctx, stmt.updateDeployTrigger, hash, aid)
	return err
}

func (s *Store) DeleteDeployTrigger(ctx context.Context, appId types.ID) error {
	_, err := s.Exec(ctx, stmt.removeDeployTrigger, nil, appId)
	return err
}

// OutboundWebhook returns an item by it's ID.
func (s *Store) OutboundWebhook(ctx context.Context, appID, whID types.ID) *OutboundWebhook {
	var headers []byte
	wh := &OutboundWebhook{}

	row, err := s.QueryRow(ctx, stmt.selectOutboundWebhook, appID, whID)

	if err != nil {
		return nil
	}

	err = row.Scan(
		&headers,
		&wh.RequestPayload,
		&wh.RequestURL,
		&wh.RequestMethod,
		&wh.TriggerWhen,
		&wh.WebhookID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}

		slog.Errorf("failed fetching outbound webhook: %s", err.Error())
		return nil
	}

	if headers != nil {
		if err := json.Unmarshal(headers, &wh.RequestHeaders); err != nil {
			slog.Errorf("failed while unmarshaling outbound webhooks headers: %s", err.Error())
			return nil
		}
	}

	return wh
}

// OutboundWebhooks returns a slice of OutboundWebhook objects which belong
// to the application with the given ID.
func (s *Store) OutboundWebhooks(ctx context.Context, appID types.ID) []OutboundWebhook {
	rows, err := s.Query(ctx, stmt.selectOutboundWebhooks, appID)

	if err != nil {
		slog.Errorf("failed while fetching outbound webhooks: %s", err.Error())
		return nil
	}

	defer rows.Close()

	whs := []OutboundWebhook{}

	for rows.Next() {
		var headers []byte

		wh := OutboundWebhook{}
		params := []interface{}{
			&headers,
			&wh.RequestPayload,
			&wh.RequestURL,
			&wh.RequestMethod,
			&wh.TriggerWhen,
			&wh.WebhookID,
		}

		if err := rows.Scan(params...); err != nil {
			slog.Errorf("failed while scanning outbound webhooks: %s", err.Error())
			return nil
		}

		if headers != nil {
			if err := json.Unmarshal(headers, &wh.RequestHeaders); err != nil {
				slog.Errorf("failed while unmarshaling outbound webhooks headers: %s", err.Error())
				return nil
			}
		}

		whs = append(whs, wh)
	}

	return whs
}

// InsertOutgoingWebhook inserts an outgoing webhook into the database.
func (s *Store) InsertOutboundWebhook(ctx context.Context, appID types.ID, wh OutboundWebhook) error {
	headers, err := json.Marshal(wh.RequestHeaders)

	if err != nil {
		return err
	}

	_, err = s.Exec(
		ctx,
		stmt.insertOutboundWebhook,
		appID,
		headers,
		wh.RequestPayload,
		wh.RequestURL,
		wh.RequestMethod,
		wh.TriggerWhen,
	)

	return err
}

// UpdateOutgoingWebhook updates the given outgoing webhook.
func (s *Store) UpdateOutboundWebhook(ctx context.Context, appID types.ID, wh *OutboundWebhook) error {
	headers, err := json.Marshal(wh.RequestHeaders)

	if err != nil {
		return err
	}

	_, err = s.Exec(
		ctx,
		stmt.updateOutboundWebhook,
		headers,
		wh.RequestPayload,
		wh.RequestURL,
		wh.RequestMethod,
		wh.TriggerWhen,
		appID,
		wh.WebhookID,
	)

	return err
}

// DeleteOutboundWebhook deletes the given webhook for the given app.
func (s *Store) DeleteOutboundWebhook(ctx context.Context, appID, whID types.ID) error {
	_, err := s.Exec(ctx, stmt.deleteOutboundWebhook, appID, whID)
	return err
}
