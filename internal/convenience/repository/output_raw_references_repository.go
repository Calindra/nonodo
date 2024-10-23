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
	Db sqlx.DB
}

type RawOutputRef struct {
	ID          uint64 `db:"id"`
	OutputIndex uint64 `db:"output_index"`
	InputIndex  uint64 `db:"input_index"`
	AppContract string `db:"app_contract"`
	Type        string `db:"type"`
	HasProof    bool   `db:"has_proof"`
}

func (r *RawOutputRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_output_raw_references (
		id 				integer NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		output_index	integer NOT NULL,
		has_proof		BOOLEAN,
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
		has_proof) VALUES ($1, $2, $3, $4, $5, $6)`,
		rawOutput.ID,
		rawOutput.InputIndex,
		rawOutput.AppContract,
		rawOutput.OutputIndex,
		rawOutput.Type,
		rawOutput.HasProof,
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

func (r *RawOutputRefRepository) GetLatestOutputId(ctx context.Context) (uint64, error) {
	var outputId uint64
	err := r.Db.GetContext(ctx, &outputId, `SELECT id FROM convenience_output_raw_references ORDER BY id DESC LIMIT 1`)

	if err != nil {
		slog.Error("Failed to retrieve the last outputId from the database", "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return outputId, err
}

func (r *RawOutputRefRepository) SetHasProofToTrue(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{&r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET has_proof = 1
		WHERE id = $1`,
		rawOutputRef.ID,
	)

	if err != nil {
		slog.Error("Error updating output", "Error", err)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		return fmt.Errorf("wrong rows updated")
	}
	slog.Debug("SetHasProofToTrue", "id", rawOutputRef.ID)
	return err
}

func (r *RawOutputRefRepository) FindByID(ctx context.Context, id uint64) (*RawOutputRef, error) {
	var outputRef RawOutputRef
	err := r.Db.GetContext(ctx, &outputRef, `
		SELECT * FROM convenience_output_raw_references 
		WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &outputRef, nil
}

func (r *RawOutputRefRepository) GetFirstOutputIdWithoutProof(ctx context.Context) (uint64, error) {
	var outputId uint64
	err := r.Db.GetContext(ctx, &outputId, `
		SELECT 
			id 
		FROM
			convenience_output_raw_references 
		WHERE
			has_proof = false
		ORDER BY id ASC LIMIT 1`)

	if err != nil {
		slog.Error("Failed to retrieve the last outputId from the database", "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return outputId, err
}
