package apikey

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

var tableAPIKeys = "api_keys"

var stmt = struct {
	addAPIKey     string
	removeAPIKey  string
	selectAPIKeys string
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
		SELECT
			key_id,
			COALESCE(app_id, 0),
			COALESCE(env_id, 0),
			COALESCE(team_id, 0),
			COALESCE(user_id, 0),
			key_name,
			key_value,
			key_scope
		FROM
			api_keys
		WHERE
			{{ .where }}
		LIMIT
			{{ .limit }};
	`,
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
		return nil, err
	}

	var qb strings.Builder

	data := map[string]any{
		"limit": 25,
	}

	switch scope {
	case SCOPE_TEAM:
		data["where"] = "team_id = $1"
	case SCOPE_USER:
		data["where"] = "user_id = $1"
	default:
		data["where"] = "env_id = $1"
	}

	if err := tmpl.Execute(&qb, data); err != nil {
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

type apiKeyFilters struct {
	Token string
	ID    types.ID
}

func (s *Store) apiKey(ctx context.Context, filters apiKeyFilters) (*Token, error) {
	var qb strings.Builder
	tmpl, err := template.New("select_api_keys").Parse(stmt.selectAPIKeys)

	if err != nil {
		return nil, err
	}

	params := []any{}
	data := map[string]any{
		"limit": 1,
	}

	if filters.Token != "" {
		data["where"] = "key_value = $1 OR key_value = $2"

		// If the token does not have the "SK_" prefix, it's not a valid token and
		// we can return early without hitting the database.
		if !strings.HasPrefix(filters.Token, "SK_") {
			return nil, nil
		}

		params = append(params, utils.SHA256Hash([]byte(filters.Token)), filters.Token)
	}

	if filters.ID != 0 {
		data["where"] = "key_id = $1"
		params = append(params, filters.ID)
	}

	if filters.ID == 0 && filters.Token == "" {
		return nil, nil
	}

	if err := tmpl.Execute(&qb, data); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, qb.String(), params...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var key *Token

	for rows.Next() {
		key = &Token{}

		err := rows.Scan(
			&key.ID, &key.AppID, &key.EnvID, &key.TeamID,
			&key.UserID, &key.Name, &key.Value, &key.Scope,
		)

		if err != nil {
			return nil, err
		}
	}

	return key, nil
}

// APIKey looks up an API key by its token value. Queries are backwards compatible
// and support both raw tokens and SHA-256 hashes. If the token does not have the
// "SK_" prefix (the only valid Stormkit token format), it returns empty.
func (s *Store) APIKey(ctx context.Context, token string) (*Token, error) {
	return s.apiKey(ctx, apiKeyFilters{Token: token})
}

// APIKeys returns the API keys associated with this app.
func (s *Store) APIKeyByID(ctx context.Context, keyID types.ID) (*Token, error) {
	return s.apiKey(ctx, apiKeyFilters{ID: keyID})
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

	// Hash the raw token before storing. key.Value is intentionally left
	// unchanged so the caller can return it to the user exactly once.
	row, err := s.QueryRow(
		ctx,
		stmt.addAPIKey,
		key.Name,
		utils.SHA256Hash([]byte((key.Value))),
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
