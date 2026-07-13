package deployhooks

import (
	"context"
	"database/sql"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// Store represents a store for the deployments and deployment logs.
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{database.NewStore()}
}

// AppDetailsForHooks returns the details necessary to comment
// on pull requests.
func (s *Store) AppDetailsForHooks(did types.ID) (*AppDetails, error) {
	d := &AppDetails{}

	row, err := s.QueryRow(context.TODO(), stmt.appDetailsForHooks, did)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&d.AppID, &d.IsAutoDeploy, &d.PullRequestNumber,
		&d.Repo, &d.DisplayName, &d.UserID, &d.AutoPublish,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return d, err
}
