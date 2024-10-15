package synchronizernode

import (
	"context"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type RawNode struct {
	connectionURL string
}

type Input struct {
	ID                 int64  `db:"id"`
	Index              string `db:"index"` // numeric(20,0)
	RawData            []byte `db:"raw_data"`
	BlockNumber        string `db:"block_number"` // numeric(20,0)
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
	ID                   int64  `db:"id"`
	Index                string `db:"index"`
	RawData              []byte `db:"raw_data"`
	Hash                 []byte `db:"hash,omitempty"`
	OutputHashesSiblings []byte `db:"output_hashes_siblings,omitempty"`
	InputID              int64  `db:"input_id"`
	TransactionHash      []byte `db:"transaction_hash,omitempty"`
}

type FilterOutput struct {
	IDgt                int64
	HaveTransactionHash bool
}

type FilterInput struct {
	IDgt         int64
	IsStatusNone bool
}

const LIMIT = 50

type FilterID struct {
	IDgt int64
}

func NewRawNode(connectionURL string) *RawNode {
	return &RawNode{connectionURL}
}

func (s *RawNode) Connect(ctx context.Context) (*sqlx.DB, error) {
	return sqlx.ConnectContext(ctx, "postgres", s.connectionURL)
}

func (s *RawNode) FindAllInputsByFilter(ctx context.Context, filter FilterInput) ([]Input, error) {
	inputs := []Input{}
	conn, err := s.Connect(ctx)
	if err != nil {
		return nil, err
	}

	result, err := conn.Queryx("SELECT * FROM input WHERE ID >= $1 LIMIT $2", filter.IDgt, LIMIT)
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
