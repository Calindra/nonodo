package reader

import (
	"fmt"
	"log/slog"

	repos "github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type Adapter interface {
	GetReport(reportIndex int, inputIndex int) (*model.Report, error)

	GetReports(
		first *int, last *int, after *string, before *string, inputIndex *int,
	) (*model.ReportConnection, error)

	GetInputs(
		first *int, last *int, after *string, before *string, where *model.InputFilter,
	) (*model.InputConnection, error)

	GetInput(index int) (*model.Input, error)
}

type AdapterV1 struct {
	reportRepository *repos.ReportRepository
}

func NewAdapterV1(db *sqlx.DB) Adapter {
	slog.Debug("NewAdapterV1")
	repo := &repos.ReportRepository{
		Db: db,
	}
	err := repo.CreateTables()
	if err != nil {
		panic(err)
	}
	return AdapterV1{
		reportRepository: repo,
	}
}

func (a AdapterV1) GetReport(
	reportIndex int, inputIndex int,
) (*model.Report, error) {
	report, err := a.reportRepository.FindByInputAndOutputIndex(
		uint64(inputIndex),
		uint64(reportIndex),
	)
	if err != nil {
		return nil, err
	}
	return a.convertToReport(*report), nil
}

func (a AdapterV1) GetReports(
	first *int, last *int, after *string, before *string, inputIndex *int,
) (*model.ReportConnection, error) {
	reports, err := a.reportRepository.FindAllByInputIndex(
		first, last, after, before, inputIndex,
	)
	if err != nil {
		slog.Error("Adapter GetReports error", err)
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
) (*model.ReportConnection, error) {
	convNodes := make([]*model.Report, len(reports))
	for i := range reports {
		convNodes[i] = a.convertToReport(reports[i])
	}
	return model.NewConnection(offset, total, convNodes), nil
}

func (a AdapterV1) convertToReport(
	report repos.Report,
) *model.Report {
	return &model.Report{
		Index:      report.Index,
		InputIndex: report.InputIndex,
		Payload:    fmt.Sprintf("0x%s", common.Bytes2Hex(report.Payload)),
	}
}

func (a AdapterV1) GetInput(index int) (*model.Input, error) {
	return nil, nil
}

func (a AdapterV1) GetInputs(
	first *int, last *int, after *string, before *string, where *model.InputFilter,
) (*model.InputConnection, error) {
	return nil, nil
}
