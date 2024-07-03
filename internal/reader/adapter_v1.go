package reader

import (
	"context"
	"fmt"
	"log/slog"

	convenience "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
	services "github.com/calindra/nonodo/internal/convenience/services"
	repos "github.com/calindra/nonodo/internal/model"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type AdapterV1 struct {
	reportRepository   *repos.ReportRepository
	inputRepository    *cRepos.InputRepository
	convenienceService *services.ConvenienceService
}

// GetProof implements Adapter.
func (a AdapterV1) GetProof(ctx context.Context, inputIndex int, outputIndex int) (*graphql.Proof, error) {
	// nonodo v1 does not have proofs
	return nil, fmt.Errorf("proofs are not supported in nonodo v1")
}

func NewAdapterV1(
	db *sqlx.DB,
	convenienceService *services.ConvenienceService,
) Adapter {
	slog.Debug("NewAdapterV1")
	reportRepository := &repos.ReportRepository{
		Db: db,
	}
	err := reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	inputRepository := &cRepos.InputRepository{
		Db: db,
	}
	err = inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return AdapterV1{
		reportRepository:   reportRepository,
		inputRepository:    inputRepository,
		convenienceService: convenienceService,
	}
}

func (a AdapterV1) GetNotices(
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*graphql.Connection[*graphql.Notice], error) {
	filters := []*convenience.ConvenienceFilter{}
	if inputIndex != nil {
		field := repos.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &convenience.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	ctx := context.Background()
	notices, err := a.convenienceService.FindAllNotices(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToNoticeConnectionV1(
		notices.Rows,
		int(notices.Offset),
		int(notices.Total),
	)
}

func (a AdapterV1) GetVoucher(voucherIndex int, inputIndex int) (*graphql.Voucher, error) {
	ctx := context.Background()
	voucher, err := a.convenienceService.FindVoucherByInputAndOutputIndex(
		ctx, uint64(inputIndex), uint64(voucherIndex))
	if err != nil {
		return nil, err
	}
	if voucher == nil {
		return nil, fmt.Errorf("voucher not found")
	}
	return &graphql.Voucher{
		Index:       voucherIndex,
		InputIndex:  int(voucher.InputIndex),
		Destination: voucher.Destination.Hex(),
		Payload:     voucher.Payload,
	}, nil
}

func (a AdapterV1) GetVouchers(
	first *int,
	last *int,
	after *string,
	before *string,
	inputIndex *int,
) (*graphql.Connection[*graphql.Voucher], error) {
	filters := []*convenience.ConvenienceFilter{}
	if inputIndex != nil {
		field := repos.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &convenience.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	ctx := context.Background()
	vouchers, err := a.convenienceService.FindAllVouchers(
		ctx,
		first,
		last,
		after,
		before,
		filters,
	)
	if err != nil {
		return nil, err
	}
	return graphql.ConvertToVoucherConnectionV1(
		vouchers.Rows,
		int(vouchers.Offset),
		int(vouchers.Total),
	)
}

func (a AdapterV1) GetNotice(noticeIndex int, inputIndex int) (*graphql.Notice, error) {
	ctx := context.Background()
	notice, err := a.convenienceService.FindNoticeByInputAndOutputIndex(
		ctx,
		uint64(inputIndex),
		uint64(noticeIndex),
	)
	if err != nil {
		return nil, err
	}
	if notice == nil {
		return nil, fmt.Errorf("notice not found")
	}
	return &graphql.Notice{
		Index:      noticeIndex,
		InputIndex: inputIndex,
		Payload:    notice.Payload,
	}, nil
}

func (a AdapterV1) GetReport(
	reportIndex int, inputIndex int,
) (*graphql.Report, error) {
	report, err := a.reportRepository.FindByInputAndOutputIndex(
		uint64(inputIndex),
		uint64(reportIndex),
	)
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, fmt.Errorf("report not found")
	}
	return a.convertToReport(*report), nil
}

func (a AdapterV1) GetReports(
	first *int, last *int, after *string, before *string, inputIndex *int,
) (*graphql.ReportConnection, error) {
	reports, err := a.reportRepository.FindAllByInputIndex(
		first, last, after, before, inputIndex,
	)
	if err != nil {
		slog.Error("Adapter GetReports", "error", err)
		return nil, err
	}
	return a.convertToReportConnection(
		reports.Rows,
		int(reports.Offset),
		int(reports.Total),
	)
}

func (a AdapterV1) convertToReportConnection(
	reports []repos.Report,
	offset int, total int,
) (*graphql.ReportConnection, error) {
	convNodes := make([]*graphql.Report, len(reports))
	for i := range reports {
		convNodes[i] = a.convertToReport(reports[i])
	}
	return graphql.NewConnection(offset, total, convNodes), nil
}

func (a AdapterV1) convertToReport(
	report repos.Report,
) *graphql.Report {
	return &graphql.Report{
		Index:      report.Index,
		InputIndex: report.InputIndex,
		Payload:    fmt.Sprintf("0x%s", common.Bytes2Hex(report.Payload)),
	}
}

func (a AdapterV1) GetInput(index int) (*graphql.Input, error) {
	input, err := a.inputRepository.FindByIndex(index)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, fmt.Errorf("input not found")
	}
	return graphql.ConvertInput(*input), nil
}

func (a AdapterV1) GetInputs(
	first *int, last *int, after *string, before *string, where *graphql.InputFilter,
) (*graphql.InputConnection, error) {
	filters := []*convenience.ConvenienceFilter{}
	if where != nil {
		field := "Index"
		if where.IndexGreaterThan != nil {
			value := fmt.Sprintf("%d", *where.IndexGreaterThan)
			filters = append(filters, &convenience.ConvenienceFilter{
				Field: &field,
				Gt:    &value,
			})
		}
		if where.IndexLowerThan != nil {
			value := fmt.Sprintf("%d", *where.IndexLowerThan)
			filters = append(filters, &convenience.ConvenienceFilter{
				Field: &field,
				Lt:    &value,
			})
		}
	}
	inputs, err := a.inputRepository.FindAll(
		first, last, after, before, filters,
	)
	if err != nil {
		return nil, err
	}
	return a.convertToInputConnection(
		inputs.Rows,
		int(inputs.Offset),
		int(inputs.Total),
	)
}

func (a AdapterV1) convertToInputConnection(
	inputs []convenience.AdvanceInput,
	offset int, total int,
) (*graphql.InputConnection, error) {
	convNodes := make([]*graphql.Input, len(inputs))
	for i := range inputs {
		convNodes[i] = graphql.ConvertInput(inputs[i])
	}
	return graphql.NewConnection(offset, total, convNodes), nil
}
