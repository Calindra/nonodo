package synchronizer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type GraphileSynchronizer struct {
	Decoder                DecoderConnector
	SynchronizerRepository *repository.SynchronizerRepository
	GraphileFetcher        *GraphileFetcher
	Adapter                AdapterConnector
}

type ProcessOutputData struct {
	OutputIndex int
	InputIndex  int
	Blob        string
	Destination common.Address
}

type AdapterConnector interface {
	RetrieveDestination(output model.OutputEdge) (common.Address, error)
}

type DecoderConnector interface {
	HandleOutput(
		ctx context.Context,
		destination common.Address,
		payload string,
		inputIndex uint64,
		outputIndex uint64,
	) error

	HandleInput(
		ctx context.Context,
		input model.InputEdge,
		status model.CompletionStatus,
	) error

	HandleReport(
		ctx context.Context,
		index int,
		outputIndex int,
		payload string,
	) error

	GetConvertedInput(output model.InputEdge) (model.ConvertedInput, error)
}

func (x GraphileSynchronizer) String() string {
	return "GraphileSynchronizer"
}

func (x GraphileSynchronizer) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}

	sleepInSeconds := 3

	lastFetch, err := x.SynchronizerRepository.GetLastFetched(ctx)

	if err != nil {
		//Com panic
		panic(err)
	}

	if lastFetch != nil {
		x.GraphileFetcher.CursorAfter = lastFetch.EndCursorAfter
		x.GraphileFetcher.CursorInputAfter = lastFetch.EndInputCursorAfter
	}

	for {
		voucherResp, err := x.GraphileFetcher.Fetch()

		if err != nil {
			slog.Warn(
				"Voucher fetcher error, we will try again",
				"error", err.Error(),
			)
		} else {
			err := x.handleGraphileResponse(ctx, *voucherResp)
			if err != nil {
				//Sem panic
				panic(err)
			}
		}
		select {
		// Wait a little before doing another request
		case <-time.After(time.Duration(sleepInSeconds) * time.Second):
		case <-ctx.Done():
			slog.Debug("GraphileSynchronizer canceled:", "Error", ctx.Err().Error())
			return nil
		}

	}

}

func (x GraphileSynchronizer) handleGraphileResponse(ctx context.Context, outputResp OutputResponse) error {
	// Handle response data
	voucherIds := []string{}
	var initCursorAfter string
	var initInputCursorAfter string
	var initReportCursorAfter string

	for _, output := range outputResp.Data.Outputs.Edges {
		processOutPutData, err := x.processOutput(output, voucherIds)

		if err != nil {
			slog.Error("Failed to process the output data.': %v", err)
			return fmt.Errorf("error processing output: %w", err)
		}

		err = x.Decoder.HandleOutput(ctx,
			processOutPutData.Destination,
			processOutPutData.Blob,
			uint64(processOutPutData.InputIndex),
			uint64(processOutPutData.OutputIndex),
		)
		if err != nil {
			slog.Error("Failed to handle output: %v", err)
			return fmt.Errorf("error handling output: %w", err)
		}
	}

	hasMoreOutputs := len(outputResp.Data.Outputs.PageInfo.EndCursor) > 0

	if hasMoreOutputs {
		initCursorAfter = x.GraphileFetcher.CursorAfter
		x.GraphileFetcher.CursorAfter = outputResp.Data.Outputs.PageInfo.EndCursor
	}

	for _, input := range outputResp.Data.Inputs.Edges {

		slog.Debug("Add Input",
			"Index", input.Node.Index,
		)

		err := x.Decoder.HandleInput(ctx,
			input,
			model.CompletionStatusUnprocessed,
		)

		if err != nil {
			slog.Error("Failed to handle input:", err)
			return fmt.Errorf("error handling input: %w", err)
		}
	}

	for _, report := range outputResp.Data.Reports.Edges {
		slog.Debug("Call HandleReport",
			"Index", report.Node.Index,
			"InputIndex", report.Node.InputIndex,
		)
		err := x.Decoder.HandleReport(
			ctx,
			report.Node.InputIndex,
			report.Node.Index,
			report.Node.Blob,
		)
		if err != nil {
			panic(err)
		}
	}

	hasMoreReports := len(outputResp.Data.Reports.PageInfo.EndCursor) > 0
	if hasMoreReports {
		initReportCursorAfter = x.GraphileFetcher.CursorReportAfter
		x.GraphileFetcher.CursorReportAfter = outputResp.Data.Reports.PageInfo.EndCursor
	}

	hasMoreInputs := len(outputResp.Data.Inputs.PageInfo.EndCursor) > 0

	if hasMoreInputs {
		initInputCursorAfter = x.GraphileFetcher.CursorInputAfter
		x.GraphileFetcher.CursorInputAfter = outputResp.Data.Inputs.PageInfo.EndCursor
	}

	if hasMoreInputs || hasMoreOutputs || hasMoreReports {
		_, err := x.SynchronizerRepository.Create(
			ctx, &model.SynchronizerFetch{
				TimestampAfter:       uint64(time.Now().UnixMilli()),
				IniCursorAfter:       initCursorAfter,
				EndCursorAfter:       x.GraphileFetcher.CursorAfter,
				LogVouchersIds:       strings.Join(voucherIds, ";"),
				IniInputCursorAfter:  initInputCursorAfter,
				EndInputCursorAfter:  x.GraphileFetcher.CursorInputAfter,
				IniReportCursorAfter: initReportCursorAfter,
				EndReportCursorAfter: x.GraphileFetcher.CursorReportAfter,
			})
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func (x GraphileSynchronizer) processOutput(output model.OutputEdge, voucherIds []string) (ProcessOutputData, error) {
	outputIndex := output.Node.Index
	inputIndex := output.Node.InputIndex
	slog.Debug("Add Voucher/Notices",
		"inputIndex", inputIndex,
		"outputIndex", outputIndex,
	)
	voucherIds = append(
		voucherIds,
		fmt.Sprintf("%d:%d", inputIndex, outputIndex),
	)

	blob := output.Node.Blob[2:] //O voucher já vem do PostGraphile no modo que o v2 precisa.
	destination, err := x.Adapter.RetrieveDestination(output)
	var emptyprocessOutputData ProcessOutputData

	if err != nil {
		slog.Error("Failed to retrieve destination for node blob '%s': %v", output.Node.Blob, err)
		return emptyprocessOutputData, fmt.Errorf("error retrieving destination for node blob '%s': %w", output.Node.Blob, err)
	}

	processOutputData := ProcessOutputData{
		OutputIndex: outputIndex,
		InputIndex:  inputIndex,
		Blob:        blob,
		Destination: destination,
	}

	return processOutputData, nil
}

func NewGraphileSynchronizer(
	decoder DecoderConnector,
	synchronizerRepository *repository.SynchronizerRepository,
	graphileFetcher *GraphileFetcher,
	adapter AdapterConnector,
) *GraphileSynchronizer {
	return &GraphileSynchronizer{
		Decoder:                decoder,
		SynchronizerRepository: synchronizerRepository,
		GraphileFetcher:        graphileFetcher,
		Adapter:                adapter,
	}
}
