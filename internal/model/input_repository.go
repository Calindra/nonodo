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
		slog.Error("Create table error", err)
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
		?,
		?,
		?,
		?,
		?,
		?,
		?
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
		SET status = ?, exception = ? 
		WHERE input_index = ?`
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

func (r *InputRepository) FindByIndex(index int) (*AdvanceInput, error) {
	sql := `SELECT 
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception FROM inputs WHERE input_index = ?`
	res, err := r.Db.Queryx(
		sql,
		index,
	)
	if err != nil {
		return nil, err
	}
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
	where, args, err := transformToInputQuery(filter)
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
	where, args, err := transformToInputQuery(filter)
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
	query += `LIMIT ? `
	args = append(args, limit)
	query += `OFFSET ? `
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
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
) (string, []interface{}, error) {
	query := ""
	if len(filter) > 0 {
		query += "WHERE "
	}
	args := []interface{}{}
	where := []string{}
	for _, filter := range filter {
		if *filter.Field == INDEX_FIELD {
			if filter.Eq != nil {
				where = append(where, "input_index = ?")
				args = append(args, *filter.Eq)
			} else if filter.Gt != nil {
				where = append(where, "input_index > ?")
				args = append(args, *filter.Gt)
			} else if filter.Lt != nil {
				where = append(where, "input_index < ?")
				args = append(args, *filter.Lt)
			} else {
				return "", nil, fmt.Errorf("operation not implemented")
			}
		} else {
			return "", nil, fmt.Errorf("unexpected field %s", *filter.Field)
		}
	}
	query += strings.Join(where, " and ")
	return query, args, nil
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
