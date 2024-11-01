package repository

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

type VoucherRepositorySuite struct {
	suite.Suite
	voucherRepository *VoucherRepository
	dbFactory         *commons.DbFactory
}

func (s *VoucherRepositorySuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	s.dbFactory = commons.NewDbFactory()
	db := s.dbFactory.CreateDb("voucher.sqlite3")
	outputRepository := OutputRepository{*db}
	s.voucherRepository = &VoucherRepository{
		Db: *db, OutputRepository: outputRepository,
	}
	noticeRepository := NoticeRepository{*db, outputRepository, false}
	err := noticeRepository.CreateTables()
	s.NoError(err)
	err = s.voucherRepository.CreateTables()
	s.NoError(err)
}

func (s *VoucherRepositorySuite) TearDownTest() {
	s.dbFactory.Cleanup()
}

func TestConvenienceRepositorySuite(t *testing.T) {
	suite.Run(t, new(VoucherRepositorySuite))
}

func (s *VoucherRepositorySuite) TestCreateVoucher() {
	ctx := context.Background()
	_, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err := s.voucherRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}

func (s *VoucherRepositorySuite) TestFindVoucher() {
	ctx := context.Background()
	appAddress := common.HexToAddress(devnet.ApplicationAddress)
	voucherSaved, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
		AppContract: appAddress,
	})
	s.NoError(err)
	voucher, err := s.voucherRepository.FindVoucherByInputAndOutputIndex(ctx, voucherSaved.InputIndex, voucherSaved.OutputIndex)
	s.NoError(err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal(appAddress.Hex(), voucher.AppContract.Hex())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(1, int(voucher.InputIndex))
	s.Equal(2, int(voucher.OutputIndex))
	s.Equal(false, voucher.Executed)
}

func (s *VoucherRepositorySuite) TestFindVoucherExecuted() {
	ctx := context.Background()
	_, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination:          common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:              "0x0011",
		InputIndex:           1,
		OutputIndex:          2,
		Executed:             true,
		OutputHashesSiblings: `["0x01","0x02"]`,
	})
	s.NoError(err)
	voucher, err := s.voucherRepository.FindVoucherByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	fmt.Println(voucher.Destination)
	s.Equal("0x26A61aF89053c847B4bd5084E2caFe7211874a29", voucher.Destination.String())
	s.Equal("0x0011", voucher.Payload)
	s.Equal(1, int(voucher.InputIndex))
	s.Equal(2, int(voucher.OutputIndex))
	s.Equal(true, voucher.Executed)
	s.Equal(`["0x01","0x02"]`, voucher.OutputHashesSiblings)
}

func (s *VoucherRepositorySuite) TestCountVoucher() {
	ctx := context.Background()
	_, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	s.NoError(err)
	_, err = s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 0,
		Executed:    false,
	})
	s.NoError(err)
	total, err := s.voucherRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(2, int(total))

	filters := []*model.ConvenienceFilter{}
	{
		field := "Executed"
		value := "false"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	total, err = s.voucherRepository.Count(ctx, filters)
	s.NoError(err)
	s.Equal(1, int(total))
}

func (s *VoucherRepositorySuite) TestPagination() {
	destination := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	ctx := context.Background()
	for i := 0; i < 30; i++ {
		_, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
			Destination: destination,
			Payload:     "0x0011",
			InputIndex:  uint64(i),
			OutputIndex: 0,
			Executed:    false,
		})
		s.NoError(err)
	}

	total, err := s.voucherRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(30, int(total))

	filters := []*model.ConvenienceFilter{}
	{
		field := "Executed"
		value := "false"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	first := 10
	vouchers, err := s.voucherRepository.FindAllVouchers(ctx, &first, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(0, int(vouchers.Rows[0].InputIndex))
	s.Equal(9, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))

	after := commons.EncodeCursor(10)
	vouchers, err = s.voucherRepository.FindAllVouchers(ctx, &first, nil, &after, nil, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(11, int(vouchers.Rows[0].InputIndex))
	s.Equal(20, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))

	last := 10
	vouchers, err = s.voucherRepository.FindAllVouchers(ctx, nil, &last, nil, nil, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(20, int(vouchers.Rows[0].InputIndex))
	s.Equal(29, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))

	before := commons.EncodeCursor(20)
	vouchers, err = s.voucherRepository.FindAllVouchers(ctx, nil, &last, nil, &before, filters)
	s.NoError(err)
	s.Equal(10, len(vouchers.Rows))
	s.Equal(10, int(vouchers.Rows[0].InputIndex))
	s.Equal(19, int(vouchers.Rows[len(vouchers.Rows)-1].InputIndex))
}

func (s *VoucherRepositorySuite) TestWrongAddress() {
	ctx := context.Background()
	_, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    true,
	})
	s.NoError(err)
	filters := []*model.ConvenienceFilter{}
	{
		field := "Destination"
		value := "0xError"
		filters = append(filters, &model.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	_, err = s.voucherRepository.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	if err == nil {
		s.Fail("where is the error?")
	}
	s.Equal("wrong address value", err.Error())
}

func (s *VoucherRepositorySuite) TestBatchFindAllVouchers() {
	ctx := context.Background()
	appContract := common.HexToAddress(devnet.ApplicationAddress)
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			_, err := s.voucherRepository.CreateVoucher(
				ctx,
				&cModel.ConvenienceVoucher{
					Destination:          common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
					Payload:              "0x1122",
					InputIndex:           uint64(i),
					OutputIndex:          uint64(j),
					Executed:             false,
					Value:                "0x1234",
					AppContract:          appContract,
					OutputHashesSiblings: `["0x01","0x02"]`,
				})
			s.Require().NoError(err)
		}
	}

	filters := []*BatchFilterItem{
		{
			AppContract: &appContract,
			InputIndex:  0,
		},
	}
	results, err := s.voucherRepository.BatchFindAllByInputIndexAndAppContract(
		ctx, filters,
	)
	s.Require().Equal(0, len(err))
	s.Equal(1, len(results))
	s.Equal(4, len(results[0].Rows))
	s.Equal(4, int(results[0].Total))
}
