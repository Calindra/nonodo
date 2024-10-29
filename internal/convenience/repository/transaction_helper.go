package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type contextKey string

const transactionKey contextKey = "transaction"

func StartTransaction(ctx context.Context, db *sqlx.DB) (context.Context, error) {
	tx, err := db.Beginx()
	if err != nil {
		return ctx, fmt.Errorf("failed to begin transaction: %w", err)
	}

	ctx = context.WithValue(ctx, transactionKey, tx)
	return ctx, nil
}

func StartTransactionContext(ctx context.Context, db *sqlx.DB) (context.Context, *sqlx.Tx, error) {
	tx, err := db.Beginx()
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	ctx = context.WithValue(ctx, transactionKey, tx)
	return ctx, tx, nil
}

func GetTransaction(ctx context.Context) (*sqlx.Tx, bool) {
	tx, ok := ctx.Value(transactionKey).(*sqlx.Tx)
	if !ok {
		slog.Debug("No transaction found in context")
		return nil, false
	}
	return tx, true
}
