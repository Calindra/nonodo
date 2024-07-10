package synchronizer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/convenience/adapter"
	"github.com/calindra/nonodo/internal/convenience/decoder"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type GraphileSynchronizer struct {
	Decoder                *decoder.OutputDecoder
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
			x.handleGraphileResponse(*voucherResp, ctx)
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

func (x GraphileSynchronizer) handleGraphileResponse(outputResp OutputResponse, ctx context.Context) {
	// Handle response data
	voucherIds := []string{}
	var initCursorAfter string
	var initInputCursorAfter string

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
		adapted := adapter.ConvertVoucherPayloadToV2(
			output.Node.Blob[2:],
		)
		destination, _ := adapter.GetDestination(output.Node.Blob)

		if len(destination) == 0 {
			panic(fmt.Errorf("graphile sync error: len(destination) is 0"))
		}

		err := x.Decoder.HandleOutput(ctx,
			destination,
			adapted,
			uint64(inputIndex),
			uint64(outputIndex),
		)
		if err != nil {
			panic(err)
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

		adapted, _ := adapter.GetConvertedInput(input.Node.Blob)

		inputIndex := adapted[1].(int)
		msgSender := common.HexToAddress(adapted[3].(string))
		payload := adapted[3].(string)
		blockNumber := adapted[4].(uint64)
		blockTimestamp := adapted[5].(time.Time)
		prevRandao := adapted[6].(string)

		err := x.Decoder.HandleInput(ctx,
			inputIndex,
			model.CompletionStatusUnprocessed,
			msgSender,
			payload,
			blockNumber,
			blockTimestamp,
			prevRandao)

		if err != nil {
			panic(err)
		}
	}

	hasMoreInputs := len(outputResp.Data.Inputs.PageInfo.EndCursor) > 0

	if hasMoreInputs {
		initInputCursorAfter = x.GraphileFetcher.CursorInputAfter
		x.GraphileFetcher.CursorInputAfter = outputResp.Data.Outputs.PageInfo.EndCursor
	}

	if hasMoreInputs || hasMoreOutputs {
		_, err := x.SynchronizerRepository.Create(
			ctx, &model.SynchronizerFetch{
				TimestampAfter:      uint64(time.Now().UnixMilli()),
				IniCursorAfter:      initCursorAfter,
				EndCursorAfter:      x.GraphileFetcher.CursorAfter,
				LogVouchersIds:      strings.Join(voucherIds, ";"),
				IniInputCursorAfter: initInputCursorAfter,
				EndInputCursorAfter: x.GraphileFetcher.CursorInputAfter,
			})
		if err != nil {
			slog.Error("Deu erro", "erro", err)
			panic(err)
		}
	}

}

func NewGraphileSynchronizer(
	decoder *decoder.OutputDecoder,
	synchronizerRepository *repository.SynchronizerRepository,
	graphileFetcher *GraphileFetcher,
) *GraphileSynchronizer {
	return &GraphileSynchronizer{
		Decoder:                decoder,
		SynchronizerRepository: synchronizerRepository,
		GraphileFetcher:        graphileFetcher}
}
