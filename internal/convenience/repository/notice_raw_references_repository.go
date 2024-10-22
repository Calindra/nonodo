package repository

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

type RawNoticeRefRepository struct {
	Db sqlx.DB
}

type RawNoticeRef struct {
	ID          string `db:"id"`
	OutputIndex uint64 `db:"output_index"`
	AppContract string `db:"app_contract"`
}

func (r *RawNoticeRefRepository) CreateTable() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_notice_raw_references (
		id 				text NOT NULL PRIMARY KEY,
		input_index		integer NOT NULL,
		app_contract    text NOT NULL,
		output_index		integer NOT NULL,
		FOREIGN KEY (input_index, app_contract) REFERENCES convenience_inputs (input_index, app_contract) ON DELETE CASCADE, 
		FOREIGN KEY (output_index) REFERENCES notices (output_index) ON DELETE CASCADE);`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Raw Notices table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}
