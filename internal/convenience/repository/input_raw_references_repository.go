package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type RawInputRefRepository struct {
	Db sqlx.DB
}

type RawInputRef struct {
	ID          string `db:"id"`     // High level ID refers to our ConvenienceInput.ID
	RawID       uint64 `db:"raw_id"` // Low level ID
	InputIndex  uint64 `db:"input_index"`
	AppContract string `db:"app_contract"`
	Status      string `db:"status"`
	ChainID     string `db:"chain_id"`
}

func (r *RawInputRefRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_input_raw_references (
		id 				text NOT NULL,
		raw_id 			integer NOT NULL,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		status	 		text,
		chain_id        text);
	CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_input_raw_references(input_index, app_contract);
	CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_input_raw_references(raw_id, app_contract);
	CREATE INDEX IF NOT EXISTS idx_convenience_input_raw_references_status_raw_id ON convenience_input_raw_references(status, raw_id);
	CREATE INDEX IF NOT EXISTS idx_status ON convenience_input_raw_references(status);`

	_, err := r.Db.Exec(schema)
	if err != nil {
		slog.Error("Failed to create tables", "error", err)
		return err
	}
	slog.Debug("Raw Inputs table created successfully")
	return nil
}

func (r *RawInputRefRepository) UpdateStatus(ctx context.Context, rawInputsIds []string, status string) error {
	if len(rawInputsIds) == 0 {
		return nil
	}

	exec := DBExecutor{&r.Db}
	query, args, err := sqlx.In(`
		UPDATE convenience_input_raw_references 
		SET status = ? 
		WHERE raw_id IN (?)`, status, rawInputsIds)
	if err != nil {
		slog.Error("Failed to build query for status update", "query", query, "args", args, "error", err)
		return err
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)
	_, err = exec.ExecContext(ctx, query, args...)
	if err != nil {
		slog.Error("Failed to execute status update query", "query", query, "args", args, "error", err)
		return err
	}

	slog.Debug("Status updated successfully", "rawInputsIds", rawInputsIds, "status", status)
	return nil
}

func (r *RawInputRefRepository) Create(ctx context.Context, rawInput RawInputRef) error {
	exec := DBExecutor{&r.Db}

	appContract := common.HexToAddress(rawInput.AppContract)
	exist, err := r.FindByRawIdAndAppContract(ctx, rawInput.RawID, &appContract)
	if err != nil {
		return err
	}
	if exist != nil {
		slog.Warn("Raw input already exists. Skipping creation")
		return nil
	}

	_, err = exec.ExecContext(ctx, `INSERT INTO convenience_input_raw_references (
		id, raw_id, input_index, app_contract, status, chain_id) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		rawInput.ID, rawInput.RawID, rawInput.InputIndex,
		rawInput.AppContract, rawInput.Status, rawInput.ChainID)

	if err != nil {
		slog.Error("Failed to insert raw input reference", "rawInput", rawInput, "error", err)
		return err
	}

	slog.Debug("Raw input reference created", "ID", rawInput.ID)
	return nil
}

func (r *RawInputRefRepository) GetLatestRawId(ctx context.Context) (uint64, error) {
	var rawId uint64
	err := r.Db.GetContext(ctx, &rawId, `
		SELECT raw_id FROM convenience_input_raw_references 
		ORDER BY raw_id DESC LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("No raw input references found")
			return 0, nil
		}
		slog.Error("Failed to get latest raw ID", "error", err)
		return 0, err
	}

	slog.Debug("Latest raw ID fetched", "rawId", rawId)
	return rawId, nil
}

func (r *RawInputRefRepository) FindFirstInputByStatusNone(ctx context.Context, limit int) (*RawInputRef, error) {
	query := `SELECT * FROM convenience_input_raw_references
			  WHERE status = 'NONE'
			  ORDER BY raw_id ASC LIMIT $1`

	stmt, err := r.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("Failed to prepare query for status NONE", "query", query, "error", err)
		return nil, err
	}
	defer stmt.Close()

	args := []interface{}{limit}
	var row RawInputRef

	err = stmt.GetContext(ctx, &row, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("No input found with status NONE")
			return nil, nil
		}
		slog.Error("Failed to execute query for status NONE", "error", err)
		return nil, err
	}

	slog.Debug("First input with status NONE fetched", "row", row)
	return &row, nil
}

func (r *RawInputRefRepository) FindByRawIdAndAppContract(ctx context.Context, rawId uint64, appContract *common.Address) (*RawInputRef, error) {
	var inputRef RawInputRef
	err := r.Db.GetContext(ctx, &inputRef, `
		SELECT * FROM convenience_input_raw_references 
		WHERE raw_id = $1 and app_contract = $2
		LIMIT 1`, rawId, appContract.Hex())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("Input reference not found", "raw_id", rawId)
			return nil, nil
		}
		slog.Error("Error finding input reference by RAW_ID", "error", err, "raw_id", rawId)
		return nil, err
	}
	return &inputRef, nil
}
