package database

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"golang.org/x/net/context"
)

// SystemRow represents a new system row.
type SystemRow struct {
	MasterInstance   string
	MasterLastAccess time.Time
}

// Store represents a generic store
type Store struct {
	Conn *sql.DB
}

// NewStore returns a new store instance.
func NewStore() *Store {
	return &Store{
		Conn: Connection(),
	}
}

// Query is a wrapper around the sql.Stmt.QueryContext method.
// It prepares and executes the query
func (s *Store) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	stmt, err := s.Prepare(ctx, query)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	return stmt.QueryContext(ctx, args...)
}

// QueryRow is a wrapper around the sql.Stmt.QueryRowContext method.
// It prepares and executes the query
func (s *Store) QueryRow(ctx context.Context, query string, args ...any) (*sql.Row, error) {
	stmt, err := s.Prepare(ctx, query)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	return stmt.QueryRowContext(ctx, args...), nil
}

// Exec is a wrapper around the sql.Stmt.ExecContext method.
// It prepares and executes the query
func (s *Store) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	stmt, err := s.Prepare(ctx, query)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()

	return stmt.ExecContext(ctx, args...)
}

// Prepare prepares a new statement with context and returns it.
func (s *Store) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, err := s.Conn.PrepareContext(ctx, query)

	if err != nil {
		isContextCanceled := strings.EqualFold(strings.TrimSpace(err.Error()), "context canceled")

		if isContextCanceled {
			return nil, err
		}

		slog.Errorf("error while preparing query=%s, err=%v", query, err)
		return nil, err
	}

	if stmt == nil {
		return nil, errors.New("empty stmt")
	}

	return stmt, nil
}

// AdvisoryLock acquires a Postgres advisory lock with the given ID.
func (s *Store) AdvisoryLock(ctx context.Context, lockID int64) error {
	_, err := s.Exec(ctx, "SELECT pg_advisory_lock($1);", lockID)
	return err
}

// AdvisoryUnlock releases a Postgres advisory lock with the given ID.
func (s *Store) AdvisoryUnlock(ctx context.Context, lockID int64) error {
	_, err := s.Exec(ctx, "SELECT pg_advisory_unlock($1);", lockID)
	return err
}
