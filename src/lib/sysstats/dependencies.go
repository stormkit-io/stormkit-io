package sysstats

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
)

const dependenciesKey = "metrics:dependencies"

// TableSize is the total on-disk size of one table, including indexes and
// TOAST. Partitions are folded into their parent, so a partitioned table like
// access_logs reports one figure rather than a row per partition.
type TableSize struct {
	Name  string `json:"name"`
	Bytes uint64 `json:"bytes"`
}

// PostgresHealth describes the shared database. Pool statistics are not here:
// those are per-process and reported by each instance separately.
type PostgresHealth struct {
	Reachable      bool        `json:"reachable"`
	Error          string      `json:"error,omitempty"`
	LatencyMS      float64     `json:"latencyMs"`
	DatabaseBytes  uint64      `json:"databaseBytes"`
	Connections    int         `json:"connections"`
	MaxConnections int         `json:"maxConnections"`
	LargestTables  []TableSize `json:"largestTables,omitempty"`
}

type RedisHealth struct {
	Reachable       bool    `json:"reachable"`
	Error           string  `json:"error,omitempty"`
	LatencyMS       float64 `json:"latencyMs"`
	UsedMemoryBytes uint64  `json:"usedMemoryBytes"`
}

// DependencyHealth is the state of the services Stormkit cannot run without.
type DependencyHealth struct {
	Timestamp int64          `json:"ts"`
	Postgres  PostgresHealth `json:"postgres"`
	Redis     RedisHealth    `json:"redis"`
}

// PoolStats is one instance's database connection pool. Exhaustion shows up
// here — as waits — long before the database itself looks unhealthy.
type PoolStats struct {
	Open               int   `json:"open"`
	InUse              int   `json:"inUse"`
	Idle               int   `json:"idle"`
	WaitCount          int64 `json:"waitCount"`
	WaitDurationMS     int64 `json:"waitDurationMs"`
	MaxOpenConnections int   `json:"maxOpenConnections"`
}

func NewPoolStats(db *sql.DB) PoolStats {
	if db == nil {
		return PoolStats{}
	}

	s := db.Stats()

	return PoolStats{
		Open:               s.OpenConnections,
		InUse:              s.InUse,
		Idle:               s.Idle,
		WaitCount:          s.WaitCount,
		WaitDurationMS:     s.WaitDuration.Milliseconds(),
		MaxOpenConnections: s.MaxOpenConnections,
	}
}

// largestTablesQuery folds partitions into their parent via pg_partition_root,
// so access_logs reports one total instead of one row per partition. System
// schemas are excluded; per-environment app schemas are not, since their growth
// is worth seeing.
const largestTablesQuery = `
SELECT COALESCE(pg_partition_root(c.oid)::regclass::text, n.nspname || '.' || c.relname) AS table_name,
       SUM(pg_total_relation_size(c.oid))::bigint AS total_bytes
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind IN ('r', 'p')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
GROUP BY 1
ORDER BY 2 DESC
LIMIT 5;`

// dependencyCollector groups the queries behind DependencyHealth.
type dependencyCollector struct {
	db    *sql.DB
	cache *rediscache.RedisCache
}

func (d *dependencyCollector) postgres(ctx context.Context) PostgresHealth {
	health := PostgresHealth{}

	if d.db == nil {
		health.Error = "no database connection"
		return health
	}

	start := time.Now()

	if err := d.db.PingContext(ctx); err != nil {
		health.Error = err.Error()
		return health
	}

	health.Reachable = true
	health.LatencyMS = float64(time.Since(start).Microseconds()) / 1000

	// Each figure is best-effort: a permission error on one query should not
	// blank out the rest of the panel.
	if err := d.db.QueryRowContext(ctx, "SELECT pg_database_size(current_database())").Scan(&health.DatabaseBytes); err != nil {
		health.Error = err.Error()
	}

	_ = d.db.QueryRowContext(ctx, "SELECT count(*) FROM pg_stat_activity").Scan(&health.Connections)

	var maxConns string

	if err := d.db.QueryRowContext(ctx, "SHOW max_connections").Scan(&maxConns); err == nil {
		_, _ = fmt.Sscanf(maxConns, "%d", &health.MaxConnections)
	}

	health.LargestTables = d.largestTables(ctx)

	return health
}

func (d *dependencyCollector) largestTables(ctx context.Context) []TableSize {
	rows, err := d.db.QueryContext(ctx, largestTablesQuery)

	if err != nil {
		return nil
	}

	defer rows.Close()

	var out []TableSize

	for rows.Next() {
		var t TableSize

		if err := rows.Scan(&t.Name, &t.Bytes); err != nil {
			continue
		}

		out = append(out, t)
	}

	return out
}

func (d *dependencyCollector) redis(ctx context.Context) RedisHealth {
	health := RedisHealth{}

	if d.cache == nil {
		health.Error = "no redis connection"
		return health
	}

	start := time.Now()

	if err := d.cache.Ping(ctx).Err(); err != nil {
		health.Error = err.Error()
		return health
	}

	health.Reachable = true
	health.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
	health.UsedMemoryBytes = d.usedMemory(ctx)

	return health
}

func (d *dependencyCollector) usedMemory(ctx context.Context) uint64 {
	info, err := d.cache.Info(ctx, "memory").Result()

	if err != nil {
		return 0
	}

	return parseInfoUint(info, "used_memory")
}

type CollectDependenciesParams struct {
	DB    *sql.DB
	Cache *rediscache.RedisCache
}

// CollectDependencies snapshots Postgres and Redis health.
func CollectDependencies(ctx context.Context, p CollectDependenciesParams) *DependencyHealth {
	c := &dependencyCollector{db: p.DB, cache: p.Cache}

	return &DependencyHealth{
		Timestamp: time.Now().Unix(),
		Postgres:  c.postgres(ctx),
		Redis:     c.redis(ctx),
	}
}

// SaveDependencies stores the latest snapshot. Only the current state is kept —
// unlike machine samples, there is no history here, because "is the database up
// right now" is the question this answers.
func (s *Store) SaveDependencies(ctx context.Context, health *DependencyHealth) error {
	if s.client == nil {
		return fmt.Errorf("redis is not available")
	}

	payload, err := json.Marshal(health)

	if err != nil {
		return err
	}

	return s.client.Set(ctx, dependenciesKey, payload, keyTTL).Err()
}

// ReadDependencies returns the latest snapshot, or nil when none was taken yet.
func (s *Store) ReadDependencies(ctx context.Context) (*DependencyHealth, error) {
	if s.client == nil {
		return nil, fmt.Errorf("redis is not available")
	}

	raw, err := s.client.Get(ctx, dependenciesKey).Result()

	if err != nil {
		// A missing key means the leader has not run yet, which is a normal
		// state right after boot rather than a failure.
		return nil, nil
	}

	var health DependencyHealth

	if err := json.Unmarshal([]byte(raw), &health); err != nil {
		return nil, err
	}

	return &health, nil
}
