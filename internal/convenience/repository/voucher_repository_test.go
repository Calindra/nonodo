package repository

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	"github.com/lmittmann/tint"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type VoucherRepositorySuite struct {
	suite.Suite
	repository *VoucherRepository
}

func (s *VoucherRepositorySuite) SetupTest() {
	logOpts := new(tint.Options)
	logOpts.Level = slog.LevelDebug
	logOpts.AddSource = true
	logOpts.NoColor = false
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
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
	count, err := s.repository.CountVouchers(ctx, nil)
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

func (s *VoucherRepositorySuite) TestCountVoucher() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	checkError2(s, err)
	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 0,
		Executed:    false,
	})
	checkError2(s, err)
	voucher, err := s.repository.CountVouchers(ctx, nil)
	checkError2(s, err)
	s.Equal(uint64(2), voucher)

	filters := []*model.ConvenienceFilter{}
	{
		field := "Executed"
		value := "false"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	voucher, err = s.repository.CountVouchers(ctx, filters)
	checkError2(s, err)
	s.Equal(uint64(1), voucher)
}

func (s *VoucherRepositorySuite) TestWrongAddress() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	checkError2(s, err)
	filters := []*model.ConvenienceFilter{}
	{
		field := "Destination"
		value := "0xError"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	vouchers, err := s.repository.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	if err == nil {
		s.Fail("where is the error?")
	}
	s.Equal("wrong address value", err.Error())
	s.Equal(0, len(vouchers))
}

func checkError2(s *VoucherRepositorySuite, err error) {
	if err != nil {
		s.T().Fatal(err.Error())
	}
}
