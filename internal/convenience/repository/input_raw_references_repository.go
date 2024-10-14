package repository

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type RawInputRefRepository struct {
	Db sqlx.DB
}

type RawInputRef struct {
	ID          string `db:"id"`
	RawID       string `db:"raw_id"`
	InputIndex  int    `db:"input_index"`
	AppContract string `db:"app_contract"`
	Status      string `db:"status"`
	ChainID     string `db:"chain_id"`
}

func (r *RawInputRefRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_input_raw_references (
		id 				text NOT NULL PRIMARY KEY,
		raw_id 			text NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		status	 		text,
		chain_id text);
	CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_input_raw_references(input_index,app_contract);
	CREATE INDEX IF NOT EXISTS idx_status ON convenience_input_raw_references(status);`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Raw Inputs table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *RawInputRefRepository) Create(ctx context.Context, rawInput RawInputRef) error {
	exec := DBExecutor{&r.Db}

	result, err := exec.ExecContext(ctx, `INSERT INTO convenience_input_raw_references (
		raw_id,
		input_index,
		app_contract,
		status,
		chain_id) VALUES ($1, $2, $3, $4, $5)`,
		rawInput.RawID,
		rawInput.InputIndex,
		rawInput.AppContract,
		rawInput.Status,
		rawInput.ChainID,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		slog.Debug("Raw Input saved", "id", id)
	}

	return err
}
