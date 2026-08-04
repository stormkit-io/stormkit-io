package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// DBConf represents a db configuration object.
type DBConf struct {
	Host     string
	User     string
	Password string
	DBName   string
	Schema   string
	SSLMode  string
	Port     string

	MaxLifetime  time.Duration
	MaxOpenConns int
	MaxIdleConns int

	DriverName string
}

// _db is read without the mutex so that observers such as the metrics endpoint
// never wait on the connect path, which holds dbmux across NewConnection's
// retry loop. dbmux still serialises establishing the pool.
var _db atomic.Pointer[sql.DB]

// Config holds the current configuration instance.
var Config DBConf

// URL returns the database url.
func (c DBConf) URL() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s", c.User, url.PathEscape(c.Password), c.Host, c.Port, c.DBName, c.SSLMode)
}

// Connection will return the currently open connection. If there is none found,
// it will create a new instance.
func Connection() *sql.DB {
	dbmux.Lock()
	defer dbmux.Unlock()

	if db := _db.Load(); db != nil {
		return db
	}

	db := NewConnection()
	_db.Store(db)

	return db
}

type ConnectionOptions struct {
	Schema string
}

var dbmux sync.Mutex

// Configure allows configure database connection.
func Configure(c DBConf) {
	Config = c
}

// NewConnection returns a new connection everytime this is called.
func NewConnection() *sql.DB {
	const maxRetry = 5

	slog.Info("connecting to database")
	var dbconnerr error

	for retry := 0; retry < maxRetry; retry++ {
		if db, err := NewConnectionWithConfig(Config); err != nil {
			slog.Info("retrying in 5 seconds:", retry)
			time.Sleep(5 * time.Second)
			dbconnerr = err
		} else {
			return db
		}
	}

	slog.Errorf("[database.NewConnection]: %s", dbconnerr.Error())
	return nil
}

func ConnectionString(cfg DBConf) string {
	return fmt.Sprintf(`
			host=%s port=%s user=%s
			password=%s dbname=%s sslmode=%s search_path=%s`,
		cfg.Host, cfg.Port, cfg.User,
		cfg.Password, cfg.DBName, utils.GetString(cfg.SSLMode, "disable"),
		cfg.Schema,
	)
}

// NewConnectionWithConfig returns a new connection with the given config.
func NewConnectionWithConfig(cfg DBConf) (*sql.DB, error) {
	db, err := sql.Open(utils.GetString(cfg.DriverName, "postgres"), ConnectionString(cfg))

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	slog.Infof("pinged successfully %s", cfg.Schema)
	return db, nil
}

// SetConnection replaces the cached sql connection with the given argument value.
// It is useful for tests so that they can mock the sql connection.
func SetConnection(db *sql.DB) {
	_db.Store(db)
}

// CurrentConnection returns the established connection, or nil when there is
// none yet. Unlike Connection it never opens one, so observers such as the
// metrics endpoint cannot trigger a connect and ping of their own.
//
// It also never takes dbmux: Connection holds that across NewConnection's retry
// loop, which sleeps 5 seconds between five attempts. A scrape that waited on
// it would exceed Prometheus' scrape timeout during exactly the database
// outage the pool metrics exist to explain.
func CurrentConnection() *sql.DB {
	return _db.Load()
}

// IsDuplicate checks whether an error is a duplicate error or not.
func IsDuplicate(dberr error) bool {
	duplicateErrCode := "23505"
	err, ok := dberr.(*pq.Error)
	return ok && err.Code == pq.ErrorCode(duplicateErrCode)
}
