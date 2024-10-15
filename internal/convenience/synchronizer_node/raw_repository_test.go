package synchronizernode

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
)

type RawNodeSuite struct {
	suite.Suite
	node RawNode
	ctx  context.Context
}

func (s *RawNodeSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.ctx = context.Background()
	s.node = RawNode{
		connectionURL: "postgres://postgres:password@localhost:5432/test_rollupsdb?sslmode=disable",
	}
}

func TestRawNodeSuite(t *testing.T) {
	suite.Run(t, new(RawNodeSuite))
}

func (s *RawNodeSuite) TestSynchronizerNodeConnection() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	_, err := s.node.Connect(ctx)
	s.NoError(err)
}

func (s *RawNodeSuite) TestSynchronizerNodeListInputs() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	conn, err := s.node.Connect(ctx)
	s.NoError(err)

	result, err := conn.QueryxContext(ctx, "SELECT * FROM input")
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

func (s *RawNodeSuite) TestSynchronizerNodeInputByID() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	inputs, err := s.node.FindAllInputsByFilter(ctx, FilterInput{IDgt: 2})
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

func (s *RawNodeSuite) TestSynchronizerNodeReportByID() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	reports, err := s.node.FindAllReportsByFilter(ctx, FilterID{IDgt: 1})
	s.NoError(err)
	firstInput := reports[0]
	s.Equal(firstInput.ID, int64(1))

	slog.Info("Report", "reports", reports)

	b := reports[0].InputID

	firstInputID := big.NewInt(b)
	slog.Info("First Input ID", "firstInputID", firstInputID)

	firstInputIDDB := big.NewInt(1)

	s.Equal(firstInputIDDB, firstInputID)
}

func (s *RawNodeSuite) TestSynchronizerNodeOutputtByID() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	outputs, err := s.node.FindAllOutputsByFilter(ctx, FilterID{IDgt: 1})
	s.NoError(err)
	firstInput := outputs[0]
	s.Equal(firstInput.ID, int64(1))

	slog.Info("Output", "outputs", outputs)

	b := outputs[0].InputID

	firstInputID := big.NewInt(b)
	slog.Info("First Input ID", "firstInputID", firstInputID)

	firstInputIDDB := big.NewInt(1)

	s.Equal(firstInputIDDB, firstInputID)
}
