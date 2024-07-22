package reader

import (
	"context"

	"github.com/calindra/nonodo/internal/convenience/model"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
)

type Adapter interface {
	GetReport(reportIndex int, inputIndex int) (*graphql.Report, error)

	GetReports(
		ctx context.Context,
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.ReportConnection, error)

	GetInputs(
		ctx context.Context,
		first *int, last *int, after *string, before *string, where *graphql.InputFilter,
	) (*graphql.InputConnection, error)

	GetInput(index int) (*graphql.Input, error)

	GetNotice(noticeIndex int, inputIndex int) (*graphql.Notice, error)

	GetNotices(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.NoticeConnection, error)

	GetVoucher(voucherIndex int, inputIndex int) (*graphql.Voucher, error)

	GetVouchers(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*graphql.VoucherConnection, error)

	GetProof(ctx context.Context, inputIndex, outputIndex int) (*graphql.Proof, error)

	RetrieveDestination(output model.OutputEdge) (common.Address, error)
}
