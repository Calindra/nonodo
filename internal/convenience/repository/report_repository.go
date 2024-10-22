package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/nonodo/internal/commons"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type ReportRepository struct {
	Db        *sqlx.DB
	AutoCount bool
}

func (r *ReportRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS convenience_reports (
		output_index  integer,
		payload       text,
		input_index   integer,
		app_contract  text,
		raw_id        integer,
		PRIMARY KEY (input_index, output_index, app_contract));`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Reports table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *ReportRepository) CreateReport(ctx context.Context, report cModel.Report) (cModel.Report, error) {
	slog.Debug("CreateReport", "payload", report.Payload)
	if r.AutoCount {
		count, err := r.Count(ctx, nil)
		if err != nil {
			slog.Error("database error", "err", err)
			return cModel.Report{}, err
		}
		report.Index = int(count)
	}
	insertSql := `INSERT INTO convenience_reports (
		output_index,
		payload,
		input_index,
		app_contract,
		raw_id) VALUES ($1, $2, $3, $4, $5)`

	exec := DBExecutor{r.Db}
	_, err := exec.ExecContext(
		ctx,
		insertSql,
		report.Index,
		common.Bytes2Hex(report.Payload),
		report.InputIndex,
		report.AppContract.Hex(),
		report.RawID,
	)

	if err != nil {
		slog.Error("database error", "err", err)
		return cModel.Report{}, err
	}
	slog.Debug("Report created",
		"outputIndex", report.Index,
		"inputIndex", report.InputIndex,
	)
	return report, nil
}

func (r *ReportRepository) Update(ctx context.Context, report cModel.Report) (*cModel.Report, error) {
	sql := `UPDATE convenience_reports
		SET payload = $1
		WHERE input_index = $2 and output_index = $3 `

	exec := DBExecutor{r.Db}
	_, err := exec.ExecContext(
		ctx,
		sql,
		common.Bytes2Hex(report.Payload),
		report.InputIndex,
		report.Index,
	)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ReportRepository) queryByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return r.Db.QueryxContext(ctx, `
			SELECT payload, input_index FROM convenience_reports
			WHERE output_index = $1 and app_contract = $2
			LIMIT 1`,
			outputIndex,
			appContract.Hex(),
		)
	} else {
		return r.Db.QueryxContext(ctx, `
			SELECT payload, input_index FROM convenience_reports
			WHERE output_index = $1
			LIMIT 1`,
			outputIndex,
		)
	}
}

func (r *ReportRepository) FindLastRawId(ctx context.Context) (uint64, error) {
	var outputId uint64
	err := r.Db.GetContext(ctx, &outputId, `SELECT raw_id FROM convenience_reports ORDER BY raw_id DESC LIMIT 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		slog.Error("Failed to retrieve the last raw_id from the database", "error", err)
		return 0, err
	}
	return outputId, err
}

func (r *ReportRepository) FindByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
) (*cModel.Report, error) {
	rows, err := r.queryByOutputIndexAndAppContract(ctx, outputIndex, appContract)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var payload string
		var inputIndex int
		if err := rows.Scan(&payload, &inputIndex); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: inputIndex,
			Index:      int(outputIndex),
			Payload:    common.Hex2Bytes(payload),
		}
		return report, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *ReportRepository) FindByInputAndOutputIndex(
	ctx context.Context,
	inputIndex uint64,
	outputIndex uint64,
) (*cModel.Report, error) {
	rows, err := r.Db.QueryxContext(ctx, `
		SELECT payload FROM convenience_reports
		WHERE input_index = $1 AND output_index = $2
		LIMIT 1`,
		inputIndex, outputIndex,
	)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: int(inputIndex),
			Index:      int(outputIndex),
			Payload:    common.Hex2Bytes(payload),
		}
		return report, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *ReportRepository) Count(
	ctx context.Context,
	filter []*cModel.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_reports `
	where, args, _, err := transformToReportQuery(filter)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.Get(&count, args...)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *ReportRepository) FindAllByInputIndex(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*commons.PageResult[cModel.Report], error) {
	filter := []*cModel.ConvenienceFilter{}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filter = append(filter, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	return c.FindAll(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ReportRepository) FindAll(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*cModel.ConvenienceFilter,
) (*commons.PageResult[cModel.Report], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}

	query := `SELECT input_index, output_index, payload FROM convenience_reports `
	where, args, argsCount, err := transformToReportQuery(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC, output_index ASC `

	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf(`LIMIT $%d `, argsCount)
	args = append(args, limit)
	argsCount += 1
	query += fmt.Sprintf(`OFFSET $%d `, argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var reports []cModel.Report
	rows, err := stmt.QueryxContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var payload string
		var inputIndex int
		var outputIndex int
		if err := rows.Scan(&inputIndex, &outputIndex, &payload); err != nil {
			return nil, err
		}
		report := &cModel.Report{
			InputIndex: inputIndex,
			Index:      outputIndex,
			Payload:    common.Hex2Bytes(payload),
		}
		reports = append(reports, *report)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	pageResult := &commons.PageResult[cModel.Report]{
		Rows:   reports,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func transformToReportQuery(
	filter []*cModel.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += WHERE
	}
	args := []interface{}{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == "OutputIndex" {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("output_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == cModel.INPUT_INDEX {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == cModel.APP_CONTRACT {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("app_contract = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else {
			return "", nil, 0, fmt.Errorf("unexpected field %s", *filter.Field)
		}
	}
	query += strings.Join(where, " and ")
	return query, args, count, nil
}
