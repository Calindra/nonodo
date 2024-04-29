package reader

import (
	"context"
	"fmt"
	convenience "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/services"
	repos "github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader/model"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/jmoiron/sqlx"
	"log/slog"
)

type AdapterV2 struct {
	reportRepository   *repos.ReportRepository
	inputRepository    *repos.InputRepository
	convenienceService *services.ConvenienceService
}

func NewAdapterV2(
	db *sqlx.DB,
	convenienceService *services.ConvenienceService,
) Adapter {
	slog.Debug("NewAdapterV2")
	reportRepository := &repos.ReportRepository{
		Db: db,
	}
	err := reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	inputRepository := &repos.InputRepository{
		Db: db,
	}
	err = inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return AdapterV2{
		reportRepository:   reportRepository,
		inputRepository:    inputRepository,
		convenienceService: convenienceService,
	}
}

func (a AdapterV2) GetReport(reportIndex int, inputIndex int) (*model.Report, error) {
	//TODO implement me
	panic("implement me")
}

func (a AdapterV2) GetReports(first *int, last *int, after *string, before *string, inputIndex *int) (*model.ReportConnection, error) {
	//TODO implement me
	panic("implement me")
}

func (a AdapterV2) GetInputs(first *int, last *int, after *string, before *string, where *model.InputFilter) (*model.InputConnection, error) {
	//TODO implement me
	panic("implement me")
}

func (a AdapterV2) GetInput(index int) (*model.Input, error) {
	//TODO implement me
	panic("implement me")
}

func (a AdapterV2) GetNotice(noticeIndex int, inputIndex int) (*model.Notice, error) {
	ctx := context.Background()
	notice, err := a.convenienceService.FindVoucherByInputAndOutputIndex(
		ctx,
		uint64(inputIndex),
		uint64(noticeIndex),
	)
	if err != nil {
		return nil, err
	}
	if notice == nil {
		return nil, fmt.Errorf("notice not found")
	}
	return &graphql.Notice{
		Index:      noticeIndex,
		InputIndex: inputIndex,
		Payload:    notice.Payload,
		Proof:      nil,
	}, nil
}

func (a AdapterV2) GetNotices(first *int, last *int, after *string, before *string, inputIndex *int) (*model.NoticeConnection, error) {
	filters := []*convenience.ConvenienceFilter{}
	if inputIndex != nil {
		field := repos.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &convenience.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	ctx := context.Background()
	notices, err := a.convenienceService.FindAllNotices(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToNoticeConnectionV1(
		notices.Rows,
		int(notices.Offset),
		int(notices.Total),
	)
}

func (a AdapterV2) GetVoucher(voucherIndex int, inputIndex int) (*model.Voucher, error) {
	ctx := context.Background()
	voucher, err := a.convenienceService.FindVoucherByInputAndOutputIndex(
		ctx, uint64(inputIndex), uint64(voucherIndex))
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, fmt.Errorf("voucher not found")
	}
	return &graphql.Voucher{
		Index:       voucherIndex,
		InputIndex:  int(voucher.InputIndex),
		Destination: voucher.Destination.Hex(),
		Payload:     voucher.Payload,
	}, nil
}

func (a AdapterV2) GetVouchers(first *int, last *int, after *string, before *string, inputIndex *int) (*model.VoucherConnection, error) {
	filters := []*convenience.ConvenienceFilter{}
	if inputIndex != nil {
		field := repos.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &convenience.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	ctx := context.Background()
	vouchers, err := a.convenienceService.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToVoucherConnectionV1(
		vouchers.Rows,
		int(vouchers.Offset),
		int(vouchers.Total),
	)
}
