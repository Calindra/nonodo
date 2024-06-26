package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
)

type SynchronizerRepository struct {
	Db sqlx.DB
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
		end_cursor_after text);`, idType)

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
		end_cursor_after) VALUES ($1, $2, $3, $4)`
	c.Db.MustExec(
		insertSql,
		data.TimestampAfter,
		data.IniCursorAfter,
		data.LogVouchersIds,
		data.EndCursorAfter,
	)
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
		return nil, err
	}
	defer stmt.Close()
	var p model.SynchronizerFetch
	err = stmt.Get(&p)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}
