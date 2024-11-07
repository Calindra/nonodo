package synchronizernode

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type RawRepository struct {
	connectionURL string
	Db            *sqlx.DB
}

type RawInput struct {
	ID                 uint64    `db:"id"`
	Index              uint64    `db:"index"` // numeric(20,0)
	RawData            []byte    `db:"raw_data"`
	BlockNumber        uint64    `db:"block_number"` // numeric(20,0)
	Status             string    `db:"status"`
	MachineHash        []byte    `db:"machine_hash,omitempty"`
	OutputsHash        []byte    `db:"outputs_hash,omitempty"`
	ApplicationAddress []byte    `db:"application_address"`
	EpochID            uint64    `db:"epoch_id"`
	TransactionId      []byte    `db:"transaction_id"`
	UpdatedAt          time.Time `db:"updated_at"`
}

type Report struct {
	ID          int64  `db:"id"`
	Index       string `db:"index"`
	InputIndex  string `db:"input_index"`
	RawData     []byte `db:"raw_data"`
	InputID     int64  `db:"input_id"`
	AppContract []byte `db:"app_contract"`
}

type Output struct {
	ID                   uint64    `db:"id"`
	Index                string    `db:"index"`
	InputIndex           string    `db:"input_index"`
	RawData              []byte    `db:"raw_data"`
	Hash                 []byte    `db:"hash,omitempty"`
	OutputHashesSiblings []byte    `db:"output_hashes_siblings,omitempty"`
	InputID              uint64    `db:"input_id"`
	TransactionHash      []byte    `db:"transaction_hash,omitempty"`
	UpdatedAt            time.Time `db:"updated_at"`
	AppContract          []byte    `db:"app_contract"`
}

type FilterOutput struct {
	IDgt                uint64
	HaveTransactionHash bool
}

type Pagination struct {
	Limit uint64
	// Offset uint64
}

type FilterInput struct {
	IDgt         uint64
	IsStatusNone bool
	Status       string
}

const LIMIT = uint64(50)

type FilterID struct {
	IDgt uint64
}

func NewRawRepository(connectionURL string, db *sqlx.DB) *RawRepository {
	return &RawRepository{connectionURL, db}
}

func (s *RawRepository) FindAllInputsByFilter(ctx context.Context, filter FilterInput, pag *Pagination) ([]RawInput, error) {
	inputs := []RawInput{}

	limit := LIMIT
	if pag != nil {
		limit = pag.Limit
	}

	bindVarIdx := 1
	baseQuery := fmt.Sprintf("SELECT * FROM input WHERE ID >= $%d", bindVarIdx)
	bindVarIdx++
	args := []any{filter.IDgt}

	additionalFilter := ""

	if filter.IsStatusNone {
		additionalFilter = fmt.Sprintf(" AND status = \"$%d\"", bindVarIdx)
		bindVarIdx++
		args = append(args, "NONE")
	}

	if filter.Status != "" {
		additionalFilter = fmt.Sprintf(" AND status = $%d", bindVarIdx)
		bindVarIdx++
		args = append(args, filter.Status)
	}

	pagination := fmt.Sprintf(" LIMIT $%d", bindVarIdx)
	args = append(args, limit)

	orderBy := " ORDER BY ID ASC "
	query := baseQuery + additionalFilter + orderBy + pagination

	result, err := s.Db.QueryxContext(ctx, query, args...)
	if err != nil {
		slog.Error("Failed to execute query in FindAllInputsByFilter",
			"query", query, "args", args, "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var input RawInput
		err := result.StructScan(&input)
		if err != nil {
			slog.Error("Failed to scan row into RawInput struct", "error", err)
			return nil, err
		}
		inputs = append(inputs, input)
	}

	return inputs, nil
}

func (s *RawRepository) FindAllReportsByFilter(ctx context.Context, filter FilterID) ([]Report, error) {
	reports := []Report{}

	result, err := s.Db.QueryxContext(ctx, `
        SELECT
            r.id, r.index, r.raw_data, r.input_id, 
            inp.application_address as app_contract,
            inp.index as input_index
        FROM 
            report as r
        INNER JOIN
            input as inp
        ON
            r.input_id = inp.id
        WHERE r.id >= $1
        ORDER BY r.id ASC
        LIMIT $2
        `, filter.IDgt, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllReportsByFilter", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var report Report
		err := result.StructScan(&report)
		if err != nil {
			slog.Error("Failed to scan row into Report struct", "error", err)
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func (s *RawRepository) FindInputByOutput(ctx context.Context, filter FilterID) (*RawInput, error) {
	query := `SELECT * FROM input WHERE input.id = $1 LIMIT 1`
	stmt, err := s.Db.Preparex(query)
	if err != nil {
		slog.Error("Failed to prepare statement in FindInputByOutput", "query", query, "error", err)
		return nil, err
	}
	defer stmt.Close()

	var input RawInput
	err = stmt.GetContext(ctx, &input, filter.IDgt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("No input found for given output", "input_id", filter.IDgt)
			return nil, nil
		}
		slog.Error("Failed to get context for input in FindInputByOutput", "error", err)
		return nil, err
	}
	return &input, nil
}

func (s *RawRepository) FindAllOutputsByFilter(ctx context.Context, filter FilterID) ([]Output, error) {
	outputs := []Output{}

	result, err := s.Db.QueryxContext(ctx, `
        SELECT o.id, o.index, o.raw_data, o.hash, 
			o.output_hashes_siblings,
			o.input_id, o.transaction_hash, o.updated_at, 
			i.application_address app_contract,
			i.index input_index
		FROM output o
		INNER JOIN
			input i
			ON i.id = o.input_id
        WHERE o.id > $1 
        ORDER BY o.id ASC
        LIMIT $2`, filter.IDgt, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsByFilter", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var report Output
		err := result.StructScan(&report)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		outputs = append(outputs, report)
	}

	return outputs, nil
}

func (s *RawRepository) FindAllOutputsWithProof(ctx context.Context, filter FilterID) ([]Output, error) {
	outputs := []Output{}
	result, err := s.Db.QueryxContext(ctx, `
        SELECT * 
        FROM output 
        WHERE ID >= $1 and output_hashes_siblings IS NOT NULL
        ORDER BY ID ASC
        LIMIT $2
    `, filter.IDgt, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsWithProof", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var report Output
		err := result.StructScan(&report)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		outputs = append(outputs, report)
	}

	return outputs, nil
}

func (s *RawRepository) FindAllOutputsExecutedAfter(ctx context.Context, afterUpdatedAt time.Time, rawId uint64) ([]Output, error) {
	outputs := []Output{}
	result, err := s.Db.QueryxContext(ctx, `
        SELECT * 
        FROM output 
        WHERE ((updated_at > $1) or (updated_at = $1 and id > $2)) and transaction_hash IS NOT NULL
        ORDER BY updated_at ASC, id ASC
        LIMIT $3
    `, afterUpdatedAt, rawId, LIMIT)
	if err != nil {
		slog.Error("Failed to execute query in FindAllOutputsExecuted", "error", err)
		return nil, err
	}
	defer result.Close()

	for result.Next() {
		var report Output
		err := result.StructScan(&report)
		if err != nil {
			slog.Error("Failed to scan row into Output struct", "error", err)
			return nil, err
		}
		outputs = append(outputs, report)
	}

	return outputs, nil
}
