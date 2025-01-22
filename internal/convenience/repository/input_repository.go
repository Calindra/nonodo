package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

const INDEX_FIELD = "Index"

type InputRepository struct {
	Db sqlx.DB
}

type inputRow struct {
	ID                     string `db:"id"`
	Index                  int    `db:"input_index"`
	Status                 int    `db:"status"`
	MsgSender              string `db:"msg_sender"`
	Payload                string `db:"payload"`
	BlockNumber            int    `db:"block_number"`
	BlockTimestamp         int    `db:"block_timestamp"`
	PrevRandao             string `db:"prev_randao"`
	Exception              string `db:"exception"`
	AppContract            string `db:"app_contract"`
	EspressoBlockNumber    int    `db:"espresso_block_number"`
	EspressoBlockTimestamp int    `db:"espresso_block_timestamp"`
	InputBoxIndex          int    `db:"input_box_index"`
	AvailBlockNumber       int    `db:"avail_block_number"`
	AvailBlockTimestamp    int    `db:"avail_block_timestamp"`
	Type                   string `db:"type"`
	CartesiTransactionId   string `db:"cartesi_transaction_id"`
	ChainId                string `db:"chain_id"`
}

func (r *InputRepository) CreateTables() error {

	// the ID is not unique anymore in a multi-dapp environment
	schema := `CREATE TABLE IF NOT EXISTS convenience_inputs (
		id 				text NOT NULL,
		input_index		integer,
		app_contract    text,
		status	 		text,
		msg_sender	 	text,
		payload			text,
		block_number	integer,
		block_timestamp	NUMERIC,
		prev_randao		text,
		exception		text,
		espresso_block_number	integer,
		espresso_block_timestamp	NUMERIC,
		input_box_index integer,
		avail_block_number integer,
		avail_block_timestamp NUMERIC,
		type text,
		cartesi_transaction_id text,
		chain_id text);

	CREATE INDEX IF NOT EXISTS idx_input_index ON convenience_inputs(input_index);
	CREATE INDEX IF NOT EXISTS idx_status ON convenience_inputs(status);
	CREATE INDEX IF NOT EXISTS idx_input_id ON convenience_inputs(app_contract, id);
	CREATE INDEX IF NOT EXISTS idx_status_app_contract ON convenience_inputs(status, app_contract);
	CREATE INDEX IF NOT EXISTS idx_input_index_app_contract ON convenience_inputs(input_index, app_contract);`
	_, err := r.Db.Exec(schema)
	if err == nil {
		slog.Debug("Inputs table created")
	} else {
		slog.Error("Create table error", "error", err)
	}
	return err
}

func (r *InputRepository) Create(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	appContract := input.AppContract
	exist, err := r.FindByIDAndAppContract(ctx, input.ID, &appContract)
	if err != nil {
		return nil, err
	}
	if exist != nil {
		slog.Warn("Input already exists. Skipping creation")
		return exist, nil
	}
	return r.rawCreate(ctx, input)
}

func (r *InputRepository) rawCreate(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	insertSql := `INSERT INTO convenience_inputs (
		id,
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		app_contract,
		espresso_block_number,
		espresso_block_timestamp,
		input_box_index,
		avail_block_number,
		avail_block_timestamp,
		type,
		chain_id
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9,
		$10,
		$11,
		$12,
		$13,
		$14,
		$15,
		$16,
		$17
	);`

	var typee string = "inputbox"

	if input.Type != "" {
		typee = input.Type
	}

	var hexPayload string
	if !strings.HasPrefix(input.Payload, "0x") {
		hexPayload = "0x" + input.Payload
	} else {
		hexPayload = input.Payload
	}

	exec := DBExecutor{&r.Db}
	_, err := exec.ExecContext(
		ctx,
		insertSql,
		input.ID,
		input.Index,
		input.Status,
		input.MsgSender.Hex(),
		hexPayload,
		input.BlockNumber,
		input.BlockTimestamp.UnixMilli(),
		input.PrevRandao,
		common.Bytes2Hex(input.Exception),
		input.AppContract.Hex(),
		input.EspressoBlockNumber,
		input.EspressoBlockTimestamp.UnixMilli(),
		input.InputBoxIndex,
		input.AvailBlockNumber,
		input.AvailBlockTimestamp.UnixMilli(),
		typee,
		input.ChainId,
	)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) UpdateStatus(ctx context.Context, appContract common.Address, inputIndex uint64, status model.CompletionStatus) error {
	sql := `UPDATE convenience_inputs
	SET status = $1
	WHERE input_index = $2 and app_contract = $3`
	exec := DBExecutor{&r.Db}
	res, err := exec.ExecContext(
		ctx,
		sql,
		status,
		inputIndex,
		appContract.Hex(),
	)
	if err != nil {
		slog.Error("Error updating input status", "Error", err)
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no input's status updated: input_index %d; app_contract %s", inputIndex, appContract.Hex())
	}
	return nil
}

