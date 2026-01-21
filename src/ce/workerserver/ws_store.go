package jobs

import (
	"bytes"
	"context"
	"text/template"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
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

func (s *Store) RemoveOldLogs(ctx context.Context) error {
	_, err := s.Exec(ctx, stmt.removeOldLogs)
	return err
}

// MarkDeploymentArtifactsDeleted marks the deployment and its artifacts as deleted.
func (s *Store) MarkDeploymentArtifactsDeleted(ctx context.Context, ids []types.ID) error {
	_, err := s.Exec(ctx, stmt.markDeploymentArtifactsDeleted, pq.Array(ids))
	return err
}

// TimedOutDeployments returns deployments that have timed out.
func (s *Store) TimedOutDeployments(ctx context.Context) ([]types.ID, error) {
	ids := []types.ID{}

	rows, err := s.Query(ctx, stmt.selectTimedOutDeployments)

	if rows == nil || err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var id types.ID

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// DeploymentsOlderThan30Days returns 100 deployments older than 30 days.
// Returned deployments are not published.
func (s *Store) DeploymentsOlderThan30Days(ctx context.Context, numberOfDays, limit int) ([]*deploy.Deployment, error) {
	var wr bytes.Buffer
	tmpl := template.Must(template.New("old_deployments").Parse(stmt.selectOldOrDeletedDeployments))

	if err := tmpl.Execute(&wr, map[string]any{"days": numberOfDays}); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, wr.String(), limit)

	if rows == nil || err != nil {
		return nil, err
	}

	defer rows.Close()

	var deploys []*deploy.Deployment

	for rows.Next() {
		if deploys == nil {
			deploys = []*deploy.Deployment{}
		}

		d := &deploy.Deployment{}

		if err := rows.Scan(&d.ID, &d.AppID, &d.UploadResult); err != nil {
			return nil, err
		}

		deploys = append(deploys, d)
	}

	return deploys, nil
}

func (s *Store) UserIDsWithoutAPIKeys(ctx context.Context) ([]types.ID, error) {
	rows, err := s.Query(ctx, stmt.selectUserIDsWithoutAPIKeys)

	if rows == nil || err != nil {
		return nil, err
	}

	defer rows.Close()

	ids := []types.ID{}

	for rows.Next() {
		var id types.ID

		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, nil
}
