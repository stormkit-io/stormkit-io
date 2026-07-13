package skauth

import (
	"bytes"
	"context"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var sqlTemplates = struct {
	selectOAuthConfig *template.Template
}{
	selectOAuthConfig: template.Must(template.New("selectOAuthConfig").Parse(`
		SELECT
			provider_id,
			provider_name,
			provider_data,
			provider_status
		FROM
			oauth_configs
		WHERE
			{{ .where }};
	`)),
}

var stmt = struct {
	saveOAuthConfig string
}{
	saveOAuthConfig: `
		INSERT INTO oauth_configs (
			provider_name,
			provider_data,
			provider_status,
			env_id,
			app_id
		) 
		VALUES (
			$1, $2, $3, $4, $5
		)
		ON CONFLICT (
			env_id, provider_name
		)
		DO UPDATE SET
			provider_data = EXCLUDED.provider_data,
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
	Provider *Provider
	EnvID    types.ID
	AppID    types.ID
}

// SaveProvider saves the OAuth2 provider configuration.
func (s *Store) SaveProvider(ctx context.Context, args SaveProviderArgs) error {
	providerData, err := utils.ByteaValue(args.Provider.Data)

	if err != nil {
		return err
	}

	_, err = s.Exec(
		ctx, stmt.saveOAuthConfig,
		args.Provider.Name,
		providerData,
		args.Provider.Status,
		args.EnvID,
		args.AppID,
	)

	return err
}

type ProvidersArgs struct {
	EnvID        types.ID
	ProviderName string
}

func (s *Store) Providers(ctx context.Context, args ProvidersArgs) ([]*Provider, error) {
	where := []string{"env_id = $1"}
	params := []any{args.EnvID}

	if args.ProviderName != "" {
		where = append(where, "provider_name = $2")
		params = append(params, args.ProviderName)
	}

	buf := bytes.Buffer{}
	err := sqlTemplates.selectOAuthConfig.Execute(&buf, map[string]string{
		"where": strings.Join(where, " AND "),
	})

	if err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, buf.String(), params...)

	if err != nil || rows == nil {
		return nil, err
	}

	providers := []*Provider{}

	defer rows.Close()

	for rows.Next() {
		provider := &Provider{}
		var providerData []byte

		err = rows.Scan(
			&provider.ID,
			&provider.Name,
			&providerData,
			&provider.Status,
		)

		if err != nil {
			return nil, err
		}

		if err := utils.ByteaScan(providerData, &provider.Data); err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// Provider retrieves the OAuth2 client for a given environment and provider name.
func (s *Store) Provider(ctx context.Context, envID types.ID, providerName string) (*Provider, error) {
	providers, err := s.Providers(ctx, ProvidersArgs{
		EnvID:        envID,
		ProviderName: providerName,
	})

	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, nil
	}

	return providers[0], nil
}
