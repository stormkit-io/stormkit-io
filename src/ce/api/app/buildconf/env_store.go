package buildconf

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var tableEnvs = "apps_build_conf"

// Store represents a store for the deployments and deployment logs.
type Store struct {
	*database.Store

	markAsDeletedTmpl *template.Template
}

// NewStore returns a store instance.
func NewStore() *Store {
	markAsDeletedTmpl, err := template.New("markEnvAsDeleted").Parse(stmt.markAsDeleted)

	if err != nil {
		panic(err)
	}

	return &Store{
		Store:             database.NewStore(),
		markAsDeletedTmpl: markAsDeletedTmpl,
	}
}

// Insert inserts a new configuration for the app.
// The tx parameter is required for app_setup method, as it creates
// the build configuration in a transaction.
func (s *Store) Insert(ctx context.Context, c *Env, txs ...*sql.Tx) error {
	var statement *sql.Stmt
	var err error

	// Either use the provided one, or create a new transaction.
	if len(txs) > 0 {
		statement, err = txs[0].PrepareContext(ctx, stmt.insertConfig)
	} else {
		statement, err = s.Prepare(ctx, stmt.insertConfig)
	}

	if err != nil {
		return err
	}

	// First create the environment in the database.
	data, err := json.Marshal(c.Data)

	if err != nil {
		return err
	}

	err = statement.QueryRowContext(ctx,
		c.AppID, c.Name, c.Branch, data,
		c.AutoPublish, c.AutoDeploy, c.AutoDeployBranches,
		c.AutoDeployCommits,
	).Scan(&c.ID)

	if err != nil {
		return err
	}

	return err
}

// Update updates the given configuration.
func (s *Store) Update(ctx context.Context, c *Env) error {
	data, err := json.Marshal(c.Data)

	if err != nil {
		return err
	}

	_, err = s.Exec(
		ctx,
		stmt.updateConfig,
		c.Env,
		c.Branch,
		data,
		c.AutoPublish,
		c.AutoDeploy,
		c.AutoDeployBranches,
		c.AutoDeployCommits,
		c.ID,
	)

	return err
}

// ListEnvironments list all the configs for the given application id.
func (s *Store) ListEnvironments(ctx context.Context, appID types.ID) (cnfs []*Env, err error) {
	rows, err := s.Query(ctx, stmt.selectByAppID, appID)

	if rows != nil {
		for rows.Next() {
			if cnfs == nil {
				cnfs = []*Env{}
			}

			cnf := &Env{
				Data: &BuildConf{},
			}

			var buildConf []byte
			var publishedInfo []byte

			err := rows.Scan(
				&cnf.ID, &cnf.Name, &cnf.AppID, &buildConf, &cnf.AutoPublish, &cnf.Branch,
				&cnf.AutoDeploy, &cnf.AutoDeployBranches, &cnf.AutoDeployCommits,
				&cnf.DeployedAt, &cnf.LastDeployID, &cnf.LastDeployExitCode, &publishedInfo,
			)

			if err != nil {
				return nil, err
			}

			if buildConf != nil {
				if err := json.Unmarshal(buildConf, cnf.Data); err != nil {
					return nil, err
				}

				if cnf.Data.Cmd != "" {
					cnf.Data.BuildCmd = cnf.Data.Cmd
				}
			}

			if publishedInfo != nil {
				if err := json.Unmarshal(publishedInfo, &cnf.Published); err != nil {
					return nil, err
				}
			}

			cnfs = append(cnfs, cnf)
		}
	}

	return cnfs, err
}

func (s *Store) selectEnvironment(ctx context.Context, query string, appID types.ID, param any) (*Env, error) {
	var buildConf []byte

	cnf := &Env{
		Data: &BuildConf{},
	}

	params := []any{param}

	if appID != 0 {
		params = []any{appID, param}
	}

	row, err := s.QueryRow(ctx, query, params...)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&cnf.ID, &cnf.Name, &cnf.AppID, &buildConf,
		&cnf.AutoPublish, &cnf.Branch, &cnf.AutoDeploy,
		&cnf.AutoDeployBranches, &cnf.AutoDeployCommits,
		&cnf.UpdatedAt, &cnf.SchemaConf,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	// TODO: Remove this once we're done renaming this field.
	cnf.Env = cnf.Name

	if buildConf != nil {
		if err := json.Unmarshal(buildConf, cnf.Data); err != nil {
			return nil, err
		}

		// alias
		if cnf.Data.Cmd != "" {
			cnf.Data.BuildCmd = cnf.Data.Cmd
		}
	}

	return cnf, err
}

// Environment returns a configuration by its environment name.
func (s *Store) Environment(ctx context.Context, appID types.ID, env string) (*Env, error) {
	return s.selectEnvironment(ctx, stmt.selectByAppIDAndEnv, appID, env)
}

// EnvironmentByID returns a configuration by its environment id.
func (s *Store) EnvironmentByID(ctx context.Context, envID types.ID) (*Env, error) {
	return s.selectEnvironment(ctx, stmt.selectByEnvID, 0, envID)
}

// MarkAsDeleted marks an environment as deleted and updates related
// tables (such as unverifying domains). It does not remove the environment
// completely from the database.
func (s *Store) MarkAsDeleted(ctx context.Context, envID types.ID) (bool, error) {
	var wr bytes.Buffer

	data := map[string]any{
		"domainsTableName": tableDomains,
		"tableName":        tableEnvs,
	}

	if err := s.markAsDeletedTmpl.Execute(&wr, data); err != nil {
		return false, err
	}

	result, err := s.Exec(ctx, wr.String(), envID)

	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()

	if err != nil {
		return false, err
	}

	return rows > 0, nil
}

// IsMember checks if the user has access to the environment.
func (s *Store) IsMember(ctx context.Context, envID, userID types.ID) bool {
	var count int

	row, err := s.QueryRow(ctx, stmt.isMember, envID, userID)

	if err != nil {
		return false
	}

	err = row.Scan(&count)

	if err != nil {
		return false
	}

	return count > 0
}

// SaveSchemaConf saves the schema configuration for the given environment.
func (s *Store) SaveSchemaConf(ctx context.Context, envID types.ID, conf *SchemaConf) error {
	_, err := s.Exec(ctx, stmt.saveSchemaConf, conf, envID)
	return err
}