func (r *InputRepository) Update(ctx context.Context, input model.AdvanceInput) (*model.AdvanceInput, error) {
	sql := `UPDATE convenience_inputs
		SET status = $1, exception = $2
		WHERE input_index = $3`

	exec := DBExecutor{&r.Db}
	_, err := exec.ExecContext(
		ctx,
		sql,
		input.Status,
		common.Bytes2Hex(input.Exception),
		input.Index,
	)
	if err != nil {
		slog.Error("Error updating input", "Error", err)
		return nil, err
	}
	return &input, nil
}

func (r *InputRepository) FindByStatusNeDesc(ctx context.Context, status model.CompletionStatus) (*model.AdvanceInput, error) {
	sql := `SELECT
		id,
		input_index,
		status,
		msg_sender,
		payload,
		block_number,
		timestamp,
		exception,
		app_contract,
		espresso_block_number,
		espresso_block_timestamp,
		input_box_index,
		avail_block_number,
		avail_block_timestamp,
		type,
		chain_id FROM convenience_inputs WHERE status <> $1
		ORDER BY input_index DESC`
	res, err := r.Db.QueryxContext(
		ctx,
		sql,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindByStatus(ctx context.Context, status model.CompletionStatus) (*model.AdvanceInput, error) {
	sql := `SELECT
			id,
			input_index,
			status,
			msg_sender,
			payload,
			block_number,
			block_timestamp,
			prev_randao,
			exception,
			app_contract,
			espresso_block_number,
			espresso_block_timestamp,
			input_box_index,
			avail_block_number,
			avail_block_timestamp,
			type,
			chain_id
		FROM convenience_inputs WHERE status = $1
		ORDER BY input_index ASC`
	res, err := r.Db.QueryxContext(
		ctx,
		sql,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindByIndexAndAppContract(ctx context.Context,
	inputIndex int,
	appContract *common.Address,
) (*model.AdvanceInput, error) {
	res, err := r.queryByIndexAndAppContract(ctx, inputIndex, appContract)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) FindByIDAndAppContract(ctx context.Context, id string, appContract *common.Address) (*model.AdvanceInput, error) {
	res, err := r.queryByIdAndAppContract(ctx, id, appContract)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if res.Next() {
		input, err := parseInput(res)
		if err != nil {
			return nil, err
		}
		return input, nil
	}
	return nil, nil
}

func (r *InputRepository) queryByIdAndAppContract(
	ctx context.Context,
	id string,
	appContract *common.Address,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return r.Db.QueryxContext(ctx, `
			SELECT
				id,
				input_index,
				status,
				msg_sender,
				payload,
				block_number,
				block_timestamp,
				prev_randao,
				exception,
				app_contract,
				espresso_block_number,
				espresso_block_timestamp,
				input_box_index,
				avail_block_number,
				avail_block_timestamp,
				type,
				chain_id FROM convenience_inputs
			WHERE id = $1 and app_contract = $2
			LIMIT 1`,
			id,
			appContract.Hex(),
		)
	} else {
		return r.Db.QueryxContext(ctx, `
			SELECT
				id,
				input_index,
				status,
				msg_sender,
				payload,
				block_number,
				block_timestamp,
				prev_randao,
				exception,
				app_contract,
				espresso_block_number,
				espresso_block_timestamp,
				input_box_index,
				avail_block_number,
				avail_block_timestamp,
				type,
				chain_id FROM convenience_inputs
			WHERE id = $1
			LIMIT 1`,
			id,
		)
	}
}

func (r *InputRepository) queryByIndexAndAppContract(
	ctx context.Context,
	id int,
	appContract *common.Address,
) (*sqlx.Rows, error) {
	if appContract != nil {
		return r.Db.QueryxContext(ctx, `
			SELECT
				id,
				input_index,
				status,
				msg_sender,
				payload,
				block_number,
				block_timestamp,
				prev_randao,
				exception,
				app_contract,
				espresso_block_number,
				espresso_block_timestamp,
				input_box_index,
				avail_block_number,
				avail_block_timestamp,
				type,
				chain_id FROM convenience_inputs
			WHERE input_index = $1 and app_contract = $2
			LIMIT 1`,
			id,
			appContract.Hex(),
		)
	} else {
		return r.Db.QueryxContext(ctx, `
			SELECT
				id,
				input_index,
				status,
				msg_sender,
				payload,
				block_number,
				block_timestamp,
				prev_randao,
				exception,
				app_contract,
				espresso_block_number,
				espresso_block_timestamp,
				input_box_index,
				avail_block_number,
				avail_block_timestamp,
				type,
				chain_id FROM convenience_inputs
			WHERE input_index = $1
			LIMIT 1`,
			id,
		)
	}
}

func (c *InputRepository) GetNonce(
	ctx context.Context,
	appContract common.Address,
	msgSender common.Address,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_inputs
	WHERE app_contract = $1 and msg_sender = $2`
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, appContract.Hex(), msgSender.Hex())
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *InputRepository) Count(
	ctx context.Context,
	filter []*model.ConvenienceFilter,
) (uint64, error) {
	query := `SELECT count(*) FROM convenience_inputs `
	where, args, _, err := transformToInputQuery(filter)
	if err != nil {
		slog.Error("Count execution error", "err", err)
		return 0, err
	}
	query += where
	slog.Debug("Query", "query", query, "args", args)
	stmt, err := c.Db.Preparex(query)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	defer stmt.Close()
	var count uint64
	err = stmt.GetContext(ctx, &count, args...)
	if err != nil {
		slog.Error("Count execution error")
		return 0, err
	}
	return count, nil
}

func (c *InputRepository) FindAll(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.AdvanceInput], error) {
	total, err := c.Count(ctx, filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query := `SELECT
			id,
			input_index,
			status,
			msg_sender,
			payload,
			block_number,
			block_timestamp,
			prev_randao,
			exception,
			app_contract,
			espresso_block_number,
			espresso_block_timestamp,
			input_box_index,
			avail_block_number,
			avail_block_timestamp,
			type,
			chain_id
		FROM convenience_inputs `
	where, args, argsCount, err := transformToInputQuery(filter)
	if err != nil {
		slog.Error("database error", "err", err)
		return nil, err
	}
	query += where
	query += `ORDER BY input_index ASC `

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
		slog.Error("Find all error", "error", err)
		return nil, err
	}
	defer stmt.Close()
	var rows []inputRow
	erro := stmt.SelectContext(ctx, &rows, args...)
	if erro != nil {
		slog.Error("Find all error", "error", erro)
		return nil, erro
	}

	inputs := make([]model.AdvanceInput, len(rows))

	for i, row := range rows {
		inputs[i] = parseRowInput(row)
	}

	pageResult := &commons.PageResult[model.AdvanceInput]{
		Rows:   inputs,
		Total:  total,
		Offset: uint64(offset),
	}
	return pageResult, nil
}

func transformToInputQuery(
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
		if *filter.Field == INDEX_FIELD {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_index = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else if filter.Gt != nil {
				where = append(where, fmt.Sprintf("input_index > $%d ", count))
				args = append(args, *filter.Gt)
				count += 1
			} else if filter.Lt != nil {
				where = append(where, fmt.Sprintf("input_index < $%d ", count))
				args = append(args, *filter.Lt)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == "Status" {
			if filter.Ne != nil {
				where = append(where, fmt.Sprintf("status <> $%d ", count))
				args = append(args, *filter.Ne)
				count += 1
			} else if filter.Eq != nil {
				where = append(where, fmt.Sprintf("status = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented")
			}
		} else if *filter.Field == "MsgSender" {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("msg_sender = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented field msg_sender")
			}
		} else if *filter.Field == "Type" {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("type = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else if filter.Ne != nil {
				where = append(where, fmt.Sprintf("type <> $%d ", count))
				args = append(args, *filter.Ne)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented field type")
			}
		} else if *filter.Field == model.APP_CONTRACT {
			if filter.Eq != nil {
				where = append(where, fmt.Sprintf("app_contract = $%d ", count))
				args = append(args, *filter.Eq)
				count += 1
			} else {
				return "", nil, 0, fmt.Errorf("operation not implemented field app_contract")
			}
		} else if *filter.Field == "InputBoxIndex" {
			if filter.Ne != nil {
				where = append(where, fmt.Sprintf("input_box_index <> $%d ", count))
				args = append(args, *filter.Ne)
				count += 1
			} else if filter.Eq != nil {
				where = append(where, fmt.Sprintf("input_box_index = $%d ", count))
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

func parseRowInput(row inputRow) model.AdvanceInput {
	return model.AdvanceInput{
		ID:                     row.ID,
		Index:                  row.Index,
		Status:                 model.CompletionStatus(row.Status),
		MsgSender:              common.HexToAddress(row.MsgSender),
		Payload:                row.Payload,
		BlockNumber:            uint64(row.BlockNumber),
		BlockTimestamp:         time.UnixMilli(int64(row.BlockTimestamp)),
		PrevRandao:             row.PrevRandao,
		Exception:              common.Hex2Bytes(row.Exception),
		AppContract:            common.HexToAddress(row.AppContract),
		EspressoBlockNumber:    row.EspressoBlockNumber,
		EspressoBlockTimestamp: time.UnixMilli(int64(row.EspressoBlockTimestamp)),
		InputBoxIndex:          row.InputBoxIndex,
		AvailBlockNumber:       row.AvailBlockNumber,
		AvailBlockTimestamp:    time.UnixMilli(int64(row.AvailBlockTimestamp)),
		Type:                   row.Type,
		CartesiTransactionId:   row.CartesiTransactionId,
		ChainId:                row.ChainId,
	}
}

func parseInput(res *sqlx.Rows) (*model.AdvanceInput, error) {
	var (
		input                  model.AdvanceInput
		msgSender              string
		payload                string
		blockTimestamp         int64
		espressoBlockTimestamp int64
		prevRandao             string
		exception              string
		appContract            string
		availBlockTimestamp    int64
	)
	err := res.Scan(
		&input.ID,
		&input.Index,
		&input.Status,
		&msgSender,
		&payload,
		&input.BlockNumber,
		&blockTimestamp,
		&prevRandao,
		&exception,
		&appContract,
		&input.EspressoBlockNumber,
		&espressoBlockTimestamp,
		&input.InputBoxIndex,
		&input.AvailBlockNumber,
		&availBlockTimestamp,
		&input.Type,
		&input.ChainId,
	)
	if err != nil {
		return nil, err
	}
	input.Payload = payload
	input.MsgSender = common.HexToAddress(msgSender)
	input.BlockTimestamp = time.UnixMilli(blockTimestamp)
	input.PrevRandao = prevRandao
	input.Exception = common.Hex2Bytes(exception)
	input.AppContract = common.HexToAddress(appContract)
	input.EspressoBlockTimestamp = time.UnixMilli(espressoBlockTimestamp)
	input.AvailBlockTimestamp = time.UnixMilli(availBlockTimestamp)
	return &input, nil
}

func (c *InputRepository) BatchFindInputByInputIndexAndAppContract(
	ctx context.Context,
	filters []*BatchFilterItem,
) ([]*model.AdvanceInput, []error) {
	slog.Debug("BatchFindInputByInputIndexAndAppContract", "len", len(filters))

	query := `SELECT
		id,
		input_index,
		app_contract,
		status,
		msg_sender,
		payload,
		block_number,
		block_timestamp,
		prev_randao,
		exception,
		espresso_block_number,
		espresso_block_timestamp,
		input_box_index,
		avail_block_number,
		avail_block_timestamp,
		type,
		chain_id
	FROM convenience_inputs WHERE `

	args := []interface{}{}
	where := []string{}

	for i, filter := range filters {
		// nolint
		where = append(where, fmt.Sprintf(" (app_contract = $%d and input_index = $%d) ", i*2+1, i*2+2))
		args = append(args, filter.AppContract.Hex())
		args = append(args, filter.InputIndex)
	}

	query += strings.Join(where, " or ")

	errors := []error{}
	results := []*model.AdvanceInput{}
	stmt, err := c.Db.PreparexContext(ctx, query)
	if err != nil {
		slog.Error("BatchFind prepare context", "error", err)
		return nil, errors
	}
	defer stmt.Close()

	var inputs []inputRow

	err = stmt.SelectContext(ctx, &inputs, args...)
	if err != nil {
		slog.Error("BatchFind", "error", err)
		return nil, errors
	}

	inputMap := make(map[string]*model.AdvanceInput)
	for _, input := range inputs {

		key := GenerateBatchInputKey(input.AppContract, uint64(input.Index))
		advanceInput := parseRowInput(input)
		inputMap[key] = &advanceInput
	}

	for _, filter := range filters {
		key := GenerateBatchInputKey(filter.AppContract.Hex(), uint64(filter.InputIndex))
		advanceInput := inputMap[key]
		results = append(results, advanceInput)
	}

	slog.Debug("BatchResult", "results", len(results), "args", args, "query", "query")
	return results, nil
}

func GenerateBatchInputKey(appContract string, inputIndex uint64) string {
	return fmt.Sprintf("%s|%d", appContract, inputIndex)
}
