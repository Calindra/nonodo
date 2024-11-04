package reader

import (
	"context"

	graphql "github.com/calindra/nonodo/internal/reader/model"
)

type Adapter interface {
	GetReport(
		ctx context.Context,
		reportIndex int,
	) (*graphql.Report, error)

	GetReports(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetAllReportsByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetInputs(
		ctx context.Context,
		first *int, last *int, after *string, before *string, where *graphql.InputFilter,
	) (*graphql.InputConnection, error)

	GetInput(
		ctx context.Context,
		id string,
	) (*graphql.Input, error)
	GetInputByIndex(
		ctx context.Context,
		inputIndex int,
	) (*graphql.Input, error)

	GetNotice(
		ctx context.Context,
		outputIndex int,
	) (*graphql.Notice, error)

	GetNotices(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.NoticeConnection, error)

	GetVoucher(
		ctx context.Context,
		outputIndex int) (*graphql.Voucher, error)

	GetVouchers(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
		filter []*graphql.ConvenientFilter,
	) (*graphql.VoucherConnection, error)

	GetAllVouchersByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.VoucherConnection, error)

	GetAllNoticesByInputIndex(
		ctx context.Context,
		inputIndex *int,
	) (*graphql.Connection[*graphql.Notice], error)
}
