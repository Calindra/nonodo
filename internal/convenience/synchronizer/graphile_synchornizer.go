package synchronizer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
)

type GraphileSynchronizer struct {
	Decoder                model.DecoderInterface
	SynchronizerRepository *repository.SynchronizerRepository
	GraphileFetcher        *GraphileFetcher
}

func (x GraphileSynchronizer) String() string {
	return "GraphileSynchronizer"
}

func (x GraphileSynchronizer) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}

	sleepInSeconds := 3

	lastFetch, err := x.SynchronizerRepository.GetLastFetched(ctx)

	if err != nil {
		return err
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
				slog.Error("Failed to handle graphile response.", "err", err)
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
	var initCursorAfter string
	var initInputCursorAfter string
	var initReportCursorAfter string

	for _, output := range outputResp.Data.Outputs.Edges {

		processOutputData := model.ProcessOutputData{
			OutputIndex: uint64(output.Node.Index),
			InputIndex:  uint64(output.Node.InputIndex),
			Payload:     output.Node.Blob[2:],
			Destination: output.Node.Blob,
		}

		err := x.Decoder.HandleOutputV2(ctx, processOutputData)
		if err != nil {
			slog.Error("Failed to handle output: ", "err", err)
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
			slog.Error("Failed to handle input:", "err", err)
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
			slog.Error("Failed to handle report:", "err", err)
			return fmt.Errorf("error handling report: %w", err)
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
				LogVouchersIds:       "",
				IniInputCursorAfter:  initInputCursorAfter,
				EndInputCursorAfter:  x.GraphileFetcher.CursorInputAfter,
				IniReportCursorAfter: initReportCursorAfter,
				EndReportCursorAfter: x.GraphileFetcher.CursorReportAfter,
			})
		if err != nil {
			slog.Error("Failed to create synchronize repository:", "err", err)
			return fmt.Errorf("error creating synchronize repository: %w", err)
		}
	}
	return nil
}

func NewGraphileSynchronizer(
	decoder model.DecoderInterface,
	synchronizerRepository *repository.SynchronizerRepository,
	graphileFetcher *GraphileFetcher,
) *GraphileSynchronizer {
	return &GraphileSynchronizer{
		Decoder:                decoder,
		SynchronizerRepository: synchronizerRepository,
		GraphileFetcher:        graphileFetcher,
	}
}
