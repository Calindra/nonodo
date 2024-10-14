package synchronizernode

import (
	"log/slog"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type SynchronizerNodeSuite struct {
	suite.Suite
	node SynchronizerNode
}

func (s *SynchronizerNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.node = SynchronizerNode{
		connectionURL: "postgres://postgres:password@localhost:5432/test_rollupsdb?sslmode=disable",
	}

}

func TestGraphileV2Suite(t *testing.T) {
	suite.Run(t, new(SynchronizerNodeSuite))
}

func (s *SynchronizerNodeSuite) TestSynchronizerNodeConnection() {
	_, err := s.node.Connect()
	s.NoError(err)
}

func (s *SynchronizerNodeSuite) TestSynchronizerNodeListInputs() {
	conn, err := s.node.Connect()
	s.NoError(err)

	result, err := conn.Queryx("SELECT * FROM input")
	s.NoError(err)

	inputs := []Input{}

	for result.Next() {
		var input Input
		err := result.StructScan(&input)
		s.NoError(err)
		inputs = append(inputs, input)
	}

	firstInput := inputs[0]
	s.Equal(firstInput.ID, int64(1))

	slog.Info("Inputs", "inputs", inputs)

	b := inputs[0].BlockNumber

	firstBlockNumber, ok := big.NewInt(0).SetString(b, 10)
	s.True(ok)
	slog.Info("First block number", "blockNumber", firstBlockNumber)

	firstBlockNumberDB := big.NewInt(1129)

	s.Equal(firstBlockNumberDB, firstBlockNumber)

	s.Equal("0x5112cF49F2511ac7b13A032c4c62A48410FC28Fb", common.BytesToAddress(inputs[0].ApplicationAddress).Hex())

}

func (s *SynchronizerNodeSuite) TestSynchronizerNodeSelectByID() {
	inputs, err := s.node.FindAllInputsByFilter(Filter{IDgt: 2})
	s.NoError(err)
	firstInput := inputs[0]
	s.Equal(firstInput.ID, int64(2))

	slog.Info("Inputs", "inputs", inputs)

	b := inputs[0].BlockNumber

	firstBlockNumber, ok := big.NewInt(0).SetString(b, 10)
	s.True(ok)
	slog.Info("First block number", "blockNumber", firstBlockNumber)

	firstBlockNumberDB := big.NewInt(1152)

	s.Equal(firstBlockNumberDB, firstBlockNumber)

	s.Equal("0x5112cF49F2511ac7b13A032c4c62A48410FC28Fb", common.BytesToAddress(inputs[0].ApplicationAddress).Hex())
}
