package synchronizer

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/convenience/adapter"
	"github.com/calindra/nonodo/internal/convenience/decoder"
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

type Service struct {
	adapter        adapter.Adapter
	decoderService *decoder.OutputDecoder
}
type AdapterConnector interface {
	ConvertVoucher(output Edge) string
	RetrieveDestination(output Edge) (common.Address, error)
	GetConvertedInput(output InputEdge) ([]interface{}, error)
}

type DecoderConnector interface {
	GetHandleOutput(
		ctx context.Context,
		destination common.Address,
		payload string,
		inputIndex uint64,
		outputIndex uint64,
	) error

	GetHandleInput(
		ctx context.Context,
		index int,
		status model.CompletionStatus,
		msgSender common.Address,
		payload string,
		blockNumber uint64,
		blockTimestamp time.Time,
		prevRandao string,
	) error

	GetHandleReport(
		ctx context.Context,
		index int,
		outputIndex int,
		payload string,
	) error
}

type Edge struct {
	Cursor string `json:"cursor"`
	Node   struct {
		Index      int    `json:"index"`
		Blob       string `json:"blob"`
		InputIndex int    `json:"inputIndex"`
	} `json:"node"`
}

type InputEdge struct {
	Cursor string `json:"cursor"`
	Node   struct {
		Index int    `json:"index"`
		Blob  string `json:"blob"`
	} `json:"node"`
}

func (m *Service) ConvertVoucher(output Edge) string {
	adapted := m.adapter.ConvertVoucherPayloadToV3(output.Node.Blob[2:])
	return adapted
}

func (m *Service) RetrieveDestination(output Edge) (common.Address, error) {
	return m.adapter.GetDestinationV2(output.Node.Blob)
}

func (m *Service) RetrieveConvertedInput(input InputEdge) ([]interface{}, error) {
	return m.adapter.GetConvertedInputV2(input.Node.Blob)
}

func (m *Service) GetHandleOutput(ctx context.Context, destination common.Address, payload string, inputIndex uint64, outputIndex uint64) error {
	return m.decoderService.HandleOutput(ctx, destination, payload, inputIndex, outputIndex)
}

func (m *Service) GetHandleInput(ctx context.Context, index int, status model.CompletionStatus, msgSender common.Address, payload string, blockNumber uint64, blockTimestamp time.Time, prevRandao string) error {
	return m.decoderService.HandleInput(ctx, index, status, msgSender, payload, blockNumber, blockTimestamp, prevRandao)
}

func (m *Service) GetHandleReport(ctx context.Context, index int, outputIndex int, payload string) error {
	return m.decoderService.HandleReport(ctx, index, outputIndex, payload)
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
			err := x.handleGraphileResponse(*voucherResp, ctx)
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

func (x GraphileSynchronizer) handleGraphileResponse(outputResp OutputResponse, ctx context.Context) error {
	// Handle response data
	voucherIds := []string{}
	var initCursorAfter string
	var initInputCursorAfter string
	var initReportCursorAfter string

	for _, output := range outputResp.Data.Outputs.Edges {
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
		adapted := x.Adapter.ConvertVoucher(output)
		destination, err := x.Adapter.RetrieveDestination(output)

		if err != nil {
			slog.Error("Failed to retrieve destination for node blob '%s': %v", output.Node.Blob, err)
			return fmt.Errorf("error retrieving destination for node blob '%s': %w", output.Node.Blob, err)
		}

		err = x.Decoder.GetHandleOutput(ctx,
			destination,
			adapted,
			uint64(inputIndex),
			uint64(outputIndex),
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

		adapted, err := x.Adapter.GetConvertedInput(input)

		if err != nil {
			slog.Error("Failed to get converted:", err)
			return fmt.Errorf("error getting converted input: %w", err)
		}

		inputIndex := input.Node.Index
		msgSender := adapted[2].(common.Address)
		payload := string(adapted[7].([]uint8))
		blockNumber := adapted[3].(*big.Int)
		blockTimestamp := adapted[4].(*big.Int).Int64()
		prevRandao := adapted[5].(*big.Int).String()

		err = x.Decoder.GetHandleInput(ctx,
			inputIndex,
			model.CompletionStatusUnprocessed,
			msgSender,
			payload,
			blockNumber.Uint64(),
			time.Unix(blockTimestamp, 0),
			prevRandao)

		if err != nil {
			panic(err)
		}
	}

	for _, report := range outputResp.Data.Reports.Edges {
		slog.Debug("Add Report",
			"Index", report.Node.Index,
			"InputIndex", report.Node.InputIndex,
		)
		err := x.Decoder.GetHandleReport(
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

	if hasMoreInputs || hasMoreOutputs {
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
			slog.Error("Deu erro", "erro", err)
			panic(err)
		}
	}
	return nil
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
		Adapter:                adapter}
}
