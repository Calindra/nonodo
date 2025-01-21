package loaders

import (
	"context"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/vikstrous/dataloadgen"
)

type ctxKey string

const (
	LoadersKey = ctxKey("dataLoaders")
)

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	ReportLoader  *dataloadgen.Loader[string, *commons.PageResult[cModel.Report]]
	VoucherLoader *dataloadgen.Loader[string, *commons.PageResult[cModel.ConvenienceVoucher]]
	NoticeLoader  *dataloadgen.Loader[string, *commons.PageResult[cModel.ConvenienceNotice]]
	InputLoader   *dataloadgen.Loader[string, *cModel.AdvanceInput]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(
	reportRepository *repository.ReportRepository,
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	inputRepository *repository.InputRepository,
) *Loaders {
	// define the data loader
	ur := &dataReader{
		reportRepository:  reportRepository,
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
		inputRepository:   inputRepository,
	}
	return &Loaders{
		ReportLoader: dataloadgen.NewLoader(
			ur.getReports,
			dataloadgen.WithWait(time.Millisecond),
		),
		VoucherLoader: dataloadgen.NewLoader(
			ur.getVouchers,
			dataloadgen.WithWait(time.Millisecond),
		),
		NoticeLoader: dataloadgen.NewLoader(
			ur.getNotices,
			dataloadgen.WithWait(time.Millisecond),
		),
		InputLoader: dataloadgen.NewLoader(
			ur.getInputs,
			dataloadgen.WithWait(time.Millisecond),
		),
	}
}

// For returns the dataloader for a given context
func For(ctx context.Context) *Loaders {
	aux := ctx.Value(LoadersKey)
	if aux == nil {
		return nil
	}
	return aux.(*Loaders)
}

// GetReports returns single reports by reportsKey efficiently
func GetReports(ctx context.Context, reportsKey string) (*commons.PageResult[cModel.Report], error) {
	loaders := For(ctx)
	return loaders.ReportLoader.Load(ctx, reportsKey)
}

// GetMayReports returns many reports by reportsKeys efficiently
func GetMayReports(ctx context.Context, reportsKeys []string) ([]*commons.PageResult[cModel.Report], error) {
	loaders := For(ctx)
	return loaders.ReportLoader.LoadAll(ctx, reportsKeys)
}
