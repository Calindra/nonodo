package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type RawOutputRefRepository struct {
	Db sqlx.DB
}

type RawOutputRef struct {
	ID          uint64 `db:"id"`
	RawID       uint64 `db:"raw_id"`
	OutputIndex uint64 `db:"output_index"`
	InputIndex  uint64 `db:"input_index"`
	AppContract string `db:"app_contract"`
	Type        string `db:"type"`
}

func (r *RawOutputRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_output_raw_references (
		id 				integer NOT NULL,
		raw_id 			integer NOT NULL,
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

func (r *RawOutputRefRepository) Create(ctx context.Context, rawOutput RawOutputRef) error {
	exec := DBExecutor{&r.Db}

	result, err := exec.ExecContext(ctx, `INSERT INTO convenience_output_raw_references (
		id,
		input_index,
		app_contract,
		output_index,
		type,
		raw_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		rawOutput.ID,
		rawOutput.InputIndex,
		rawOutput.AppContract,
		rawOutput.OutputIndex,
		rawOutput.Type,
		rawOutput.RawID,
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

func (r *RawOutputRefRepository) GetLatestOutputRawId(ctx context.Context) (uint64, error) {
	var outputId uint64
	err := r.Db.GetContext(ctx, &outputId, `SELECT raw_id FROM convenience_output_raw_references ORDER BY id DESC LIMIT 1`)

	if err != nil {
		slog.Error("Failed to retrieve the last outputId from the database", "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return outputId, err
}
