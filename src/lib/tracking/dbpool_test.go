package tracking

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/suite"
)

var errNotImplemented = errors.New("not implemented")

// fakeDriver hands out connections without a server, so pool behaviour can be
// exercised without a live database.
type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return &fakeConn{}, nil }

type fakeConn struct{}

func (*fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errNotImplemented }
func (*fakeConn) Close() error                        { return nil }
func (*fakeConn) Begin() (driver.Tx, error)           { return nil, errNotImplemented }

func init() {
	sql.Register("trackingfake", fakeDriver{})
}

type DBPoolSuite struct {
	suite.Suite

	db *sql.DB
}

func (s *DBPoolSuite) SetupTest() {
	db, err := sql.Open("trackingfake", "")
	s.Require().NoError(err)

	db.SetMaxOpenConns(7)

	s.db = db
}

func (s *DBPoolSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

// collect gathers the collector into a name+label keyed map.
func (s *DBPoolSuite) collect(c prometheus.Collector) map[string]float64 {
	reg := prometheus.NewPedanticRegistry()
	s.Require().NoError(reg.Register(c))

	families, err := reg.Gather()
	s.Require().NoError(err)

	values := map[string]float64{}

	for _, family := range families {
		for _, metric := range family.GetMetric() {
			key := family.GetName()

			for _, label := range metric.GetLabel() {
				key += "|" + label.GetValue()
			}

			switch {
			case metric.GetGauge() != nil:
				values[key] = metric.GetGauge().GetValue()
			case metric.GetCounter() != nil:
				values[key] = metric.GetCounter().GetValue()
			}
		}
	}

	return values
}

func (s *DBPoolSuite) Test_ExportsEverySeries() {
	values := s.collect(newDBPoolCollector(func() *sql.DB { return s.db }))

	s.Len(values, 8)

	s.Contains(values, "stormkit_db_connections|in_use")
	s.Contains(values, "stormkit_db_connections|idle")
	s.Contains(values, "stormkit_db_wait_total")
	s.Contains(values, "stormkit_db_wait_seconds_total")
	s.Contains(values, "stormkit_db_closed_total|max_idle")
	s.Contains(values, "stormkit_db_closed_total|max_idle_time")
	s.Contains(values, "stormkit_db_closed_total|max_lifetime")
}

func (s *DBPoolSuite) Test_ReportsConfiguredMaxConnections() {
	values := s.collect(newDBPoolCollector(func() *sql.DB { return s.db }))

	s.Equal(float64(7), values["stormkit_db_connections_max"])
}

func (s *DBPoolSuite) Test_InUseTracksHeldConnection() {
	collector := newDBPoolCollector(func() *sql.DB { return s.db })

	s.Equal(float64(0), s.collect(collector)["stormkit_db_connections|in_use"])

	conn, err := s.db.Conn(context.Background())
	s.Require().NoError(err)

	s.Equal(float64(1), s.collect(collector)["stormkit_db_connections|in_use"])

	s.Require().NoError(conn.Close())

	s.Equal(float64(0), s.collect(collector)["stormkit_db_connections|in_use"])
}

// A pool that does not exist yet must export nothing. Zeroes would read as an
// idle, healthy pool.
func (s *DBPoolSuite) Test_NoPoolExportsNothing() {
	s.Empty(s.collect(newDBPoolCollector(nil)))
	s.Empty(s.collect(newDBPoolCollector(func() *sql.DB { return nil })))
}

func TestDBPoolSuite(t *testing.T) {
	suite.Run(t, new(DBPoolSuite))
}
