package synchronizernode

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type SynchronizerNode struct {
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

type Filter struct {
	IDgt int64
}

func (s *SynchronizerNode) Connect() (*sqlx.DB, error) {
	return sqlx.Connect("postgres", s.connectionURL)
}

func (s *SynchronizerNode) FindAllInputsByFilter(filter Filter) ([]Input, error) {
	inputs := []Input{}
	conn, err := s.Connect()

	if err != nil {
		return nil, err
	}

	result, err := conn.Queryx("SELECT * FROM input WHERE ID >= $1", filter.IDgt)

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
