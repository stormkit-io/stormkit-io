package jobs

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"text/template"
	"time"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// accessLogPartitionName matches a dated request_logs partition (request_logs_YYYYMMDD),
// guarding the DDL helpers — whose table names cannot be parameterized — against
// anything but a name this code constructs itself.
var accessLogPartitionName = regexp.MustCompile(`^request_logs_(\d{8})$`)

const accessLogPartitionDateLayout = "20060102"

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

// RemoveOldAnalyticsParams configures a single batched deletion of raw
// analytics rows older than the retention window.
type RemoveOldAnalyticsParams struct {
	RetentionDays int
	BatchSize     int
}

// RemoveOldAnalytics deletes up to BatchSize raw analytics rows older than
// RetentionDays and returns the number of rows removed.
func (s *Store) RemoveOldAnalytics(ctx context.Context, p RemoveOldAnalyticsParams) (int64, error) {
	res, err := s.Exec(ctx, stmt.removeOldAnalytics, p.RetentionDays, p.BatchSize)

	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

// RemoveOldAnalyticsEvents deletes up to BatchSize raw custom-event rows older
// than RetentionDays and returns the number of rows removed.
func (s *Store) RemoveOldAnalyticsEvents(ctx context.Context, p RemoveOldAnalyticsParams) (int64, error) {
	res, err := s.Exec(ctx, stmt.removeOldAnalyticsEvents, p.RetentionDays, p.BatchSize)

	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

// CreateAccessLogPartition creates the daily request_logs partition covering the
// given day, if it does not already exist. The day is normalized to its date.
func (s *Store) CreateAccessLogPartition(ctx context.Context, day time.Time) error {
	day = day.UTC().Truncate(24 * time.Hour)
	name := "request_logs_" + day.Format(accessLogPartitionDateLayout)
	from := day.Format("2006-01-02")
	to := day.Add(24 * time.Hour).Format("2006-01-02")

	query := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s PARTITION OF request_logs FOR VALUES FROM ('%s') TO ('%s');",
		name, from, to,
	)

	_, err := s.Exec(ctx, query)

	return err
}

// AccessLogPartitions returns the names of all child partitions of request_logs
// (including the DEFAULT partition).
func (s *Store) AccessLogPartitions(ctx context.Context) ([]string, error) {
	rows, err := s.Query(ctx, stmt.selectAccessLogPartitions)

	if rows == nil || err != nil {
		return nil, err
	}

	defer rows.Close()

	names := []string{}

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return names, nil
}

// DropAccessLogPartition drops a dated request_logs partition by name. The name
// must match the request_logs_YYYYMMDD pattern; anything else is rejected so the
// non-parameterizable DROP cannot touch an arbitrary table.
func (s *Store) DropAccessLogPartition(ctx context.Context, name string) error {
	if !accessLogPartitionName.MatchString(name) {
		return fmt.Errorf("refusing to drop unexpected partition name: %q", name)
	}

	_, err := s.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", name))

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
