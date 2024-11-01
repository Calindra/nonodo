package loaders

import (
	"context"
	"net/http"
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
	reportRepository *repository.ReportRepository
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

// Loaders wrap your data loaders to inject via middleware
type Loaders struct {
	ReportLoader  *dataloadgen.Loader[string, *commons.PageResult[cModel.Report]]
	VoucherLoader *dataloadgen.Loader[string, *commons.PageResult[cModel.ConvenienceVoucher]]
}

// NewLoaders instantiates data loaders for the middleware
func NewLoaders(reportRepository *repository.ReportRepository) *Loaders {
	// define the data loader
	ur := &reportReader{reportRepository: reportRepository}
	return &Loaders{
		ReportLoader: dataloadgen.NewLoader(
			ur.getReports,
			dataloadgen.WithWait(time.Millisecond),
		),
	}
}

// Middleware injects data loaders into the context
func Middleware(reportRepository *repository.ReportRepository, next http.Handler) http.Handler {
	// return a middleware that injects the loader to the request context
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loader := NewLoaders(reportRepository)
		r = r.WithContext(context.WithValue(r.Context(), LoadersKey, loader))
		next.ServeHTTP(w, r)
	})
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
