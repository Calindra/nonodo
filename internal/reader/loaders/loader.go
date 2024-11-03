package loaders

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/vikstrous/dataloadgen"
)

type ctxKey string

const (
	LoadersKey = ctxKey("dataLoaders")
)

// reportReader reads Users from a database
type reportReader struct {
	reportRepository  *repository.ReportRepository
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
}

// getReports implements a batch function that can retrieve many users by ID,
// for use in a dataloader
func (u *reportReader) getReports(ctx context.Context, reportsKeys []string) ([]*commons.PageResult[cModel.Report], []error) {
	errors := []error{}
	filters := []*repository.BatchFilterItem{}
	for _, reportKey := range reportsKeys {
		aux := strings.Split(reportKey, "|")
		appContract := common.HexToAddress(aux[0])
		inputIndex, err := strconv.Atoi(aux[1])
		if err != nil {
			return nil, errors
		}
		filter := repository.BatchFilterItem{
			AppContract: &appContract,
			InputIndex:  inputIndex,
		}
		filters = append(filters, &filter)
	}

	return u.reportRepository.BatchFindAllByInputIndexAndAppContract(ctx, filters)
}

func (u *reportReader) getVouchers(ctx context.Context, voucherKeys []string) ([]*commons.PageResult[cModel.ConvenienceVoucher], []error) {
	errors := []error{}
	filters := []*repository.BatchFilterItem{}
	for _, reportKey := range voucherKeys {
		aux := strings.Split(reportKey, "|")
		appContract := common.HexToAddress(aux[0])
		inputIndex, err := strconv.Atoi(aux[1])
		if err != nil {
			return nil, errors
		}
		filter := repository.BatchFilterItem{
			AppContract: &appContract,
			InputIndex:  inputIndex,
		}
		filters = append(filters, &filter)
	}

	return u.voucherRepository.BatchFindAllByInputIndexAndAppContract(ctx, filters)
}

func (u reportReader) getNotices(ctx context.Context, noticesKeys []string) ([]*commons.PageResult[cModel.ConvenienceNotice], []error) {
	errors := []error{}
	filters := []*repository.BatchFilterItemForNotice{}
	for _, noticeKey := range noticesKeys {
		aux := strings.Split(noticeKey, "|")
		appContract := common.HexToAddress(aux[0])
		inputIndex, err := strconv.Atoi(aux[1])

		if err != nil {
			return nil, errors
		}

		filter := repository.BatchFilterItemForNotice{
			AppContract: appContract.Hex(),
			InputIndex:  inputIndex,
		}
		filters = append(filters, &filter)
	}
	return u.noticeRepository.BatchFindAllNoticesByInputIndexAndAppContract(ctx, filters)
}

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	ReportLoader  *dataloadgen.Loader[string, *commons.PageResult[cModel.Report]]
	VoucherLoader *dataloadgen.Loader[string, *commons.PageResult[cModel.ConvenienceVoucher]]
	NoticeLoader  *dataloadgen.Loader[string, *commons.PageResult[cModel.ConvenienceNotice]]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(reportRepository *repository.ReportRepository, voucherRepository *repository.VoucherRepository, noticeRepository *repository.NoticeRepository) *Loaders {
	// define the data loader
	ur := &reportReader{reportRepository: reportRepository, voucherRepository: voucherRepository, noticeRepository: noticeRepository}
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
