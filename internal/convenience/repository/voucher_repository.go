package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const EXECUTED = "Executed"
const FALSE = "false"

type VoucherRepository struct {
	Db sqlx.DB
}

func (c *VoucherRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS vouchers (
		Destination text,
		Payload 	text,
		Executed	BOOLEAN,
		InputIndex 	integer,
		OutputIndex integer);`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *VoucherRepository) CreateVoucher(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	insertVoucher := `INSERT INTO vouchers (
		Destination,
		Payload,
		Executed,
		InputIndex,
		OutputIndex) VALUES (?, ?, ?, ?, ?)`
	c.Db.MustExec(
		insertVoucher,
		voucher.Destination,
		voucher.Payload,
		voucher.Executed,
		voucher.InputIndex,
		voucher.OutputIndex,
	)
	return voucher, nil
}

func (c *VoucherRepository) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceVoucher, error) {
	query := `SELECT * FROM vouchers WHERE inputIndex = ? and outputIndex = ?`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var p model.ConvenienceVoucher
	err = stmt.Get(&p, inputIndex, outputIndex)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *VoucherRepository) UpdateExecuted(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
	executedValue bool,
) error {
	query := `UPDATE vouchers SET Executed = ? WHERE inputIndex = ? and outputIndex = ?`
	c.Db.MustExec(query, executedValue, inputIndex, outputIndex)
	return nil
}

func (c *VoucherRepository) CountVouchers(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM vouchers `
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

func (c *VoucherRepository) FindAllVouchers(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) ([]model.ConvenienceVoucher, error) {
	total, err := c.CountVouchers(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM vouchers `
	where, args, err := transformToQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	query += `ORDER BY InputIndex ASC, OutputIndex ASC `
	offset, limit, err := computePage(first, last, after, before, int(total))
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
	var vouchers []model.ConvenienceVoucher
	err = stmt.Select(&vouchers, args...)
	if err != nil {
		return nil, err
	}
	return vouchers, nil
}

func transformToQuery(
	filter []*model.ConvenienceFilter,
) (string, []interface{}, error) {
	query := ""
	if len(filter) > 0 {
		query += "WHERE "
	}
	args := []interface{}{}
	where := []string{}
	for _, filter := range filter {
		if *filter.Field == EXECUTED {
			if *filter.Eq == "true" {
				where = append(where, "Executed = ?")
				args = append(args, true)
			} else if *filter.Eq == FALSE {
				where = append(where, "Executed = ?")
				args = append(args, false)
			} else {
				return "", nil, fmt.Errorf(
					"unexpected executed value %s", *filter.Eq,
				)
			}
		} else if *filter.Field == "Destination" {
			if filter.Eq != nil {
				where = append(where, "Destination = ?")
				if !common.IsHexAddress(*filter.Eq) {
					return "", nil, fmt.Errorf("wrong address value")
				}
				args = append(args, common.HexToAddress(*filter.Eq))
			} else {
				return "", nil, fmt.Errorf("operation not implemented")
			}
		} else {
			return "", nil, fmt.Errorf("unexpected field %s", *filter.Field)
		}
	}
	query += strings.Join(where, " and ")
	slog.Debug("Query", "query", query, "args", args)
	return query, args, nil
}
