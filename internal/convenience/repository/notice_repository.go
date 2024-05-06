package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/jmoiron/sqlx"
)

type NoticeRepository struct {
	Db sqlx.DB
}

func (c *NoticeRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS notices (
		payload 		text,
		input_index		integer,
		output_index	integer,
		PRIMARY KEY (input_index, output_index));`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *NoticeRepository) Create(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	insertSql := `INSERT INTO notices (
		payload,
		input_index,
		output_index) VALUES ($1, $2, $3)`
	_, err := c.Db.ExecContext(ctx,
		insertSql,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
	)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) Update(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	sqlUpdate := `UPDATE notices SET 
		payload = $1
		WHERE input_index = $2 and output_index = $3`
	_, err := c.Db.ExecContext(
		ctx,
		sqlUpdate,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
	)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM notices `
	where, args, _, err := transformToNoticeQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, args...)
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
	where, args, argsCount, err := transformToNoticeQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC, output_index ASC `
	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf("LIMIT $%d ", argsCount)
	args = append(args, limit)
	argsCount = argsCount + 1
	query += fmt.Sprintf("OFFSET $%d ", argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var notices []model.ConvenienceNotice
	err = stmt.SelectContext(ctx, &notices, args...)
	if err != nil {
		return nil, err
	}
	pageResult := &commons.PageResult[model.ConvenienceNotice]{
		Rows:   notices,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func (c *NoticeRepository) FindByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	query := `SELECT * FROM notices WHERE input_index = $1 and output_index = $2 LIMIT 1`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var p model.ConvenienceNotice
	err = stmt.GetContext(ctx, &p, inputIndex, outputIndex)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func transformToNoticeQuery(
	filter []*model.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += "WHERE "
	}
	args := []interface{}{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == model.INPUT_INDEX {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("input_index = $%d ", count),
				)
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else {
			return "", nil, 0, fmt.Errorf("unexpected field %s", *filter.Field)
		}
	}
	query += strings.Join(where, " and ")
	slog.Debug("Query", "query", query, "args", args)
	return query, args, count, nil
}
