package convenience

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

type ConvenienceServiceSuite struct {
	suite.Suite
	repository *ConvenienceRepositoryImpl
	service    *ConvenienceService
}

func (s *ConvenienceServiceSuite) SetupTest() {
	db := sqlx.MustConnect("sqlite3", ":memory:")
	s.repository = &ConvenienceRepositoryImpl{
		db: *db,
	}
	err := s.repository.CreateTables()
	checkError3(s.T(), err)
	s.service = &ConvenienceService{
		repository: s.repository,
	}
}

func TestConvenienceServiceSuite(t *testing.T) {
	suite.Run(t, new(ConvenienceServiceSuite))
}

func (s *ConvenienceServiceSuite) TestCreateVoucher() {
	ctx := context.Background()
	_, err := s.service.CreateVoucher(ctx, &ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	checkError3(s.T(), err)
	count, err := s.repository.VoucherCount(ctx)
	checkError3(s.T(), err)
	s.Equal(uint64(1), count)
}

func (s *ConvenienceServiceSuite) TestFindAllVouchers() {
	ctx := context.Background()
	_, err := s.service.CreateVoucher(ctx, &ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})
	checkError3(s.T(), err)
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	checkError3(s.T(), err)
	s.Equal(1, len(vouchers))
}

func (s *ConvenienceServiceSuite) TestFindAllVouchersExecuted() {
	ctx := context.Background()
	_, err := s.repository.CreateVoucher(ctx, &ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	checkError3(s.T(), err)
	_, err = s.repository.CreateVoucher(ctx, &ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 1,
		Executed:    false,
	})
	checkError3(s.T(), err)
	field := "Executed"
	value := "true"
	byExecuted := ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	}
	filters := []*ConvenienceFilter{}
	filters = append(filters, &byExecuted)
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	checkError3(s.T(), err)
	s.Equal(1, len(vouchers))
}

func checkError3(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err.Error())
	}
}
