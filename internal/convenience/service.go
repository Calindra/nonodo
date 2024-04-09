package convenience

import (
	"context"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
)

type ConvenienceService struct {
	repository *repository.VoucherRepository
}

func (s *ConvenienceService) CreateVoucher(
	ctx context.Context,
	voucher *model.ConvenienceVoucher,
) (*model.ConvenienceVoucher, error) {
	return s.repository.CreateVoucher(ctx, voucher)
}

func (c *ConvenienceService) UpdateExecuted(
	ctx context.Context,
	inputIndex uint64,
	outputIndex uint64,
	executedValue bool,
) error {
	return c.repository.UpdateExecuted(
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
) ([]model.ConvenienceVoucher, error) {
	return c.repository.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}
