package repository

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	convenience "github.com/calindra/nonodo/internal/convenience/model"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type InputRepositorySuite struct {
	suite.Suite
	inputRepository *InputRepository
	dbFactory       *commons.DbFactory
}

func (s *InputRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateTempDb()
	s.inputRepository = &InputRepository{
		Db: *db,
	}
	err := s.inputRepository.CreateTables()
	s.NoError(err)
}

func TestInputRepositorySuite(t *testing.T) {
	// t.Parallel()
	suite.Run(t, new(InputRepositorySuite))
}

func (s *InputRepositorySuite) TestCreateTables() {
	err := s.inputRepository.CreateTables()
	s.NoError(err)
}

func (s *InputRepositorySuite) TestCreateInput() {
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          0,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
}

func (s *InputRepositorySuite) TestFixCreateInputDuplicated() {
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          0,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
	input, err = s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          0,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(0, input.Index)
	count, err := s.inputRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(uint64(1), count)
}

func (s *InputRepositorySuite) TestCreateAndFindInputByID() {
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		ID:             "123",
		Index:          123,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		Payload:        common.Hex2Bytes("1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
	})
	s.NoError(err)
	s.Equal(123, input.Index)

	input2, err := s.inputRepository.FindByID(ctx, "123")
	s.NoError(err)
	s.Equal("123", input.ID)
	s.Equal(input.Status, input2.Status)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", input.MsgSender.Hex())
	s.Equal("1122", common.Bytes2Hex(input.Payload))
	s.Equal(1, int(input2.BlockNumber))
	s.Equal(input.BlockTimestamp.UnixMilli(), input2.BlockTimestamp.UnixMilli())
}

func (s *InputRepositorySuite) TestCreateInputAndUpdateStatus() {
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		ID:             "2222",
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
	})
	s.NoError(err)
	s.Equal(2222, input.Index)

	input.Status = convenience.CompletionStatusAccepted
	_, err = s.inputRepository.Update(ctx, *input)
	s.NoError(err)

	input2, err := s.inputRepository.FindByID(ctx, "2222")
	s.NoError(err)
	s.Equal(convenience.CompletionStatusAccepted, input2.Status)
	s.Equal("0x70997970C51812dc3A010C7d01b50e0d17dc79C8", input2.AppContract.Hex())
}

func (s *InputRepositorySuite) TestCreateInputFindByStatus() {
	ctx := context.Background()
	input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		PrevRandao:     "0xdeadbeef",
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})
	s.NoError(err)
	s.Equal(2222, input.Index)

	input2, err := s.inputRepository.FindByStatus(ctx, convenience.CompletionStatusUnprocessed)
	s.NoError(err)
	s.Equal(2222, input2.Index)

	input.Status = convenience.CompletionStatusAccepted
	_, err = s.inputRepository.Update(ctx, *input)
	s.NoError(err)

	input2, err = s.inputRepository.FindByStatus(ctx, convenience.CompletionStatusUnprocessed)
	s.NoError(err)
	s.Nil(input2)

	input2, err = s.inputRepository.FindByStatus(ctx, convenience.CompletionStatusAccepted)
	s.NoError(err)
	s.Equal(2222, input2.Index)
}

func (s *InputRepositorySuite) TestFindByIndexGt() {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			ID:             strconv.Itoa(i),
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.Address{},
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    common.Address{},
		})
		s.NoError(err)
		s.Equal(i, input.Index)
	}
	filters := []*convenience.ConvenienceFilter{}
	value := "1"
	field := INDEX_FIELD
	filters = append(filters, &convenience.ConvenienceFilter{
		Field: &field,
		Gt:    &value,
	})
	resp, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(3, int(resp.Total))
}

func (s *InputRepositorySuite) TestFindByIndexLt() {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			ID:             strconv.Itoa(i),
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.Address{},
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    common.Address{},
		})
		s.NoError(err)
		s.Equal(i, input.Index)
	}
	filters := []*convenience.ConvenienceFilter{}
	value := "3"
	field := INDEX_FIELD
	filters = append(filters, &convenience.ConvenienceFilter{
		Field: &field,
		Lt:    &value,
	})
	resp, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(3, int(resp.Total))
}

func (s *InputRepositorySuite) TestFindByMsgSender() {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		input, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
			ID:             strconv.Itoa(i),
			Index:          i,
			Status:         convenience.CompletionStatusUnprocessed,
			MsgSender:      common.HexToAddress(fmt.Sprintf("000000000000000000000000000000000000000%d", i)),
			Payload:        common.Hex2Bytes("0x1122"),
			BlockNumber:    1,
			BlockTimestamp: time.Now(),
			AppContract:    common.Address{},
		})
		s.NoError(err)
		s.Equal(i, input.Index)
	}
	filters := []*convenience.ConvenienceFilter{}
	value := "0x0000000000000000000000000000000000000002"
	field := "MsgSender"
	filters = append(filters, &convenience.ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	})
	resp, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(1, int(resp.Total))
	s.Equal(common.HexToAddress(value), resp.Rows[0].MsgSender)
}

func (s *InputRepositorySuite) TestColumnDappAddressExists() {
	query := `PRAGMA table_info(convenience_inputs);`

	rows, err := s.inputRepository.Db.Queryx(query)
	s.NoError(err)

	defer rows.Close()

	var columnExists bool
	for rows.Next() {
		var cid int
		var name, fieldType string
		var notNull, pk int
		var dfltValue interface{}

		err = rows.Scan(&cid, &name, &fieldType, &notNull, &dfltValue, &pk)
		s.NoError(err)

		if name == "app_contract" {
			columnExists = true
			break
		}
	}

	s.True(columnExists, "Column 'app_contract' does not exist in the table 'convenience_inputs'")

}

func (s *InputRepositorySuite) TestCreateInputAndCheckAppContract() {
	ctx := context.Background()
	_, err := s.inputRepository.Create(ctx, convenience.AdvanceInput{
		ID:             "2222",
		Index:          2222,
		Status:         convenience.CompletionStatusUnprocessed,
		MsgSender:      common.Address{},
		Payload:        common.Hex2Bytes("0x1122"),
		BlockNumber:    1,
		BlockTimestamp: time.Now(),
		AppContract:    common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
	})

	s.NoError(err)

	input2, err := s.inputRepository.FindByID(ctx, "2222")
	s.NoError(err)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", input2.AppContract.Hex())
}

func (s *InputRepositorySuite) TearDownTest() {
	defer s.dbFactory.Cleanup()
}
