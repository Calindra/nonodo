package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type RawInputRefRepository struct {
	Db sqlx.DB
}

type RawInputRef struct {
	ID          string `db:"id"`     // high level ID refers to our ConvenienceInput.ID
	RawID       uint64 `db:"raw_id"` // low level id
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
		return err
	}
	_, err = exec.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (r *RawInputRefRepository) Create(ctx context.Context, rawInput RawInputRef) error {
	exec := DBExecutor{&r.Db}

	result, err := exec.ExecContext(ctx, `INSERT INTO convenience_input_raw_references (
		id,
		raw_id,
		input_index,
		app_contract,
		status,
		chain_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		rawInput.ID,
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

func (r *RawInputRefRepository) GetLatestRawId(ctx context.Context) (uint64, error) {
	var rawId uint64
	err := r.Db.GetContext(ctx, &rawId, `SELECT raw_id FROM convenience_input_raw_references ORDER BY raw_id DESC LIMIT 1`)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return rawId, err
}

func (r *RawInputRefRepository) FindFirstInputByStatusNone(ctx context.Context, limit int) (*RawInputRef, error) {
	query := `SELECT * FROM convenience_input_raw_references
			WHERE status = 'NONE'
			ORDER BY raw_id ASC LIMIT $1
	`
	stmt, err := r.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("Find all by status none error", "error", err)
		return nil, err
	}
	defer stmt.Close()
	args := []interface{}{}
	args = append(args, limit)
	var row RawInputRef
	err = stmt.GetContext(ctx, &row, args...)
	if err != nil {
		slog.Error("Select context error", "error", err)
		return nil, err
	}
	return &row, err
}
