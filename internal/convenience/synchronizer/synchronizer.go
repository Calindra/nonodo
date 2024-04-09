package synchronizer

import (
	"context"
	"log/slog"
	"time"

	"github.com/calindra/nonodo/internal/convenience"
	"github.com/ethereum/go-ethereum/common"
)

type Synchronizer struct {
	decoder *convenience.OutputDecoder
}

func NewSynchronizer(decoder *convenience.OutputDecoder) *Synchronizer {
	return &Synchronizer{
		decoder: decoder,
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
	voucherFetcher := NewVoucherFetcher()
	sleepInSeconds := 3
	for {
		voucherResp, err := voucherFetcher.Fetch()
		if err != nil {
			slog.Warn(
				"Voucher fetcher error, we will try again",
				"error", err.Error(),
			)
		} else {
			// Handle response data
			for _, edge := range voucherResp.Data.Vouchers.Edges {
				outputIndex := edge.Node.Index
				inputIndex := edge.Node.Input.Index
				slog.Debug("Add Voucher",
					"inputIndex", inputIndex,
					"outputIndex", outputIndex,
				)
				err := x.decoder.HandleOutput(ctx,
					common.HexToAddress(edge.Node.Destination),
					edge.Node.Payload,
					uint64(inputIndex),
					uint64(outputIndex),
				)
				if err != nil {
					panic(err)
				}
			}
			if len(voucherResp.Data.Vouchers.PageInfo.EndCursor) > 0 {
				voucherFetcher.CursorAfter = voucherResp.Data.
					Vouchers.PageInfo.EndCursor
			}
		}
		select {
		// Wait a little before doing another request
		case <-time.After(time.Duration(sleepInSeconds) * time.Second):
		case <-ctx.Done():
			slog.Debug("Synchronizer canceled:", ctx.Err())
			return nil
		}
	}
}
