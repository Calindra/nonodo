package services

import (
	"context"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
)

type ConvenienceService struct {
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
	inputRepository   *repository.InputRepository
}

func NewConvenienceService(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	inputRepository *repository.InputRepository,
) *ConvenienceService {
	return &ConvenienceService{
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
		inputRepository:   inputRepository,
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
	noticeInDb, err := s.inputRepository.FindByIndex(ctx, input.Index)

	if err != nil {
		return nil, err
	}

	if noticeInDb != nil {
		return s.inputRepository.Update(ctx, *input)
	}
	return s.inputRepository.Create(ctx, *input)
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

func (c *ConvenienceService) FindVoucherByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceVoucher, error) {
	return c.voucherRepository.FindVoucherByInputAndOutputIndex(
		ctx, inputIndex, outputIndex,
	)
}

func (c *ConvenienceService) FindNoticeByInputAndOutputIndex(
	ctx context.Context, inputIndex uint64, outputIndex uint64,
) (*model.ConvenienceNotice, error) {
	return c.noticeRepository.FindByInputAndOutputIndex(
		ctx, inputIndex, outputIndex,
	)
}

func (c *ConvenienceService) FindInputByIndex(
	ctx context.Context, index int,
) (*model.AdvanceInput, error) {
	return c.inputRepository.FindByIndex(ctx, index)
}
