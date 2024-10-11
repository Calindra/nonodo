package reader

import (
	"context"

	graphql "github.com/calindra/nonodo/internal/reader/model"
)

type Adapter interface {
	GetReport(reportIndex int) (*graphql.Report, error)

	GetReports(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetInputs(
		ctx context.Context,
		first *int, last *int, after *string, before *string, where *graphql.InputFilter,
	) (*graphql.InputConnection, error)

	GetInput(id string) (*graphql.Input, error)
	GetInputByIndex(inputIndex int) (*graphql.Input, error)

	GetNotice(outputIndex int) (*graphql.Notice, error)

	GetNotices(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.NoticeConnection, error)

	GetVoucher(outputIndex int) (*graphql.Voucher, error)

	GetVouchers(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.VoucherConnection, error)

	GetProof(ctx context.Context, inputIndex, outputIndex int) (*graphql.Proof, error)
}
