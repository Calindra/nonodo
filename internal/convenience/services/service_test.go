package services

import (
	"context"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type ConvenienceServiceSuite struct {
	suite.Suite
	repository *repository.VoucherRepository
	service    *ConvenienceService
}

func (s *ConvenienceServiceSuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &repository.VoucherRepository{
		Db: *db,
	}
	err := s.repository.CreateTables()
	checkError3(s.T(), err)
	s.service = &ConvenienceService{
		voucherRepository: s.repository,
	}
}

func TestConvenienceServiceSuite(t *testing.T) {
	suite.Run(t, new(ConvenienceServiceSuite))
}

func (s *ConvenienceServiceSuite) TestCreateVoucher() {
	ctx := context.Background()
	_, err := s.service.CreateVoucher(ctx, &model.ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	checkError3(s.T(), err)
	count, err := s.repository.Count(ctx, nil)
	checkError3(s.T(), err)
	s.Equal(1, int(count))
}

func (s *ConvenienceServiceSuite) TestFindAllVouchers() {
	ctx := context.Background()
	_, err := s.service.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})
	checkError3(s.T(), err)
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	checkError3(s.T(), err)
	s.Equal(1, len(vouchers.Rows))
}

func (s *ConvenienceServiceSuite) TestFindAllVouchersExecuted() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})
	checkError3(s.T(), err)
	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 1,
		Executed:    true,
	})
	checkError3(s.T(), err)
	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  3,
		OutputIndex: 1,
		Executed:    false,
	})
	checkError3(s.T(), err)
	field := "Executed"
	value := "true"
	byExecuted := model.ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	}
	filters := []*model.ConvenienceFilter{}
	filters = append(filters, &byExecuted)
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	checkError3(s.T(), err)
	s.Equal(1, len(vouchers.Rows))
	s.Equal(2, int(vouchers.Rows[0].InputIndex))
}

func (s *ConvenienceServiceSuite) TestFindAllVouchersByDestination() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	checkError3(s.T(), err)
	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0xf795b3D15D47ac1c61BEf4Cc6469EBb2454C6a9b"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 1,
		Executed:    true,
	})
	checkError3(s.T(), err)
	_, err = s.repository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0xf795b3D15D47ac1c61BEf4Cc6469EBb2454C6a9b"),
		Payload:     "0x0011",
		InputIndex:  3,
		OutputIndex: 1,
		Executed:    false,
	})
	checkError3(s.T(), err)
	filters := []*model.ConvenienceFilter{}
	{
		field := "Destination"
		value := "0xf795b3D15D47ac1c61BEf4Cc6469EBb2454C6a9b"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	{
		field := "Executed"
		value := "true"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	checkError3(s.T(), err)
	s.Equal(1, len(vouchers.Rows))
	s.Equal(2, int(vouchers.Rows[0].InputIndex))
}

func checkError3(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err.Error())
	}
}
