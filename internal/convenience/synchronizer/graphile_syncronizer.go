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
	}

	for {
		voucherResp, err := x.GraphileFetcher.Fetch()

		if err != nil {
			slog.Warn(
				"Voucher fetcher error, we will try again",
				"error", err.Error(),
			)
		} else {
			// Handle response data
			voucherIds := []string{}
			for _, edge := range voucherResp.Data.Outputs.Edges {
				outputIndex := edge.Node.Index
				inputIndex := edge.Node.InputIndex
				slog.Debug("Add Voucher",
					"inputIndex", inputIndex,
					"outputIndex", outputIndex,
				)
				voucherIds = append(
					voucherIds,
					fmt.Sprintf("%d:%d", inputIndex, outputIndex),
				)
				adapted := adapter.ConvertVoucherPayloadToV2(
					edge.Node.Blob[2:],
				)
				destination, _ := adapter.GetDestination(adapted)

				if len(destination) == 0 {
					panic(err)
				}

				err := x.Decoder.HandleOutput(ctx,
					common.HexToAddress(destination),
					adapted,
					uint64(inputIndex),
					uint64(outputIndex),
				)
				if err != nil {
					panic(err)
				}
			}
			if len(voucherResp.Data.Outputs.PageInfo.EndCursor) > 0 {
				initCursorAfter := x.GraphileFetcher.CursorAfter
				x.GraphileFetcher.CursorAfter = voucherResp.Data.Outputs.PageInfo.EndCursor
				_, err := x.SynchronizerRepository.Create(
					ctx, &model.SynchronizerFetch{
						TimestampAfter: uint64(time.Now().UnixMilli()),
						IniCursorAfter: initCursorAfter,
						EndCursorAfter: x.GraphileFetcher.CursorAfter,
						LogVouchersIds: strings.Join(voucherIds, ";"),
					})
				if err != nil {
					panic(err)
				}
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
