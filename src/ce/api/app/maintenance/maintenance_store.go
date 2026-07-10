package maintenance

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var stmt = struct {
	setConfig    string
	selectConfig string
}{
	setConfig: `
		UPDATE apps_build_conf SET maintenance_conf = $1 WHERE env_id = $2;
	`,
	selectConfig: `
		SELECT maintenance_conf FROM apps_build_conf WHERE env_id = $1;
	`,
}

type store struct {
	*database.Store
}

// Store returns a store instance.
func Store() *store {
	return &store{database.NewStore()}
}

// SetConfig stores the maintenance configuration for the environment.
func (s *store) SetConfig(ctx context.Context, envID types.ID, cfg *Config) error {
	_, err := s.Exec(ctx, stmt.setConfig, cfg, envID)
	return err
}

// Config returns the maintenance configuration associated with the environment.
func (s *store) Config(ctx context.Context, envID types.ID) (*Config, error) {
	row, err := s.QueryRow(ctx, stmt.selectConfig, envID)

	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	cfg := &Config{}

	if err := row.Scan(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
