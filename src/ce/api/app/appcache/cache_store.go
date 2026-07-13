package appcache

import (
	"database/sql"

	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type statement struct {
	selectResetCacheArgs string
}

var stmt = &statement{
	selectResetCacheArgs: `
		SELECT
			COALESCE(d.domain_name, ''), a.display_name
		FROM apps_build_conf e
			LEFT JOIN domains d ON e.env_id = d.env_id AND d.domain_verified IS TRUE
			LEFT JOIN apps a ON a.app_id = e.app_id
		WHERE
			e.env_id = $1 AND
			a.display_name IS NOT NULL;
	`,
}

// Store is the store to handle appconf logic
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{
		Store: database.NewStore(),
	}
}

type ResetCacheArgs struct {
	DomainName  string
	DisplayName string
}

func (s *Store) ResetCacheArgs(ctx context.Context, envID types.ID) ([]ResetCacheArgs, error) {
	args := []ResetCacheArgs{}
	rows, err := s.Query(ctx, stmt.selectResetCacheArgs, envID)

	if err == sql.ErrNoRows {
		return args, nil
	}

	if err != nil || rows == nil {
		return args, err
	}

	defer rows.Close()

	for rows.Next() {
		arg := ResetCacheArgs{}

		if err := rows.Scan(&arg.DomainName, &arg.DisplayName); err != nil {
			return args, err
		}

		args = append(args, arg)
	}

	return args, nil
}
