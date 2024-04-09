package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type VoucherRepositorySuite struct {
	suite.Suite
	repository *VoucherRepository
}

func (s *VoucherRepositorySuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &VoucherRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	checkError2(s, err)
}

func TestConvenienceRepositorySuite(t *testing.T) {
	suite.Run(t, new(VoucherRepositorySuite))
}

func (s *VoucherRepositorySuite) TestCreateVoucher() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	checkError2(s, err)
	count, err := s.repository.VoucherCount(ctx)
	checkError2(s, err)
	s.Equal(uint64(1), count)
}

func (s *VoucherRepositorySuite) TestFindVoucher() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})
	checkError2(s, err)
	voucher, err := s.repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	checkError2(s, err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(uint64(1), voucher.InputIndex)
	s.Equal(uint64(2), voucher.OutputIndex)
	s.Equal(false, voucher.Executed)
}

func (s *VoucherRepositorySuite) TestFindVoucherExecuted() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	checkError2(s, err)
	voucher, err := s.repository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	checkError2(s, err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(uint64(1), voucher.InputIndex)
	s.Equal(uint64(2), voucher.OutputIndex)
	s.Equal(true, voucher.Executed)
}

func (s *VoucherRepositorySuite) TestTransformToQuery() {

}

func checkError2(s *VoucherRepositorySuite, err error) {
	if err != nil {
		s.T().Fatal(err.Error())
	}
}
