package repository

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type RawNoticeRefRepository struct {
	Db sqlx.DB
}

type RawNoticeRef struct {
	ID          string `db:"id"`
	OutputIndex uint64 `db:"output_index"`
	InputIndex  uint64 `db:"input_index"`
	AppContract string `db:"app_contract"`
	Type        string `db:"type"`
}

func (r *RawNoticeRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_output_raw_references (
		id 				text NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		output_index	integer NOT NULL,
		type            text NOT NULL CHECK (type IN ('voucher', 'notice')),
		PRIMARY KEY (input_index, output_index, app_contract));`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Raw Outputs table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *RawNoticeRefRepository) Create(ctx context.Context, rawOutput RawNoticeRef) error {
	exec := DBExecutor{&r.Db}

	result, err := exec.ExecContext(ctx, `INSERT INTO convenience_output_raw_references (
		id,
		input_index,
		app_contract,
		output_index,
		type) VALUES ($1, $2, $3, $4, $5)`,
		rawOutput.ID,
		rawOutput.InputIndex,
		rawOutput.AppContract,
		rawOutput.OutputIndex,
		rawOutput.Type,
	)

	if err != nil {
		slog.Error("Error creating output", "Error", err)
		return err
	}

	id, err := result.LastInsertId()
	if err == nil {
		slog.Debug("Raw Output saved", "id", id)
	}

	return err
}
