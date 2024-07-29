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

type Synchronizer struct {
	decoder                *decoder.OutputDecoder
	VoucherFetcher         *VoucherFetcher
	SynchronizerRepository *repository.SynchronizerRepository
}

func NewSynchronizer(
	decoder *decoder.OutputDecoder,
	voucherFetcher *VoucherFetcher,
	SynchronizerRepository *repository.SynchronizerRepository,
) *Synchronizer {
	return &Synchronizer{
		decoder:                decoder,
		VoucherFetcher:         voucherFetcher,
		SynchronizerRepository: SynchronizerRepository,
	}
}

// String implements supervisor.Worker.
func (x Synchronizer) String() string {
	return "Synchronizer"
}

func (x Synchronizer) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	return x.VoucherPolling(ctx)
}

func (x *Synchronizer) VoucherPolling(ctx context.Context) error {
	sleepInSeconds := 3
	lastFetch, err := x.SynchronizerRepository.GetLastFetched(ctx)
	if err != nil {
		return err
	}
	if lastFetch != nil {
		x.VoucherFetcher.CursorAfter = lastFetch.EndCursorAfter
	}
	for {
		voucherResp, err := x.VoucherFetcher.Fetch()
		if err != nil {
			slog.Warn(
				"Voucher fetcher error, we will try again",
				"error", err.Error(),
			)
		} else {
			// Handle response data
			voucherIds := []string{}
			for _, edge := range voucherResp.Data.Vouchers.Edges {
				outputIndex := edge.Node.Index
				inputIndex := edge.Node.Input.Index
				slog.Debug("Add Voucher",
					"inputIndex", inputIndex,
					"outputIndex", outputIndex,
				)
				voucherIds = append(
					voucherIds,
					fmt.Sprintf("%d:%d", inputIndex, outputIndex),
				)
				adapted := adapter.ConvertVoucherPayloadToV2(
					edge.Node.Payload[2:],
				)
				err := x.decoder.HandleOutput(ctx,
					common.HexToAddress(edge.Node.Destination),
					adapted,
					uint64(inputIndex),
					uint64(outputIndex),
				)
				if err != nil {
					return err
				}
			}
			if len(voucherResp.Data.Vouchers.PageInfo.EndCursor) > 0 {
				initCursorAfter := x.VoucherFetcher.CursorAfter
				x.VoucherFetcher.CursorAfter = voucherResp.Data.
					Vouchers.PageInfo.EndCursor
				_, err := x.SynchronizerRepository.Create(
					ctx, &model.SynchronizerFetch{
						TimestampAfter: uint64(time.Now().UnixMilli()),
						IniCursorAfter: initCursorAfter,
						EndCursorAfter: x.VoucherFetcher.CursorAfter,
						LogVouchersIds: strings.Join(voucherIds, ";"),
					})
				if err != nil {
					return err
				}
			}
		}
		select {
		// Wait a little before doing another request
		case <-time.After(time.Duration(sleepInSeconds) * time.Second):
		case <-ctx.Done():
			slog.Debug("Synchronizer canceled:", "Error", ctx.Err().Error())
			return nil
		}
	}
}
