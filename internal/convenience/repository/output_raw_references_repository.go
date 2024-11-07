package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

const RAW_VOUCHER_TYPE = "voucher"
const RAW_NOTICE_TYPE = "notice"

type RawOutputRefRepository struct {
	Db *sqlx.DB
}

type RawOutputRef struct {
	RawID       uint64    `db:"raw_id"`
	OutputIndex uint64    `db:"output_index"`
	InputIndex  uint64    `db:"input_index"`
	AppContract string    `db:"app_contract"`
	Type        string    `db:"type"`
	HasProof    bool      `db:"has_proof"`
	Executed    bool      `db:"executed"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (r *RawOutputRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_output_raw_references (
		raw_id 			integer NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		output_index	integer NOT NULL,
		has_proof		BOOLEAN,
		type            text NOT NULL CHECK (type IN ('voucher', 'notice')),
		executed        BOOLEAN,
		updated_at      TIMESTAMP NOT NULL,
		PRIMARY KEY (input_index, output_index, app_contract));
		
		CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_output_raw_references(input_index, app_contract);
		CREATE INDEX IF NOT EXISTS idx_convenience_output_raw_references_raw_id ON convenience_output_raw_references(raw_id);
		CREATE INDEX IF NOT EXISTS idx_convenience_output_raw_references_has_proof_raw_id ON convenience_output_raw_references(has_proof, raw_id);`
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
		has_proof,
		executed,
		updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		rawOutput.InputIndex,
		rawOutput.AppContract,
		rawOutput.OutputIndex,
		rawOutput.Type,
		rawOutput.RawID,
		rawOutput.HasProof,
		rawOutput.Executed,
		rawOutput.UpdatedAt,
	)

	if err != nil {
		slog.Error("Error creating output reference", "error", err)
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

func (r *RawOutputRefRepository) SetExecutedToTrue(ctx context.Context, rawOutputRef *RawOutputRef) error {
	exec := DBExecutor{r.Db}

	result, err := exec.ExecContext(ctx, `
		UPDATE convenience_output_raw_references
		SET executed = true,
		updated_at = $1
		WHERE raw_id = $2`, rawOutputRef.UpdatedAt, rawOutputRef.RawID)

	if err != nil {
		slog.Error("Error updating executed field", "error", err)
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
	slog.Debug("SetExecutedToTrue",
		"updated_at", rawOutputRef.UpdatedAt,
		"raw_id", rawOutputRef.RawID,
	)
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

func (r *RawOutputRefRepository) GetLastUpdatedAtExecuted(ctx context.Context) (*time.Time, *uint64, error) {
	var result struct {
		LastUpdatedAt time.Time `db:"updated_at"`
		RawID         uint64    `db:"raw_id"`
	}
	err := r.Db.GetContext(ctx, &result, `
		SELECT 
			updated_at, raw_id
		FROM
			convenience_output_raw_references 
		WHERE
			executed = true and type = 'voucher'
		ORDER BY updated_at DESC, raw_id DESC LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("No output ID executed = true")
			return nil, nil, nil
		}
		slog.Error("Failed to retrieve output ID not executed", "error", err)
		return nil, nil, err
	}
	return &result.LastUpdatedAt, &result.RawID, err
}
