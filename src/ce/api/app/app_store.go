package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// Store is the store to handle app logic
type Store struct {
	*database.Store
	markAsDeletedTmpl    *template.Template
	deployCandidatesTmpl *template.Template
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{
		Store:                database.NewStore(),
		markAsDeletedTmpl:    template.Must(template.New("markAppsAsDeleted").Parse(stmt.markAsDeleted)),
		deployCandidatesTmpl: template.Must(template.New("deployCandidates").Parse(stmt.selectDeployCandidates)),
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

// AppByID returns an app by its id.
func (s *Store) AppByID(ctx context.Context, appID types.ID) (*App, error) {
	apps, err := s.Apps(ctx, AppsArgs{Limit: 1, AppID: appID})

	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, nil
	}

	return apps[0], nil
}

// AppByDisplayName returns an app by it's display name.
func (s *Store) AppByDisplayName(ctx context.Context, displayName string) (*App, error) {
	apps, err := s.Apps(ctx, AppsArgs{Limit: 1, DisplayName: displayName})

	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, nil
	}

	return apps[0], nil
}

// AppByDomainName returns the app that the domain is associated with.
func (s *Store) AppByDomainName(ctx context.Context, domainName string) (*App, error) {
	apps, err := s.Apps(ctx, AppsArgs{Limit: 1, DomainName: domainName})

	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, nil
	}

	return apps[0], nil
}

// AppByEnvID returns an app by the environment id.
func (s *Store) AppByEnvID(ctx context.Context, envID types.ID) (*App, error) {
	apps, err := s.Apps(ctx, AppsArgs{Limit: 1, EnvID: envID})

	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, nil
	}

	return apps[0], nil
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
	MailerConf         *buildconf.MailerConf
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
		var schemaConf []byte
		var buildConf []byte
		var mailerConf []byte

		ma := &DeployCandidate{
			MyApp:       &MyApp{App: &App{}},
			BuildConfig: &buildconf.BuildConf{},
		}

		err := rows.Scan(
			&ma.ID, &ma.Repo, &ma.CreatedAt,
			&ma.DisplayName, &ma.AutoDeploy, &ma.Runtime,
			&ma.UserID, &ma.TeamID, &ma.EnvName, &ma.EnvID,
			&ma.ShouldPublish, &buildConf, &ma.EnvDefaultBranch,
			&ma.AutoDeployBranches, &ma.AutoDeployCommits, &ma.EnvAutoDeploy,
			&schemaConf, &mailerConf,
		)

		if buildConf != nil {
			if err := json.Unmarshal(buildConf, ma.BuildConfig); err != nil {
				return nil, err
			}
		}

		if schemaConf != nil {
			if err := utils.ByteaScan(schemaConf, &ma.SchemaConf); err != nil {
				return nil, err
			}
		}

		if mailerConf != nil {
			if err := json.Unmarshal(mailerConf, &ma.MailerConf); err != nil {
				return nil, err
			}
		}

		if err != nil {
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

type AppsArgs struct {
	From   int
	Limit  int
	AppID  types.ID
	TeamID types.ID
	EnvID  types.ID
	// Filter performs a case-insensitive substring match across both the repo
	// and display name fields. Use Repo or DisplayName for exact matches.
	Filter      string
	DisplayName string
	Repo        string
	DomainName  string
}

// Apps returns a list of apps that belong to the user.
func (s *Store) Apps(ctx context.Context, args AppsArgs) ([]*App, error) {
	var join string
	where := []string{"a.deleted_at IS NULL"}
	params := []any{}

	if args.EnvID != 0 {
		join = "apps_build_conf envs ON envs.app_id = a.app_id"
		params = append(params, args.EnvID)
		where = append(where, fmt.Sprintf("envs.env_id = $%d AND envs.deleted_at IS NULL", len(params)))
	} else {
		join = "apps_build_conf envs ON envs.env_id = (SELECT MIN(env_id) FROM apps_build_conf abc WHERE app_id = a.app_id AND abc.deleted_at IS NULL)"
	}

	if args.Filter != "" {
		params = append(params, strings.ToLower(args.Filter))
		where = append(where, `(
			LOWER(a.repo) LIKE '%' || `+fmt.Sprintf("$%d", len(params))+` || '%' OR
			LOWER(a.display_name) LIKE '%' || `+fmt.Sprintf("$%d", len(params))+` || '%'
		)`)
	}

	if args.DomainName != "" {
		params = append(params, args.DomainName)
		where = append(where, fmt.Sprintf("a.app_id IN (SELECT d.app_id FROM domains d WHERE d.domain_name = $%d)", len(params)))
	}

	if args.DisplayName != "" {
		params = append(params, args.DisplayName)
		where = append(where, fmt.Sprintf("LOWER(a.display_name) = LOWER($%d)", len(params)))
	}

	if args.Repo != "" {
		params = append(params, args.Repo)
		where = append(where, fmt.Sprintf("LOWER(a.repo) = LOWER($%d)", len(params)))
	}

	if args.AppID != 0 {
		params = append(params, args.AppID)
		where = append(where, fmt.Sprintf("a.app_id = $%d", len(params)))
	}

	if args.TeamID != 0 {
		params = append(params, args.TeamID)
		where = append(where, fmt.Sprintf("a.team_id = $%d", len(params)))
	}

	if args.Limit == 0 {
		args.Limit = 25
	}

	if args.From < 0 {
		args.From = 0
	}

	buf := bytes.Buffer{}

	err := sqlTemplates.selectApps.Execute(&buf, map[string]any{
		"join":   join,
		"where":  strings.Join(where, " AND "),
		"limit":  args.Limit,
		"offset": args.From,
	})

	if err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, buf.String(), params...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	apps := []*App{}

	for rows.Next() {
		ma := &App{}
		err := rows.Scan(
			&ma.ID, &ma.Repo, &ma.CreatedAt,
			&ma.UserID, &ma.DisplayName, &ma.AutoDeploy,
			&ma.TeamID, &ma.DefaultEnvID,
		)

		if err != nil {
			return nil, err
		}

		apps = append(apps, ma)
	}

	return apps, nil
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
