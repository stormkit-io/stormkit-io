package apikey

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"gopkg.in/guregu/null.v3"
)

var tableAPIKeys = "api_keys"
var apiKeyColumns = `
	key_id,
	COALESCE(app_id, 0),
	COALESCE(env_id, 0),
	COALESCE(team_id, 0),
	COALESCE(user_id, 0),
	key_name,
	key_value,
	key_scope
`

var stmt = struct {
	addAPIKey        string
	removeAPIKey     string
	selectAPIKey     string
	selectAPIKeyByID string
	selectAPIKeys    string
}{
	addAPIKey: fmt.Sprintf(`
		INSERT INTO %s (key_name, key_value, app_id, env_id, user_id, team_id, key_scope)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING key_id;
	`, tableAPIKeys),

	removeAPIKey: fmt.Sprintf(`
		DELETE FROM %s WHERE key_id = $1;
	`, tableAPIKeys),

	selectAPIKeys: `
		SELECT {{ .columns }} FROM api_keys WHERE {{ .where }} = $1 LIMIT 25;
	`,

	selectAPIKey: fmt.Sprintf(`
		SELECT %s FROM %s WHERE key_value = $1;
	`, apiKeyColumns, tableAPIKeys),

	selectAPIKeyByID: fmt.Sprintf(`
		SELECT %s FROM %s WHERE key_id = $1;
	`, apiKeyColumns, tableAPIKeys),
}

// Store is the store to handle app logic
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{database.NewStore()}
}

// APIKeys returns the API keys associated with this app.
func (s *Store) APIKeys(ctx context.Context, id types.ID, scope string) ([]Token, error) {
	tmpl, err := template.New("select_api_keys").Parse(stmt.selectAPIKeys)

	if err != nil {
		slog.Errorf("error while parsing select_api_keys: %s", err.Error())
		return nil, err
	}

	var qb strings.Builder

	data := map[string]any{
		"columns": apiKeyColumns,
	}

	switch scope {
	case SCOPE_TEAM:
		data["where"] = "team_id"
	case SCOPE_USER:
		data["where"] = "user_id"
	default:
		data["where"] = "env_id"
	}

	if err := tmpl.Execute(&qb, data); err != nil {
		slog.Errorf("error executing query template: %s", err.Error())
		return nil, err
	}

	rows, err := s.Query(ctx, qb.String(), id)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	keys := []Token{}

	defer rows.Close()

	for rows.Next() {
		key := Token{}
		err := rows.Scan(
			&key.ID, &key.AppID, &key.EnvID, &key.TeamID,
			&key.UserID, &key.Name, &key.Value, &key.Scope,
		)

		if err != nil {
			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// APIKey returns the API keys associated with this app.
func (s *Store) APIKey(ctx context.Context, token string) (*Token, error) {
	key := &Token{}
	row, err := s.QueryRow(ctx, stmt.selectAPIKey, token)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&key.ID, &key.AppID, &key.EnvID, &key.TeamID,
		&key.UserID, &key.Name, &key.Value, &key.Scope,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return key, err
}

// APIKeys returns the API keys associated with this app.
func (s *Store) APIKeyByID(ctx context.Context, keyID types.ID) (*Token, error) {
	key := &Token{}
	row, err := s.QueryRow(ctx, stmt.selectAPIKeyByID, keyID)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&key.ID, &key.AppID, &key.EnvID, &key.TeamID,
		&key.UserID, &key.Name, &key.Value, &key.Scope,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return key, err
}

// AddAPIKey sets the API token for the given app id.
func (s *Store) AddAPIKey(ctx context.Context, key *Token) error {
	if !IsScopeValid(key.Scope) {
		return fmt.Errorf("scope is invalid")
	}

	usrID := null.NewInt(int64(key.UserID), key.UserID != 0)
	envID := null.NewInt(int64(key.EnvID), key.EnvID != 0)
	appID := null.NewInt(int64(key.AppID), key.AppID != 0)
	teamID := null.NewInt(int64(key.TeamID), key.TeamID != 0)

	row, err := s.QueryRow(
		ctx,
		stmt.addAPIKey,
		key.Name,
		key.Value,
		appID,
		envID,
		usrID,
		teamID,
		key.Scope,
	)

	if err != nil {
		return err
	}

	return row.Scan(&key.ID)
}

func (s *Store) RemoveAPIKey(ctx context.Context, keyID types.ID) error {
	_, err := s.Exec(ctx, stmt.removeAPIKey, keyID)
	return err
}
