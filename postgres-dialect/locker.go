package postgres_dialect

import (
	"context"
	"database/sql"

	"github.com/duchng/dblock"
)

func New() dblock.Dialect {
	return &postgresDialect{}
}

type postgresDialect struct{}

func (d *postgresDialect) TryLock(ctx context.Context, querier dblock.Querier, key any) (err error) {
	rows, err := querier.QueryContext(ctx, "select pg_try_advisory_lock($1)", key)
	if err != nil {
		return err
	}
	defer func() {
		scopedErr := rows.Close()
		if scopedErr != nil && err == nil {
			err = scopedErr
		}
	}()
	if !scanBool(rows) {
		return dblock.ErrLockAlreadyAcquired
	}
	return nil
}

func (d *postgresDialect) ReleaseLock(ctx context.Context, querier dblock.Querier, key any) (err error) {
	rows, err := querier.QueryContext(ctx, "select pg_advisory_unlock($1)", key)
	if err != nil {
		return err
	}
	defer func() {
		scopedErr := rows.Close()
		if scopedErr != nil && err == nil {
			err = scopedErr
		}
	}()
	if !scanBool(rows) {
		return dblock.ErrLockReleaseFailed
	}
	return nil
}

func scanBool(rows *sql.Rows) bool {
	if !rows.Next() {
		return false
	}
	var result bool
	if err := rows.Scan(&result); err != nil {
		return false
	}
	return result
}
