package loaders

import (
	"context"
	"strconv"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

// dataReader reads Users from a database
type dataReader struct {
	reportRepository  *repository.ReportRepository
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
	inputRepository   *repository.InputRepository
}

// getReports implements a batch function that can retrieve many users by ID,
// for use in a dataloader
func (u *dataReader) getReports(ctx context.Context, reportsKeys []string) ([]*commons.PageResult[cModel.Report], []error) {
	filters, errors := buildBatchFilters(reportsKeys, func(appContract common.Address, inputIndex int) *repository.BatchFilterItem {
		return &repository.BatchFilterItem{
			AppContract: &appContract,
			InputIndex:  inputIndex,
		}
	})
	if errors != nil {
		return nil, errors
	}

	return u.reportRepository.BatchFindAllByInputIndexAndAppContract(ctx, filters)
}

func (u *dataReader) getVouchers(ctx context.Context, voucherKeys []string) ([]*commons.PageResult[cModel.ConvenienceVoucher], []error) {
	filters, errors := buildBatchFilters(voucherKeys, func(appContract common.Address, inputIndex int) *repository.BatchFilterItem {
		return &repository.BatchFilterItem{
			AppContract: &appContract,
			InputIndex:  inputIndex,
		}
	})
	if errors != nil {
		return nil, errors
	}

	return u.voucherRepository.BatchFindAllByInputIndexAndAppContract(ctx, filters)
}

func (u *dataReader) getNotices(ctx context.Context, noticesKeys []string) ([]*commons.PageResult[cModel.ConvenienceNotice], []error) {
	filters, errors := buildBatchFilters(noticesKeys, func(appContract common.Address, inputIndex int) *repository.BatchFilterItemForNotice {
		return &repository.BatchFilterItemForNotice{
			AppContract: appContract.Hex(),
			InputIndex:  inputIndex,
		}
	})
	if errors != nil {
		return nil, errors
	}

	return u.noticeRepository.BatchFindAllNoticesByInputIndexAndAppContract(ctx, filters)
}

func (u *dataReader) getInputs(ctx context.Context, inputsKeys []string) ([]*cModel.AdvanceInput, []error) {
	filters, errors := buildBatchFilters(inputsKeys, func(appContract common.Address, inputIndex int) *repository.BatchFilterItem {
		return &repository.BatchFilterItem{
			AppContract: &appContract,
			InputIndex:  inputIndex,
		}
	})
	if errors != nil {
		return nil, errors
	}

	return u.inputRepository.BatchFindInputByInputIndexAndAppContract(ctx, filters)
}

func buildBatchFilters[T any](keys []string, filterFunc func(appContract common.Address, inputIndex int) T) ([]T, []error) {
	errors := []error{}
	filters := []T{}

	for _, key := range keys {
		aux := strings.Split(key, "|")
		appContract := common.HexToAddress(aux[0])
		inputIndex, err := strconv.Atoi(aux[1])
		if err != nil {
			return nil, errors
		}

		filter := filterFunc(appContract, inputIndex)
		filters = append(filters, filter)
	}

	return filters, nil
}
