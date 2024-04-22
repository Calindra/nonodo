package model

import "github.com/jmoiron/sqlx"

type ReportRepository struct {
	Db *sqlx.DB
}

func (r *ReportRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS reports (
		OutputIndex	integer,
		Payload 	text,
		InputIndex 	integer);`
	_, err := r.Db.Exec(schema)
	return err
}

func (r *ReportRepository) Create(report Report) (Report, error) {
	insertSql := `INSERT INTO reports (
		OutputIndex,
		Payload,
		InputIndex) VALUES (?, ?, ?)`
	r.Db.MustExec(
		insertSql,
		report.Index,
		report.Payload,
		report.InputIndex,
	)
	return report, nil
}
