package repository

import (
	"context"
	"log/slog"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
)

type NoticeRepository struct {
	Db sqlx.DB
}

func (c *NoticeRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS notices (
		Destination text,
		Payload 	text,
		InputIndex 	integer,
		OutputIndex integer);`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *NoticeRepository) Create(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	insertSql := `INSERT INTO notices (
		Destination,
		Payload,
		InputIndex,
		OutputIndex) VALUES (?, ?, ?, ?)`
	c.Db.MustExec(
		insertSql,
		data.Destination,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
	)
	return data, nil
}

func (c *NoticeRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM notices `
	where, args, err := transformToQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return 0, err
	}
	var count uint64
	err = stmt.Get(&count, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (c *NoticeRepository) FindAllNotices(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceNotice], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM notices `
	where, args, err := transformToQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	query += `ORDER BY InputIndex ASC, OutputIndex ASC `
	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += `LIMIT ? `
	args = append(args, limit)
	query += `OFFSET ? `
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var vouchers []model.ConvenienceNotice
	err = stmt.Select(&vouchers, args...)
	if err != nil {
		return nil, err
	}
	pageResult := &commons.PageResult[model.ConvenienceNotice]{
		Rows:   vouchers,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func (c *NoticeRepository) FindByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	query := `SELECT * FROM notices WHERE inputIndex = ? and outputIndex = ? LIMIT 1`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var p model.ConvenienceNotice
	err = stmt.Get(&p, inputIndex, outputIndex)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
