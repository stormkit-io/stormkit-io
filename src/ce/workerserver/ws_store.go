package jobs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"text/template"
	"time"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
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

// SyncAnalyticsRollupsParams is the range of whole UTC days, [From, To), to
// roll up into the daily aggregates.
type SyncAnalyticsRollupsParams struct {
	From time.Time
	To   time.Time

	// Finalize, when set, marks that day's rollups final in the same
	// transaction, so later runs stop rebuilding it.
	Finalize time.Time
}

// IsAnalyticsDayFinalized reports whether the path rollups of a day are final.
func (s *Store) IsAnalyticsDayFinalized(ctx context.Context, day time.Time) (bool, error) {
	row, err := s.QueryRow(ctx, stmt.isAnalyticsDayFinalized, day.UTC().Format(time.DateOnly))

	if err != nil {
		return false, err
	}

	var finalized bool

	err = row.Scan(&finalized)

	return finalized, err
}

const (
	// analyticsSingleViewPathsLimit is how many single-view paths a domain
	// keeps per day before the rest are folded into '(other)'.
	analyticsSingleViewPathsLimit = 1000

	// analyticsPathsLimit is how many paths a domain keeps per day at most.
	analyticsPathsLimit = 10000

	// analyticsRollupTimeout bounds one rollup run unless
	// STORMKIT_ANALYTICS_ROLLUP_TIMEOUT overrides it. It stays below the
	// hourly interval: a run that takes longer is rolled back and retried by
	// the next one, instead of loading the database indefinitely.
	analyticsRollupTimeout = 30 * time.Minute
)

// rollupTimeout returns how long one rollup run may take.
func rollupTimeout() time.Duration {
	if d, err := time.ParseDuration(os.Getenv("STORMKIT_ANALYTICS_ROLLUP_TIMEOUT")); err == nil && d > 0 {
		return d
	}

	return analyticsRollupTimeout
}

// rollupQuery is one statement of a rollup run.
type rollupQuery struct {
	sql  string
	args []any
}

// SyncAnalyticsRollups recomputes every daily aggregate for every day in the
// range, and the hourly visitor buckets of those days.
//
// The upserted aggregates (visitors, countries, events) commit one
// by one, each within its own timeout, so one slow rollup does not hold back
// the others. The path aggregates are deleted and rebuilt in one transaction,
// so reports never see a partly rebuilt day and a path folded into '(other)'
// since the last run leaves no stale row. The day is only finalized once every
// rollup has succeeded.
func (s *Store) SyncAnalyticsRollups(ctx context.Context, p SyncAnalyticsRollupsParams) error {
	from := p.From.UTC().Format(time.DateOnly)
	to := p.To.UTC().Format(time.DateOnly)
	days := []any{from, to}

	upserts := []rollupQuery{
		{stmt.syncAnalyticsCountries, days},
		{stmt.syncAnalyticsEvents, days},
	}

	for _, v := range []visitorsStatementParams{{}, {NotFound: true}, {Hourly: true}, {Hourly: true, NotFound: true}} {
		sql, err := visitorsStatement(v)

		if err != nil {
			return err
		}

		statuses := visitorStatuses200

		if v.NotFound {
			statuses = visitorStatuses404
		}

		upserts = append(upserts, rollupQuery{sql, []any{pq.Array(statuses), from, to}})
	}

	for _, q := range upserts {
		if err := s.execWithTimeout(ctx, q); err != nil {
			return err
		}
	}

	// The per-path rollup runs first: the others fold every path it did not
	// keep.
	paths := []rollupQuery{
		{stmt.deleteAnalyticsPathsDays, days},
		{stmt.deleteAnalyticsSiteDays, days},
		{stmt.deleteAnalyticsPathCountriesDays, days},
		{stmt.deleteAnalyticsCampaignsDays, days},
		{stmt.deleteAnalyticsChannelsDays, days},
		{stmt.syncAnalyticsPaths, []any{from, to, analyticsSingleViewPathsLimit, analyticsPathsLimit, analytics.MaxEngagementSeconds}},
		{stmt.syncAnalyticsSite, days},
		{stmt.syncAnalyticsPathCountries, days},
		{stmt.syncAnalyticsCampaigns, days},
		{stmt.syncAnalyticsChannels, []any{
			from, to,
			pq.Array(analytics.ChannelPatterns(analytics.ChannelAI)),
			pq.Array(analytics.ChannelPatterns(analytics.ChannelEmail)),
			pq.Array(analytics.ChannelPatterns(analytics.ChannelSearch)),
			pq.Array(analytics.ChannelPatterns(analytics.ChannelSocial)),
		}},
	}

	if !p.Finalize.IsZero() {
		paths = append(paths, rollupQuery{stmt.finalizeAnalyticsDay, []any{p.Finalize.UTC().Format(time.DateOnly)}})
	}

	ctx, cancel := context.WithTimeout(ctx, rollupTimeout())
	defer cancel()

	tx, err := s.Conn.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	for _, q := range paths {
		if _, err := tx.ExecContext(ctx, q.sql, q.args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// execWithTimeout runs one rollup statement within the rollup timeout.
func (s *Store) execWithTimeout(ctx context.Context, q rollupQuery) error {
	ctx, cancel := context.WithTimeout(ctx, rollupTimeout())
	defer cancel()

	_, err := s.Exec(ctx, q.sql, q.args...)

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

// RemoveOldAnalyticsEngagement deletes up to BatchSize engagement rows older
// than RetentionDays and returns the number of rows removed.
func (s *Store) RemoveOldAnalyticsEngagement(ctx context.Context, p RemoveOldAnalyticsParams) (int64, error) {
	res, err := s.Exec(ctx, stmt.removeOldAnalyticsEngagement, p.RetentionDays, p.BatchSize)

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

var (
	visitorStatuses200 = []int{200, 304}
	visitorStatuses404 = []int{404}
	visitorsTemplate   = template.Must(template.New("syncAnalyticsVisitors").Parse(stmt.syncAnalyticsVisitors))
)

type visitorsStatementParams struct {
	Hourly   bool
	NotFound bool
}

// visitorsStatement renders the visitors rollup for its table.
func visitorsStatement(p visitorsStatementParams) (string, error) {
	var sql bytes.Buffer

	if err := visitorsTemplate.Execute(&sql, p); err != nil {
		return "", err
	}

	return sql.String(), nil
}

// SyncAnalyticsVisitorsParams is the range [From, To) of the visitors rollup,
// in whole days, or in hours when Hourly is set.
type SyncAnalyticsVisitorsParams struct {
	From   time.Time
	To     time.Time
	Hourly bool
}

// SyncAnalyticsVisitors recomputes the 200 and 404 visitor aggregates of a
// range, outside the daily rollups' transaction.
func (s *Store) SyncAnalyticsVisitors(ctx context.Context, p SyncAnalyticsVisitorsParams) error {
	from := p.From.UTC().Format(time.DateTime)
	to := p.To.UTC().Format(time.DateTime)

	for _, notFound := range []bool{false, true} {
		sql, err := visitorsStatement(visitorsStatementParams{Hourly: p.Hourly, NotFound: notFound})

		if err != nil {
			return err
		}

		statuses := visitorStatuses200

		if notFound {
			statuses = visitorStatuses404
		}

		if _, err := s.Exec(ctx, sql, pq.Array(statuses), from, to); err != nil {
			return err
		}
	}

	return nil
}
