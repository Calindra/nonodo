package convenience

import (
	"context"
)

type ConvenienceService struct {
	repository *ConvenienceRepositoryImpl
}

func (s *ConvenienceService) CreateVoucher(
	ctx context.Context,
	voucher *ConvenienceVoucher,
) (*ConvenienceVoucher, error) {
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
	filter []*ConvenienceFilter,
) ([]ConvenienceVoucher, error) {
	return c.repository.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filter,
	)
}
