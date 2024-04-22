package reader

import (
	"fmt"
	"log/slog"

	convenience "github.com/calindra/nonodo/internal/convenience/model"
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
	inputRepository  *repos.InputRepository
}

func NewAdapterV1(db *sqlx.DB) Adapter {
	slog.Debug("NewAdapterV1")
	reportRepository := &repos.ReportRepository{
		Db: db,
	}
	err := reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	inputRepository := &repos.InputRepository{
		Db: db,
	}
	err = inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return AdapterV1{
		reportRepository: reportRepository,
		inputRepository:  inputRepository,
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
	input, err := a.inputRepository.FindByIndex(index)
	if err != nil {
		return nil, err
	}
	return model.ConvertInput(*input), nil
}

func (a AdapterV1) GetInputs(
	first *int, last *int, after *string, before *string, where *model.InputFilter,
) (*model.InputConnection, error) {
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
	inputs []repos.AdvanceInput,
	offset int, total int,
) (*model.InputConnection, error) {
	convNodes := make([]*model.Input, len(inputs))
	for i := range inputs {
		convNodes[i] = model.ConvertInput(inputs[i])
	}
	return model.NewConnection(offset, total, convNodes), nil
}
