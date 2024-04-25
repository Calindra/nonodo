package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
)

type SynchronizerRepository struct {
	Db sqlx.DB
}

func (c *SynchronizerRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS synchronizer_fetch (
		Id INTEGER NOT NULL PRIMARY KEY,
		TimestampAfter 	INTEGER,
		IniCursorAfter	text,
		LogVouchersIds 	text,
		EndCursorAfter  text);`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}
func (c *SynchronizerRepository) Create(
	ctx context.Context, data *model.SynchronizerFetch,
) (*model.SynchronizerFetch, error) {
	insertSql := `INSERT INTO synchronizer_fetch (
		TimestampAfter,
		IniCursorAfter,
		LogVouchersIds,
		EndCursorAfter) VALUES (?, ?, ?, ?)`
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
	query := `SELECT * FROM synchronizer_fetch ORDER BY Id DESC LIMIT 1`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
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
