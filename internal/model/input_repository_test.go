package model

import (
	"log/slog"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"
)

type InputRepositorySuite struct {
	suite.Suite
	inputRepository *InputRepository
}

func (s *InputRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.inputRepository = &InputRepository{
		Db: db,
	}
	err := s.inputRepository.CreateTables()
	s.NoError(err)
}

func TestInputRepositorySuite(t *testing.T) {
	suite.Run(t, new(InputRepositorySuite))
}

func (s *InputRepositorySuite) TestCreateTables() {
	err := s.inputRepository.CreateTables()
	s.NoError(err)
}

func (s *InputRepositorySuite) TestCreateInput() {
	input, err := s.inputRepository.Create(AdvanceInput{
		Index:       0,
		Status:      CompletionStatusUnprocessed,
		MsgSender:   common.Address{},
		Payload:     common.Hex2Bytes("0x1122"),
		BlockNumber: 1,
		Timestamp:   time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
}

func (s *InputRepositorySuite) TestCreateAndFindInputByIndex() {
	input, err := s.inputRepository.Create(AdvanceInput{
		Index:       123,
		Status:      CompletionStatusUnprocessed,
		MsgSender:   common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		Payload:     common.Hex2Bytes("1122"),
		BlockNumber: 1,
		Timestamp:   time.Now(),
	})
	s.NoError(err)
	s.Equal(123, input.Index)

	input2, err := s.inputRepository.FindByIndex(123)
	s.NoError(err)
	s.Equal(123, input.Index)
	s.Equal("1122", common.Bytes2Hex(input.Payload))
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", input.MsgSender.Hex())
	s.Equal(input.Timestamp.UnixMilli(), input2.Timestamp.UnixMilli())
}
