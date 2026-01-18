package skauth

import (
	"context"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var stmt = struct {
	selectOAuthConfig string
	saveOAuthConfig   string
}{
	selectOAuthConfig: `
		SELECT
			provider_id,
			provider_name,
			provider_client_id,
			provider_client_secret,
			provider_redirect_url,
			provider_scopes,
			provider_status
		FROM
			oauth_configs
		WHERE
			env_id = $1 AND
			provider_name = $2;
	`,

	saveOAuthConfig: `
		INSERT INTO oauth_configs (
			provider_name,
			provider_client_id,
			provider_client_secret,
			provider_redirect_url,
			provider_scopes,
			provider_status,
			env_id,
			app_id
		) 
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (
			env_id, provider_name
		)
		DO UPDATE SET
			provider_client_id = EXCLUDED.provider_client_id,
			provider_client_secret = EXCLUDED.provider_client_secret,
			provider_redirect_url = EXCLUDED.provider_redirect_url,
			provider_scopes = EXCLUDED.provider_scopes,
			provider_status = EXCLUDED.provider_status;
	`,
}

type Store struct {
	*database.Store
}

func NewStore() *Store {
	return &Store{
		Store: database.NewStore(),
	}
}

type SaveProviderArgs struct {
	Client Client
	EnvID  types.ID
	AppID  types.ID
	Status bool
}

// SaveProvider saves the OAuth2 provider configuration.
func (s *Store) SaveProvider(ctx context.Context, args SaveProviderArgs) error {
	config := args.Client.Config()

	_, err := s.Exec(
		ctx, stmt.saveOAuthConfig,
		args.Client.Name(),
		config.ClientID,
		utils.EncryptToString(config.ClientSecret),
		config.RedirectURL,
		pq.Array(config.Scopes),
		args.Status,
		args.EnvID,
		args.AppID,
	)

	return err
}

// Provider retrieves the OAuth2 client for a given environment and provider name.
func (s *Store) Provider(ctx context.Context, envID types.ID, providerName string) (*Provider, error) {
	row, err := s.QueryRow(ctx, stmt.selectOAuthConfig, envID, providerName)

	if err != nil || row == nil {
		return nil, err
	}

	provider := &Provider{}

	err = row.Scan(
		&provider.ID,
		&provider.Name,
		&provider.ClientID,
		&provider.ClientSecret,
		&provider.RedirectURL,
		pq.Array(&provider.Scopes),
		&provider.Status,
	)

	if err != nil {
		return nil, err
	}

	provider.ClientSecret = utils.DecryptToString(provider.ClientSecret)

	return provider, nil
}
