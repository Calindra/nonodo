package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

const RAW_VOUCHER_TYPE = "voucher"
const RAW_NOTICE_TYPE = "notice"

type RawOutputRefRepository struct {
	Db *sqlx.DB
}

type RawOutputRef struct {
	RawID       uint64 `db:"raw_id"`
	OutputIndex uint64 `db:"output_index"`
	InputIndex  uint64 `db:"input_index"`
	AppContract string `db:"app_contract"`
	Type        string `db:"type"`
	HasProof    bool   `db:"has_proof"`
}

func (r *RawOutputRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_output_raw_references (
		raw_id 			integer NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		output_index	integer NOT NULL,
		has_proof		BOOLEAN,
		type            text NOT NULL CHECK (type IN ('voucher', 'notice')),
		PRIMARY KEY (input_index, output_index, app_contract));`
	_, err := r.Db.Exec(schema)
	if err != nil {
		slog.Error("Failed to create Raw Outputs table", "error", err)
	} else {
		slog.Debug("Raw Outputs table created successfully")
	}
	return err
}

func (r *RawOutputRefRepository) Create(ctx context.Context, rawOutput RawOutputRef) error {
	exec := DBExecutor{r.Db}

	_, err := exec.ExecContext(ctx, `INSERT INTO convenience_output_raw_references (
		input_index,
		app_contract,
		output_index,
		type,
		raw_id,
		has_proof) VALUES ($1, $2, $3, $4, $5, $6)`,
		rawOutput.InputIndex,
		rawOutput.AppContract,
		rawOutput.OutputIndex,
		rawOutput.Type,
		rawOutput.RawID,
		rawOutput.HasProof,
	)

	if err != nil {
		slog.Error("Error creating output", "error", err)
		return err
	}

	return err
}

func (r *RawOutputRefRepository) GetLatestOutputRawId(ctx context.Context) (uint64, error) {
	var outputId uint64
	err := r.Db.GetContext(ctx, &outputId, `SELECT raw_id FROM convenience_output_raw_references ORDER BY raw_id DESC LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		slog.Error("Failed to retrieve the latest output ID", "error", err)
		return 0, err
	}
	return outputId, err
}

func (r *RawOutputRefRepository) SetHasProofToTrue(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET has_proof = true
		WHERE raw_id = $1`, rawOutputRef.RawID)

	if err != nil {
		slog.Error("Error updating output proof", "error", err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Error fetching rows affected", "error", err)
		return err
	}

	if affected != 1 {
		return fmt.Errorf("unexpected number of rows updated: %d", affected)
	}

	return nil
}

func (r *RawOutputRefRepository) FindByID(ctx context.Context, id uint64) (*RawOutputRef, error) {
	var outputRef RawOutputRef
	err := r.Db.GetContext(ctx, &outputRef, `
		SELECT * FROM convenience_output_raw_references 
		WHERE raw_id = $1`, id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("Output reference not found", "raw_id", id)
			return nil, nil
		}
		slog.Error("Error finding output reference by ID", "error", err, "raw_id", id)
		return nil, err
	}
	return &outputRef, nil
}

func (r *RawOutputRefRepository) GetFirstOutputIdWithoutProof(ctx context.Context) (uint64, error) {
	var outputId uint64
	err := r.Db.GetContext(ctx, &outputId, `
		SELECT 
			raw_id 
		FROM
			convenience_output_raw_references 
		WHERE
			has_proof = false
		ORDER BY raw_id ASC LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("No output ID without proof found")
			return 0, nil
		}
		slog.Error("Failed to retrieve output ID without proof", "error", err)
		return 0, err
	}
	return outputId, err
}
