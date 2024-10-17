package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type RawNode struct {
	connectionURL string
}

type Input struct {
	ID                 int64  `db:"id"`
	Index              uint64 `db:"index"` // numeric(20,0)
	RawData            []byte `db:"raw_data"`
	BlockNumber        uint64 `db:"block_number"` // numeric(20,0)
	Status             string `db:"status"`
	MachineHash        []byte `db:"machine_hash,omitempty"`
	OutputsHash        []byte `db:"outputs_hash,omitempty"`
	ApplicationAddress []byte `db:"application_address"`
	EpochID            int64  `db:"epoch_id"`
}

type Report struct {
	ID      int64  `db:"id"`
	Index   string `db:"index"`
	RawData []byte `db:"raw_data"`
	InputID int64  `db:"input_id"`
}

type Output struct {
	ID                   uint64 `db:"id"`
	Index                string `db:"index"`
	RawData              []byte `db:"raw_data"`
	Hash                 []byte `db:"hash,omitempty"`
	OutputHashesSiblings []byte `db:"output_hashes_siblings,omitempty"`
	InputID              uint64 `db:"input_id"`
	TransactionHash      []byte `db:"transaction_hash,omitempty"`
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
}

const LIMIT = uint64(50)

type FilterID struct {
	IDgt uint64
}

func NewRawNode(connectionURL string) *RawNode {
	return &RawNode{connectionURL}
}

func (s *RawNode) Connect(ctx context.Context) (*sqlx.DB, error) {
	return sqlx.ConnectContext(ctx, "postgres", s.connectionURL)
}

func (s *RawNode) FindAllInputsByFilter(ctx context.Context, filter FilterInput, pag *Pagination) ([]Input, error) {
	inputs := []Input{}
	conn, err := s.Connect(ctx)
	if err != nil {
		return nil, err
	}

	limit := LIMIT
	if pag != nil {
		limit = pag.Limit
	}

	bindvarIdx := 1
	baseQuery := fmt.Sprintf("SELECT * FROM input WHERE ID >= $%d", bindvarIdx)
	bindvarIdx++
	args := []any{filter.IDgt}

	additionalFilter := ""

	if filter.IsStatusNone {
		additionalFilter = fmt.Sprintf(" AND status = '$%d'", bindvarIdx)
		bindvarIdx++
		args = append(args, "NONE")
	}

	pagination := fmt.Sprintf(" LIMIT $%d", bindvarIdx)
	// bindvarIdx += 2
	args = append(args, limit)

	query := baseQuery + additionalFilter + pagination
	slog.Debug("Query", "query", query)

	result, err := conn.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	for result.Next() {
		var input Input
		err := result.StructScan(&input)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}

	return inputs, nil
}

func (s *RawNode) FindAllReportsByFilter(ctx context.Context, filter FilterID) ([]Report, error) {
	reports := []Report{}
	conn, err := s.Connect(ctx)
	if err != nil {
		return nil, err
	}

	result, err := conn.QueryxContext(ctx, "SELECT * FROM report WHERE ID >= $1 LIMIT $2", filter.IDgt, LIMIT)
	if err != nil {
		return nil, err
	}

	for result.Next() {
		var report Report
		err := result.StructScan(&report)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func (s *RawNode) FindAllOutputsByFilter(ctx context.Context, filter FilterID) ([]Output, error) {
	outputs := []Output{}
	conn, err := s.Connect(ctx)
	if err != nil {
		return nil, err
	}

	result, err := conn.QueryxContext(ctx, "SELECT * FROM output WHERE ID >= $1 LIMIT $2", filter.IDgt, LIMIT)
	if err != nil {
		return nil, err
	}

	for result.Next() {
		var report Output
		err := result.StructScan(&report)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, report)
	}

	return outputs, nil
}
