package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
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

type voucherRow struct {
	Destination string `db:"destination"`
	Payload     string `db:"payload"`
	InputIndex  uint64 `db:"input_index"`
	OutputIndex uint64 `db:"output_index"`
	Executed    bool   `db:"executed"`
}

func (c *VoucherRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS vouchers (
		destination text,
		payload 	text,
		executed	BOOLEAN,
		input_index  integer,
		output_index integer,
		PRIMARY KEY (input_index, output_index));`

	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *VoucherRepository) CreateVoucher(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	insertVoucher := `INSERT INTO vouchers (
		destination,
		payload,
		executed,
		input_index,
		output_index) VALUES ($1, $2, $3, $4, $5)`
	c.Db.MustExec(
		insertVoucher,
		voucher.Destination.Hex(),
		voucher.Payload,
		voucher.Executed,
		voucher.InputIndex,
		voucher.OutputIndex,
	)
	return voucher, nil
}

func (c *VoucherRepository) UpdateVoucher(
	ctx context.Context, voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	updateVoucher := `UPDATE vouchers SET 
		destination = $1,
		payload = $2,
		executed = $3
		WHERE input_index = $4 and output_index = $5`

	c.Db.MustExec(
		updateVoucher,
		voucher.Destination.Hex(),
		voucher.Payload,
		voucher.Executed,
		voucher.InputIndex,
		voucher.OutputIndex,
	)

	return voucher, nil
}

func (c *VoucherRepository) VoucherCount(
	ctx context.Context,
) (uint64, error) {
	var count int
	err := c.Db.Get(&count, "SELECT count(*) FROM vouchers")
	if err != nil {
		return 0, nil
	}
	return uint64(count), nil
}

func (c *VoucherRepository) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceVoucher, error) {
	query := `SELECT * FROM vouchers WHERE input_index = $1 and output_index = $2`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var row voucherRow
	err = stmt.Get(&row, inputIndex, outputIndex)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, err
		}
	}

	p := convertToConvenienceVoucher(row)

	return &p, nil
}

func (c *VoucherRepository) UpdateExecuted(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
	executedValue bool,
) error {
	query := `UPDATE vouchers SET executed = $1 WHERE input_index = $2 and output_index = $3`
	c.Db.MustExec(query, executedValue, inputIndex, outputIndex)
	return nil
}

func (c *VoucherRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM vouchers `
	where, args, _, err := transformToQuery(filter)
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
) (*PageResult[model.ConvenienceVoucher], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM vouchers `
	where, args, argsCount, err := transformToQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	query += ` ORDER BY input_index ASC, output_index ASC `
	offset, limit, err := computePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}

	query += `LIMIT $` + strconv.Itoa(argsCount) + ` `
	args = append(args, limit)
	argsCount = argsCount + 1
	query += `OFFSET $` + strconv.Itoa(argsCount) + ` `
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	var rows []voucherRow
	err = stmt.Select(&rows, args...)
	if err != nil {
		return nil, err
	}

	fmt.Println("rows>>>>>>>>>>>>", rows)

	vouchers := make([]model.ConvenienceVoucher, len(rows))

	for i, row := range rows {
		vouchers[i] = convertToConvenienceVoucher(row)
	}

	pageResult := &PageResult[model.ConvenienceVoucher]{
		Rows:   vouchers,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func convertToConvenienceVoucher(row voucherRow) model.ConvenienceVoucher {
	destinationAddress := common.HexToAddress(row.Destination)

	voucher := model.ConvenienceVoucher{
		Destination: destinationAddress,
		Payload:     row.Payload,
		InputIndex:  row.InputIndex,
		OutputIndex: row.OutputIndex,
		Executed:    row.Executed,
	}

	return voucher
}

func transformToQuery(
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
		if *filter.Field == EXECUTED {
			if *filter.Eq == "true" {
				where = append(where, "executed = $"+strconv.Itoa(count))
				args = append(args, true)
				count += 1
			} else if *filter.Eq == FALSE {
				where = append(where, "executed = $"+strconv.Itoa(count))
				args = append(args, false)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf(
					"unexpected executed value %s", *filter.Eq,
				)
			}
		} else if *filter.Field == "Destination" {
			if filter.Eq != nil {
				where = append(where, "destination = $"+strconv.Itoa(count))
				if !common.IsHexAddress(*filter.Eq) {
					return "", nil, 0, fmt.Errorf("wrong address value")
				}
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
