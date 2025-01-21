package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type NoticeRepository struct {
	Db               sqlx.DB
	OutputRepository OutputRepository
	AutoCount        bool
}

func (c *NoticeRepository) CreateTables() error {
	schema := `CREATE TABLE IF NOT EXISTS notices (
		payload 		text,
		input_index		integer,
		output_index	integer,
		app_contract    text,
		output_hashes_siblings text,
		proof_output_index integer DEFAULT 0,
		PRIMARY KEY (input_index, output_index, app_contract)
	);

	CREATE INDEX IF NOT EXISTS idx_app_contract_input_index ON notices(app_contract, input_index);
	CREATE INDEX IF NOT EXISTS idx_app_contract_output_index ON notices(app_contract, output_index);
	CREATE INDEX IF NOT EXISTS idx_input_index_output_index ON notices(input_index, output_index);`
	// execute a query on the server
	_, err := c.Db.Exec(schema)
	return err
}

func (c *NoticeRepository) Create(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	slog.Debug("CreateNotice", "payload", data.Payload)
	if c.AutoCount {
		count, err := c.OutputRepository.CountAllOutputs(ctx)
		if err != nil {
			return nil, err
		}
		data.OutputIndex = count
		data.ProofOutputIndex = count
	}
	insertSql := `INSERT INTO notices (
		payload,
		input_index,
		output_index,
		app_contract,
		output_hashes_siblings,
		proof_output_index) VALUES ($1, $2, $3, $4, $5, $6)`

	exec := DBExecutor{&c.Db}
	_, err := exec.ExecContext(ctx,
		insertSql,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
		common.HexToAddress(data.AppContract).Hex(),
		data.OutputHashesSiblings,
		data.ProofOutputIndex,
	)
	if err != nil {
		slog.Error("Error creating notice", "Error", err)
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) Update(
	ctx context.Context, data *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	sqlUpdate := `UPDATE notices SET
		payload = $1
		WHERE input_index = $2 and output_index = $3`
	exec := DBExecutor{&c.Db}
	_, err := exec.ExecContext(
		ctx,
		sqlUpdate,
		data.Payload,
		data.InputIndex,
		data.OutputIndex,
	)
	if err != nil {
		slog.Error("Error updating notice", "Error", err)
		return nil, err
	}
	return data, nil
}

func (c *NoticeRepository) SetProof(
	ctx context.Context, notice *model.ConvenienceNotice,
) error {
	updateVoucher := `UPDATE notices SET
		output_hashes_siblings = $1,
		proof_output_index = $2
		WHERE app_contract = $3 and output_index = $4`
	exec := DBExecutor{&c.Db}
	res, err := exec.ExecContext(
		ctx,
		updateVoucher,
		notice.OutputHashesSiblings,
		notice.ProofOutputIndex,
		common.HexToAddress(notice.AppContract).Hex(),
		notice.OutputIndex,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("wrong number of notices affected: %d; app_contract %v; output_index %d",
			affected, notice.AppContract, notice.OutputIndex,
		)
	}
	return nil
}

func (c *NoticeRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM notices `
	where, args, _, err := transformToNoticeQuery(filter)
	if err != nil {
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (c *NoticeRepository) FindAllNotices(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceNotice], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		return nil, err
	}
	query := `SELECT * FROM notices `
	where, args, argsCount, err := transformToNoticeQuery(filter)
	if err != nil {
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC, output_index ASC `
	offset, limit, err := commons.ComputePage(first, last, after, before, int(total))
	if err != nil {
		return nil, err
	}
	query += fmt.Sprintf("LIMIT $%d ", argsCount)
	args = append(args, limit)
	argsCount = argsCount + 1
	query += fmt.Sprintf("OFFSET $%d ", argsCount)
	args = append(args, offset)

	slog.Debug("Query", "query", query, "args", args, "total", total)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var notices []model.ConvenienceNotice
	err = stmt.SelectContext(ctx, &notices, args...)
	if err != nil {
		return nil, err
	}
	pageResult := &commons.PageResult[model.ConvenienceNotice]{
		Rows:   notices,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func (c *NoticeRepository) FindAllNoticesByBlockNumber(
	ctx context.Context, startBlockGte uint64, endBlockLt uint64,
) ([]*model.ConvenienceNotice, error) {
	stmt, err := c.Db.Preparex(`
		SELECT
			n.payload,
			n.input_index,
			n.output_index,
			n.app_contract,
			n.output_hashes_siblings,
			n.proof_output_index
		FROM notices n
			INNER JOIN convenience_inputs i
				ON i.app_contract = n.app_contract AND i.input_index = n.input_index
		WHERE i.block_number >= $1 and i.block_number < $2`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var notices []*model.ConvenienceNotice
	err = stmt.SelectContext(ctx, &notices, startBlockGte, endBlockLt)
	if err != nil {
		return nil, err
	}
	return notices, nil
}

func (c *NoticeRepository) FindNoticeByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
) (*model.ConvenienceNotice, error) {
	rows, err := c.queryByOutputIndexAndAppContract(ctx, outputIndex, appContract)

	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var cNotice model.ConvenienceNotice
		if err := rows.StructScan(&cNotice); err != nil {
			return nil, err
		}

		return &cNotice, nil
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (c *NoticeRepository) queryByOutputIndexAndAppContract(
	ctx context.Context,
	outputIndex uint64,
	appContract *common.Address,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return c.Db.QueryxContext(ctx, `
			SELECT * FROM notices
			WHERE output_index = $1 and app_contract = $2
			LIMIT 1`,
			outputIndex,
			appContract.Hex(),
		)
	} else {
		return c.Db.QueryxContext(ctx, `
			SELECT * FROM notices
			WHERE output_index = $1
			LIMIT 1`,
			outputIndex,
		)
	}
}

func (c *NoticeRepository) FindByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	query := `SELECT * FROM notices WHERE input_index = $1 and output_index = $2 LIMIT 1`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var p model.ConvenienceNotice
	err = stmt.GetContext(ctx, &p, inputIndex, outputIndex)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func transformToNoticeQuery(
	filter []*model.ConvenienceFilter,
) (string, []interface{}, int, error) {
	query := ""
	if len(filter) > 0 {
		query += WHERE
	}
	args := []interface{}{}
	where := []string{}
	count := 1
	for _, filter := range filter {
		if *filter.Field == model.INPUT_INDEX {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("input_index = $%d ", count),
				)
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == model.APP_CONTRACT {
			if filter.Eq != nil {
				where = append(
					where,
					fmt.Sprintf("app_contract = $%d ", count),
				)
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

type BatchFilterItemForNotice struct {
	AppContract string
	InputIndex  int
}

func (c *NoticeRepository) BatchFindAllNoticesByInputIndexAndAppContract(
	ctx context.Context,
	filters []*BatchFilterItemForNotice,
) ([]*commons.PageResult[model.ConvenienceNotice], []error) {
	slog.Debug("BatchFindAllNoticesByInputIndexAndAppContract", "len", len(filters))

	query := `SELECT * FROM notices WHERE `

	args := []interface{}{}
	where := []string{}

	for i, filter := range filters {
		// nolint
		where = append(where, fmt.Sprintf(" (app_contract = $%d and input_index = $%d) ", i*2+1, i*2+2))
		args = append(args, filter.AppContract)
		args = append(args, filter.InputIndex)
	}

	query += strings.Join(where, " or ")

	errors := []error{}
	results := []*commons.PageResult[model.ConvenienceNotice]{}
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("BatchFind prepare context", "error", err)
		return nil, errors
	}
	defer stmt.Close()

	var notices []model.ConvenienceNotice

	err = stmt.SelectContext(ctx, &notices, args...)
	if err != nil {
		slog.Error("BatchFind", "error", err)
		return nil, errors
	}

	noticeMap := make(map[string]*commons.PageResult[model.ConvenienceNotice])
	for _, notice := range notices {
		key := GenerateBatchNoticeKey(notice.AppContract, notice.InputIndex)

		if noticeMap[key] == nil {
			noticeMap[key] = &commons.PageResult[model.ConvenienceNotice]{}
		}
		noticeMap[key].Total += 1
		noticeMap[key].Rows = append(noticeMap[key].Rows, notice)
	}

	for _, filter := range filters {
		key := GenerateBatchNoticeKey(filter.AppContract, uint64(filter.InputIndex))
		noticesItem := noticeMap[key]
		if noticesItem == nil {
			noticesItem = &commons.PageResult[model.ConvenienceNotice]{}
		}
		results = append(results, noticesItem)
	}

	slog.Debug("BatchResult", "results", len(results))
	return results, nil
}

func GenerateBatchNoticeKey(appContract string, inputIndex uint64) string {
	return fmt.Sprintf("%s|%d", appContract, inputIndex)
}
