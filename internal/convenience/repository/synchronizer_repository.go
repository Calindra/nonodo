package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
)

type SynchronizerRepository struct {
	Db sqlx.DB
}

type DBExecutor struct {
	db *sqlx.DB
}

type TxExecutor struct {
	tx *sqlx.Tx
}

func (c *SynchronizerRepository) GetDB() *sqlx.DB {
	return &c.Db
}

func (c *SynchronizerRepository) BeginTxx(ctx context.Context) (*sqlx.Tx, error) {
	tx, err := c.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (c *SynchronizerRepository) CreateTables() error {
	idType := "INTEGER"

	if c.Db.DriverName() == "postgres" {
		idType = "SERIAL"
	}

	schema := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS synchronizer_fetch (
		id %s NOT NULL PRIMARY KEY,
		timestamp_after bigint,
		ini_cursor_after text,
		log_vouchers_ids text,
		end_cursor_after text,
		ini_input_cursor_after text,
		end_input_cursor_after text,
		ini_report_cursor_after text,
		end_report_cursor_after text
		);`, idType)

	// execute a query on the server
	_, err := c.Db.Exec(schema)

	return err
}
func (c *SynchronizerRepository) Create(
	ctx context.Context, data *model.SynchronizerFetch,
) (*model.SynchronizerFetch, error) {
	insertSql := `INSERT INTO synchronizer_fetch (
		timestamp_after,
		ini_cursor_after,
		log_vouchers_ids,
		end_cursor_after,
		ini_input_cursor_after,
		end_input_cursor_after,
		ini_report_cursor_after,
		end_report_cursor_after
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	var executor model.SQLExecutor
	tx, err := GetTransaction(ctx)

	if err != nil {
		executor = &DBExecutor{db: &c.Db}
	} else {
		executor = &TxExecutor{tx: tx}
	}

	if err := executor.Execute(ctx, insertSql, data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *SynchronizerRepository) Count(
	ctx context.Context,
) (uint64, error) {
	var count int
	err := c.Db.Get(&count, "SELECT count(*) FROM synchronizer_fetch")
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (c *SynchronizerRepository) GetLastFetched(
	ctx context.Context,
) (*model.SynchronizerFetch, error) {
	query := `SELECT * FROM synchronizer_fetch ORDER BY id DESC LIMIT 1`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("Error searching for last fetched", "Error", err)
		return nil, err
	}
	defer stmt.Close()
	var p model.SynchronizerFetch
	err = stmt.Get(&p)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("Error searching for last fetched", "Error", err)
		return nil, err
	}
	return &p, nil
}

func (e *DBExecutor) Execute(ctx context.Context, sql string, data *model.SynchronizerFetch) error {
	_, err := e.db.Exec(
		sql,
		data.TimestampAfter,
		data.IniCursorAfter,
		data.LogVouchersIds,
		data.EndCursorAfter,
		data.IniInputCursorAfter,
		data.EndInputCursorAfter,
		data.IniReportCursorAfter,
		data.EndReportCursorAfter,
	)
	return err
}

func (e *TxExecutor) Execute(ctx context.Context, sql string, data *model.SynchronizerFetch) error {
	_, err := e.tx.ExecContext(
		ctx,
		sql,
		data.TimestampAfter,
		data.IniCursorAfter,
		data.LogVouchersIds,
		data.EndCursorAfter,
		data.IniInputCursorAfter,
		data.EndInputCursorAfter,
		data.IniReportCursorAfter,
		data.EndReportCursorAfter,
	)
	return err
}
