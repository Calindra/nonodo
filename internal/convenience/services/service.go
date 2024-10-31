package services

import (
	"context"
	"log/slog"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type ConvenienceService struct {
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
	inputRepository   *repository.InputRepository
	ReportRepository  *repository.ReportRepository
}

func NewConvenienceService(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	inputRepository *repository.InputRepository,
	reportRepository *repository.ReportRepository,
) *ConvenienceService {
	return &ConvenienceService{
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
		inputRepository:   inputRepository,
		ReportRepository:  reportRepository,
	}
}

func (s *ConvenienceService) CreateVoucher1(
	ctx context.Context,
	voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	return s.voucherRepository.CreateVoucher(ctx, voucher)
}

func (s *ConvenienceService) CreateNotice(
	ctx context.Context,
	notice *model.ConvenienceNotice,
) (*model.ConvenienceNotice, error) {
	noticeInDb, err := s.noticeRepository.FindByInputAndOutputIndex(
		ctx, notice.InputIndex, notice.OutputIndex,
	)
	if err != nil {
		return nil, err
	}

	if noticeInDb != nil {
		return s.noticeRepository.Update(ctx, notice)
	}
	return s.noticeRepository.Create(ctx, notice)
}

func (s *ConvenienceService) CreateVoucher(
	ctx context.Context,
	voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {

	voucherInDb, err := s.voucherRepository.FindVoucherByInputAndOutputIndex(
		ctx, voucher.InputIndex,
		voucher.OutputIndex,
	)

	if err != nil {
		return nil, err
	}

	if voucherInDb != nil {
		return s.voucherRepository.UpdateVoucher(ctx, voucher)
	}

	return s.voucherRepository.CreateVoucher(ctx, voucher)
}

func (s *ConvenienceService) CreateInput(
	ctx context.Context,
	input *model.AdvanceInput,
) (*model.AdvanceInput, error) {

	inputInDb, err := s.inputRepository.FindByIDAndAppContract(ctx, input.ID, &input.AppContract)

	if err != nil {
		return nil, err
	}

	if inputInDb != nil {
		return s.inputRepository.Update(ctx, *input)
	}
	return s.inputRepository.Create(ctx, *input)
}

func (s *ConvenienceService) CreateReport(
	ctx context.Context,
	report *model.Report,
) (*model.Report, error) {
	reportInDb, err := s.ReportRepository.FindByInputAndOutputIndex(ctx,
		uint64(report.InputIndex),
		uint64(report.Index),
	)
	if err != nil {
		return nil, err
	}

	if reportInDb != nil {
		slog.Debug("Report exist",
			"inputIndex", report.InputIndex,
			"outputIndex", report.Index,
		)
		return s.ReportRepository.Update(ctx, *reportInDb)
	}
	reportCreated, err := s.ReportRepository.CreateReport(ctx, *report)
	if err != nil {
		return nil, err
	}
	return &reportCreated, err
}

func (c *ConvenienceService) UpdateExecuted(
	ctx context.Context,
	inputIndex uint64,
	outputIndex uint64,
	executedValue bool,
) error {
	return c.voucherRepository.UpdateExecuted(
		ctx,
		inputIndex,
		outputIndex,
		executedValue,
	)
}

func (c *ConvenienceService) FindAllVouchers(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceVoucher], error) {
	return c.voucherRepository.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ConvenienceService) FindAllNotices(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.ConvenienceNotice], error) {
	return c.noticeRepository.FindAllNotices(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ConvenienceService) FindAllInputs(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	filter []*model.ConvenienceFilter,
) (*commons.PageResult[model.AdvanceInput], error) {
	return c.inputRepository.FindAll(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}

func (c *ConvenienceService) FindAllByInputIndex(
	ctx context.Context,
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*commons.PageResult[model.Report], error) {
	return c.ReportRepository.FindAllByInputIndex(
		ctx,
		first,
		last,
		after,
		before,
		inputIndex,
	)
}

func (c *ConvenienceService) FindVoucherByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
) (*model.ConvenienceVoucher, error) {
	return c.voucherRepository.FindVoucherByOutputIndexAndAppContract(
		ctx, outputIndex, appContract,
	)
}

func (c *ConvenienceService) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceVoucher, error) {
	return c.voucherRepository.FindVoucherByInputAndOutputIndex(
		ctx, inputIndex, outputIndex,
	)
}

func (c *ConvenienceService) FindNoticeByOutputIndexAndAppContract(
	ctx context.Context, outputIndex uint64,
	appContract *common.Address,
) (*model.ConvenienceNotice, error) {
	return c.noticeRepository.FindNoticeByOutputIndexAndAppContract(
		ctx, outputIndex, appContract,
	)
}

func (c *ConvenienceService) FindNoticeByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	return c.noticeRepository.FindByInputAndOutputIndex(
		ctx, inputIndex, outputIndex,
	)
}
