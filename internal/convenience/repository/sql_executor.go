package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type DBExecutor struct {
	db *sqlx.DB
}

func (c *DBExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, err := GetTransaction(ctx)

	if err != nil {
		return c.db.ExecContext(ctx, query, args...)
	} else {
		return tx.ExecContext(ctx, query, args...)
	}
}
