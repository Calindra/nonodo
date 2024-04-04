package convenience

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type ConvenienceRepositorySuite struct {
	suite.Suite
	repository *ConvenienceRepositoryImpl
}

func (s *ConvenienceRepositorySuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &ConvenienceRepositoryImpl{
		db: *db,
	}
	err := s.repository.CreateTables()
	checkError2(s, err)
}

func TestConvenienceRepositorySuite(t *testing.T) {
	suite.Run(t, new(ConvenienceRepositorySuite))
}

func (s *ConvenienceRepositorySuite) TestCreateVoucher() {
	ctx := context.Background()
	s.repository.CreateVoucher(ctx, &ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	count, err := s.repository.VoucherCount(ctx)
	checkError2(s, err)
	s.Equal(uint64(1), count)
}

func (s *ConvenienceRepositorySuite) TestFindVoucher() {
	ctx := context.Background()
	s.repository.CreateVoucher(ctx, &ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})
	voucher, err := s.repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	checkError2(s, err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(uint64(1), voucher.InputIndex)
	s.Equal(uint64(2), voucher.OutputIndex)
	s.Equal(false, voucher.Executed)
}

func (s *ConvenienceRepositorySuite) TestFindVoucherExecuted() {
	ctx := context.Background()
	s.repository.CreateVoucher(ctx, &ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	voucher, err := s.repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	checkError2(s, err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(uint64(1), voucher.InputIndex)
	s.Equal(uint64(2), voucher.OutputIndex)
	s.Equal(true, voucher.Executed)
}

func checkError2(s *ConvenienceRepositorySuite, err error) {
	if err != nil {
		s.T().Fatal(err.Error())
	}
}
