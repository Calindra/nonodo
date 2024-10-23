package repository

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type DBExecutor struct {
	db *sqlx.DB
}

func (c *DBExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx, isTxEnable := GetTransaction(ctx)

	if !isTxEnable {
		slog.Debug("Using ExecContext without transaction.")
		return c.db.ExecContext(ctx, query, args...)
	} else {
		// slog.Debug("Using ExecContext with transaction.")
		return tx.ExecContext(ctx, query, args...)
	}
}
