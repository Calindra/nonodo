package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Defina um tipo de chave para o contexto
type contextKey string

// Crie uma constante para a chave da transação
const transactionKey contextKey = "transaction"

func StartTransaction(ctx context.Context, db *sqlx.DB) (context.Context, error) {
	tx, err := db.Beginx() // Inicia uma nova transação
	if err != nil {
		return ctx, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Adiciona a transação ao contexto
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
