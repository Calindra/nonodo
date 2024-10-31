package services

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/stretchr/testify/suite"
)

type ConvenienceServiceSuite struct {
	suite.Suite
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
	reportRepository  *repository.ReportRepository
	service           *ConvenienceService
}

func (s *ConvenienceServiceSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	db := sqlx.MustConnect("sqlite3", ":memory:")
	outputRepository := repository.OutputRepository{
		Db: *db,
	}
	s.voucherRepository = &repository.VoucherRepository{
		Db:               *db,
		OutputRepository: outputRepository,
	}
	err := s.voucherRepository.CreateTables()
	s.NoError(err)

	s.noticeRepository = &repository.NoticeRepository{
		Db:               *db,
		OutputRepository: outputRepository,
	}
	err = s.noticeRepository.CreateTables()
	s.NoError(err)

	s.reportRepository = &repository.ReportRepository{
		Db: db,
	}
	err = s.reportRepository.CreateTables()
	s.NoError(err)

	s.service = &ConvenienceService{
		voucherRepository: s.voucherRepository,
		noticeRepository:  s.noticeRepository,
		ReportRepository:  s.reportRepository,
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
	s.NoError(err)
	count, err := s.voucherRepository.Count(ctx, nil)
	s.NoError(err)
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
	s.NoError(err)
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(1, len(vouchers.Rows))
}

func (s *ConvenienceServiceSuite) TestFindAllVouchersExecuted() {
	ctx := context.Background()
	_, err := s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  1,
		OutputIndex: 2,
		Executed:    false,
	})
	s.NoError(err)
	_, err = s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 1,
		Executed:    true,
	})
	s.NoError(err)
	_, err = s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29"),
		Payload:     "0x0011",
		InputIndex:  3,
		OutputIndex: 1,
		Executed:    false,
	})
	s.NoError(err)
	field := "Executed"
	value := "true"
	byExecuted := model.ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	}
	filters := []*model.ConvenienceFilter{}
	filters = append(filters, &byExecuted)
	vouchers, err := s.service.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	s.Equal(1, len(vouchers.Rows))
	s.Equal(2, int(vouchers.Rows[0].InputIndex))
}

func (s *ConvenienceServiceSuite) TestFindAllVouchersByDestination() {
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
		Destination: common.HexToAddress("0xf795b3D15D47ac1c61BEf4Cc6469EBb2454C6a9b"),
		Payload:     "0x0011",
		InputIndex:  2,
		OutputIndex: 1,
		Executed:    true,
	})
	s.NoError(err)
	_, err = s.voucherRepository.CreateVoucher(ctx, &model.ConvenienceVoucher{
		Destination: common.HexToAddress("0xf795b3D15D47ac1c61BEf4Cc6469EBb2454C6a9b"),
		Payload:     "0x0011",
		InputIndex:  3,
		OutputIndex: 1,
		Executed:    false,
	})
	s.NoError(err)
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
	s.NoError(err)
	s.Equal(1, len(vouchers.Rows))
	s.Equal(2, int(vouchers.Rows[0].InputIndex))
}

func (s *ConvenienceServiceSuite) XTestCreateVoucherIdempotency() {
	// we need a better way to do this
	ctx := context.Background()
	_, err := s.service.CreateVoucher(ctx, &model.ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err := s.voucherRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))

	if err != nil {
		panic(err)
	}

	_, err = s.service.CreateVoucher(ctx, &model.ConvenienceVoucher{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err = s.voucherRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))

	if err != nil {
		panic(err)
	}
}

func (s *ConvenienceServiceSuite) XTestCreateNoticeIdempotency() {
	// we need a better way to do this
	ctx := context.Background()
	_, err := s.service.CreateNotice(ctx, &model.ConvenienceNotice{
		InputIndex:  1,
		OutputIndex: 2,
	})
	s.NoError(err)
	count, err := s.noticeRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))

	if err != nil {
		panic(err)
	}

	_, err = s.service.CreateNotice(ctx, &model.ConvenienceNotice{
		InputIndex:  1,
		OutputIndex: 2,
		Payload:     "1122",
	})
	s.NoError(err)
	count, err = s.noticeRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
	notice, err := s.service.FindNoticeByInputAndOutputIndex(ctx, 1, 2)
	s.NoError(err)
	s.NotNil(notice)
	s.Equal("1122", notice.Payload)
}

func (s *ConvenienceServiceSuite) XTestCreateReportIdempotency() {
	// we need a better way to do this
	ctx := context.Background()
	_, err := s.service.CreateReport(ctx, &model.Report{
		InputIndex: 1,
		Index:      2,
	})
	s.NoError(err)
	count, err := s.reportRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))

	if err != nil {
		panic(err)
	}

	_, err = s.service.CreateReport(ctx, &model.Report{
		InputIndex: 1,
		Index:      2,
		Payload:    common.Hex2Bytes("1122"),
	})
	s.NoError(err)
	count, err = s.reportRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(count))
}
