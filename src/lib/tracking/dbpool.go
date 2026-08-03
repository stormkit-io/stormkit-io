package tracking

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	dbConnectionsDesc = prometheus.NewDesc(
		"stormkit_db_connections",
		"Connections in the Stormkit database pool by state.",
		[]string{"state"},
		nil,
	)

	dbConnectionsMaxDesc = prometheus.NewDesc(
		"stormkit_db_connections_max",
		"Maximum number of open connections the pool allows.",
		nil,
		nil,
	)

	dbWaitTotalDesc = prometheus.NewDesc(
		"stormkit_db_wait_total",
		"Total number of times a caller had to wait for a free connection. A sustained rate here means the pool is undersized.",
		nil,
		nil,
	)

	dbWaitSecondsDesc = prometheus.NewDesc(
		"stormkit_db_wait_seconds_total",
		"Total time callers spent waiting for a free connection, in seconds.",
		nil,
		nil,
	)

	dbClosedTotalDesc = prometheus.NewDesc(
		"stormkit_db_closed_total",
		"Total number of connections closed by the pool, by the limit that closed them.",
		[]string{"reason"},
		nil,
	)
)

// dbPoolCollector exports connection pool saturation.
//
// postgres_exporter reports the server's view of connections. It cannot see
// that Stormkit is queuing for a slot in its own pool, which is what
// stormkit_db_wait_total measures — the pool can be starving while Postgres
// looks idle.
//
// The pool is resolved at scrape time rather than held, so registering the
// collector never forces a database connection to be opened.
type dbPoolCollector struct {
	db func() *sql.DB
}

func newDBPoolCollector(db func() *sql.DB) *dbPoolCollector {
	return &dbPoolCollector{db: db}
}

func (c *dbPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- dbConnectionsDesc
	ch <- dbConnectionsMaxDesc
	ch <- dbWaitTotalDesc
	ch <- dbWaitSecondsDesc
	ch <- dbClosedTotalDesc
}

func (c *dbPoolCollector) Collect(ch chan<- prometheus.Metric) {
	if c.db == nil {
		return
	}

	db := c.db()

	// Export nothing rather than zeroes when there is no pool yet. A flat zero
	// would read as an idle, healthy pool.
	if db == nil {
		return
	}

	stats := db.Stats()

	ch <- prometheus.MustNewConstMetric(dbConnectionsDesc, prometheus.GaugeValue, float64(stats.InUse), "in_use")
	ch <- prometheus.MustNewConstMetric(dbConnectionsDesc, prometheus.GaugeValue, float64(stats.Idle), "idle")
	ch <- prometheus.MustNewConstMetric(dbConnectionsMaxDesc, prometheus.GaugeValue, float64(stats.MaxOpenConnections))
	ch <- prometheus.MustNewConstMetric(dbWaitTotalDesc, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(dbWaitSecondsDesc, prometheus.CounterValue, stats.WaitDuration.Seconds())
	ch <- prometheus.MustNewConstMetric(dbClosedTotalDesc, prometheus.CounterValue, float64(stats.MaxIdleClosed), "max_idle")
	ch <- prometheus.MustNewConstMetric(dbClosedTotalDesc, prometheus.CounterValue, float64(stats.MaxIdleTimeClosed), "max_idle_time")
	ch <- prometheus.MustNewConstMetric(dbClosedTotalDesc, prometheus.CounterValue, float64(stats.MaxLifetimeClosed), "max_lifetime")
}
