package repository

import (
	"context"
	"fmt"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
)

type DBExecutor[T model.SQLExecutorData] struct {
	db *sqlx.DB
}

type TxExecutor[T model.SQLExecutorData] struct {
	tx *sqlx.Tx
}

func (e *DBExecutor[T]) Execute(ctx context.Context, sql string, data T, getParams func(data interface{}) ([]interface{}, bool)) error {
	params, ok := getParams(data)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	_, err := e.db.ExecContext(ctx, sql, params...)
	return err
}

// Execute para TxExecutor
func (e *TxExecutor[T]) Execute(ctx context.Context, sql string, data T, getParams func(data interface{}) ([]interface{}, bool)) error {
	params, ok := getParams(data)
	if !ok {
		return fmt.Errorf("invalid data type")
	}

	_, err := e.tx.ExecContext(ctx, sql, params...)
	return err
}
