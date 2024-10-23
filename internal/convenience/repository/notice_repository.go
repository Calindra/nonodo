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
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type NoticeRepository struct {
	Db               sqlx.DB
	OutputRepository OutputRepository
	AutoCount        bool
}

func (c *NoticeRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS notices (
		payload 		text,
		input_index		integer,
		output_index	integer,
		app_contract    text,
		output_hashes_siblings text,
		PRIMARY KEY (input_index, output_index, app_contract));`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *NoticeRepository) Create(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	slog.Debug("CreateNotice", "payload", data.Payload)
	if c.AutoCount {
		count, err := c.OutputRepository.CountAllOutputs(ctx)
		if err != nil {
			return nil, err
		}
		data.OutputIndex = count
	}
	insertSql := `INSERT INTO notices (
		payload,
		input_index,
		output_index,
		app_contract,
		output_hashes_siblings) VALUES ($1, $2, $3, $4, $5)`

	exec := DBExecutor{&c.Db}
	_, err := exec.ExecContext(ctx,
		insertSql,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
		common.HexToAddress(data.AppContract).Hex(),
		data.OutputHashesSiblings,
	)
	if err != nil {
		slog.Error("Error creating notice", "Error", err)
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
	exec := DBExecutor{&c.Db}
	_, err := exec.ExecContext(
		ctx,
		sqlUpdate,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
	)
	if err != nil {
		slog.Error("Error updating notice", "Error", err)
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) SetProof(
	ctx context.Context, notice *model.ConvenienceNotice,
) error {
	updateVoucher := `UPDATE notices SET 
		output_hashes_siblings = $1
		WHERE app_contract = $2 and output_index = $3`
	exec := DBExecutor{&c.Db}
	res, err := exec.ExecContext(
		ctx,
		updateVoucher,
		notice.OutputHashesSiblings,
		common.HexToAddress(notice.AppContract).Hex(),
		notice.OutputIndex,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("wrong number of notices affected: %d; app_contract %v; output_index %d",
			affected, notice.AppContract, notice.OutputIndex,
		)
	}
	return nil
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

func (c *NoticeRepository) FindNoticeByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
) (*model.ConvenienceNotice, error) {
	rows, err := c.queryByOutputIndexAndAppContract(ctx, outputIndex, appContract)

	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var cNotice model.ConvenienceNotice
		if err := rows.StructScan(&cNotice); err != nil {
			return nil, err
		}

		return &cNotice, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *NoticeRepository) queryByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return c.Db.QueryxContext(ctx, `
			SELECT * FROM notices
			WHERE output_index = $1 and app_contract = $2
			LIMIT 1`,
			outputIndex,
			appContract.Hex(),
		)
	} else {
		return c.Db.QueryxContext(ctx, `
			SELECT * FROM notices
			WHERE output_index = $1
			LIMIT 1`,
			outputIndex,
		)
	}
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
		query += WHERE
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
		} else if *filter.Field == model.APP_CONTRACT {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("app_contract = $%d ", count),
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
	return query, args, count, nil
}
