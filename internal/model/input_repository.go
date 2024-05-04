package model

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const INDEX_FIELD = "Index"

type InputRepository struct {
	Db *sqlx.DB
}

func (r *InputRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS inputs (
		id 				INTEGER NOT NULL PRIMARY KEY,
		input_index		integer,
		status	 		text,
		msg_sender	 	text,
		payload			text,
		block_number	integer,
		timestamp		integer,
		exception		text);`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Inputs table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *InputRepository) Create(input AdvanceInput) (*AdvanceInput, error) {
	insertSql := `INSERT INTO inputs (
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7
	);`
	_, err := r.Db.Exec(
		insertSql,
		input.Index,
		input.Status,
		input.MsgSender.Hex(),
		common.Bytes2Hex(input.Payload),
		input.BlockNumber,
		input.Timestamp.UnixMilli(),
		common.Bytes2Hex(input.Exception),
	)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) Update(input AdvanceInput) (*AdvanceInput, error) {
	sql := `UPDATE inputs
		SET status = $1, exception = $2
		WHERE input_index = $3`
	_, err := r.Db.Exec(
		sql,
		input.Status,
		common.Bytes2Hex(input.Exception),
		input.Index,
	)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) FindByStatus(status CompletionStatus) (*AdvanceInput, error) {
	sql := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception FROM inputs WHERE status = $1
		ORDER BY input_index ASC`
	res, err := r.Db.Queryx(
		sql,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindByIndex(index int) (*AdvanceInput, error) {
	sql := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception FROM inputs WHERE input_index = $1`
	res, err := r.Db.Queryx(
		sql,
		index,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (c *InputRepository) Count(
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM inputs `
	where, args, _, err := transformToInputQuery(filter)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.Get(&count, args...)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *InputRepository) FindAll(
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[AdvanceInput], error) {
	total, err := c.Count(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query := `SELECT
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception FROM inputs `
	where, args, argsCount, err := transformToInputQuery(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC `
	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf(`LIMIT $%d `, argsCount)
	args = append(args, limit)
	argsCount += 1
	query += fmt.Sprintf(`OFFSET $%d `, argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var inputs []AdvanceInput
	rows, err := stmt.Queryx(args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		input, err := parseInput(rows)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, *input)
	}

	pageResult := &commons.PageResult[AdvanceInput]{
		Rows:   inputs,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func transformToInputQuery(
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
		if *filter.Field == INDEX_FIELD {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else if filter.Gt != nil {
				where = append(where, fmt.Sprintf("input_index > $%d ", count))
				args = append(args, *filter.Gt)
				count += 1
			} else if filter.Lt != nil {
				where = append(where, fmt.Sprintf("input_index < $%d ", count))
				args = append(args, *filter.Lt)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == "Status" {
			if filter.Ne != nil {
				where = append(where, fmt.Sprintf("status <> $%d ", count))
				args = append(args, *filter.Ne)
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

func parseInput(res *sqlx.Rows) (*AdvanceInput, error) {
	var input AdvanceInput
	var msgSender string
	var payload string
	var exception string
	var timestamp int64
	err := res.Scan(
		&input.Index,
		&input.Status,
		&msgSender,
		&payload,
		&input.BlockNumber,
		&timestamp,
		&exception,
	)
	if err != nil {
		return nil, err
	}
	input.Payload = common.Hex2Bytes(payload)
	input.MsgSender = common.HexToAddress(msgSender)
	input.Timestamp = time.UnixMilli(timestamp)
	input.Exception = common.Hex2Bytes(exception)
	return &input, nil
}
