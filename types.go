package dblock

import (
	"context"
	"database/sql"
	"errors"
)

// Querier is the common interface to execute queries on a DB, Tx, or Conn.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

var (
	_ Querier = &sql.DB{}
	_ Querier = &sql.Tx{}
)

var (
	ErrLockAlreadyAcquired = errors.New("lock already acquired")
	ErrLockReleaseFailed   = errors.New("lock release failed")
)

type ReleaseFunc func() error
type lockOptions struct {
	keyFunc func(string) (any, error)
}
type LockOption func(*lockOptions)

func WithKeyFunc(keyFunc func(string) (any, error)) LockOption {
	return func(o *lockOptions) {
		o.keyFunc = keyFunc
	}
}

// Logger is the interface that wraps the basic logging methods
// used by dblock. The methods are modeled after the standard
// library slog package. The default logger is a slog logger
type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Dialect is the interface for specific database implementations
type Dialect interface {
	TryLock(ctx context.Context, querier Querier, key any) error
	ReleaseLock(ctx context.Context, querier Querier, key any) error
}
