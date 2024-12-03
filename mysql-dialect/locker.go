package mysql_dialect

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/duchng/dblock"
)

type mysqlDialect struct{}

func (d *mysqlDialect) TryLock(ctx context.Context, querier dblock.Querier, key any) (err error) {
	rows, err := querier.QueryContext(ctx, "select get_lock(?, 0)", fmt.Sprint(key))
	if err != nil {
		return err
	}
	defer func() {
		scopedErr := rows.Close()
		if scopedErr != nil && err == nil {
			err = scopedErr
		}
	}()
	if !scanResult(rows) {
		return dblock.ErrLockAlreadyAcquired
	}
	return nil
}

func (d *mysqlDialect) ReleaseLock(ctx context.Context, querier dblock.Querier, key any) error {
	rows, err := querier.QueryContext(ctx, "select release_lock(?)", fmt.Sprint(key))
	if err != nil {
		return err
	}
	defer func() {
		scopedErr := rows.Close()
		if scopedErr != nil && err == nil {
			err = scopedErr
		}
	}()
	if !scanResult(rows) {
		return dblock.ErrLockReleaseFailed
	}
	return nil
}

func scanResult(rows *sql.Rows) bool {
	if !rows.Next() {
		return false
	}
	var result sql.Null[int]
	if err := rows.Scan(&result); err != nil {
		return false
	}
	return result.Valid && result.V == 1
}

func New() dblock.Dialect {
	return &mysqlDialect{}
}
