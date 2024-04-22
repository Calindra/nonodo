package model

import (
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

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
		timestamp		integer);`
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
		timestamp
	) VALUES (
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
	)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) FindByIndex(index int) (*AdvanceInput, error) {
	insertSql := `SELECT 
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp FROM inputs WHERE input_index = ?`
	res, err := r.Db.Queryx(
		insertSql,
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

func parseInput(res *sqlx.Rows) (*AdvanceInput, error) {
	var input AdvanceInput
	var msgSender string
	var payload string
	var timestamp int64
	err := res.Scan(
		&input.Index,
		&input.Status,
		&msgSender,
		&payload,
		&input.BlockNumber,
		&timestamp,
	)
	if err != nil {
		return nil, err
	}
	input.Payload = common.Hex2Bytes(payload)
	input.MsgSender = common.HexToAddress(msgSender)
	input.Timestamp = time.UnixMilli(timestamp)
	return &input, nil
}
