package repository

import (
	"context"
	"fmt"

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

func GetTransaction(ctx context.Context) (*sqlx.Tx, error) {
	tx, ok := ctx.Value(transactionKey).(*sqlx.Tx)
	if !ok {
		return nil, fmt.Errorf("no transaction found in context")
	}
	return tx, nil
}
