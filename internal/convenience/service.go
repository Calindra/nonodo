package convenience

import "context"

type ConvenienceService struct {
	repository *ConvenienceRepositoryImpl
}

func (s *ConvenienceService) CreateVoucher(
	ctx context.Context,
	voucher *ConvenienceVoucher,
) (*ConvenienceVoucher, error) {
	return s.repository.CreateVoucher(ctx, voucher)
}
