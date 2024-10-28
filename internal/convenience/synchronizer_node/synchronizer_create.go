package synchronizernode

import (
	"context"
	"encoding/binary"
	"log/slog"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/convenience/decoder"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type SynchronizerCreateWorker struct {
	inputRepository          *repository.InputRepository
	inputRefRepository       *repository.RawInputRefRepository
	outputRefRepository      *repository.RawOutputRefRepository
	SynchronizerReport       *SynchronizerReport
	DbRawUrl                 string
	RawRepository            *RawRepository
	SynchronizerUpdate       *SynchronizerUpdate
	Decoder                  *decoder.OutputDecoder
	SynchronizerOutputUpdate *SynchronizerOutputUpdate
	SynchronizerOutputCreate *SynchronizerOutputCreate
	SynchronizerCreateInput  *SynchronizerCreateInput
}

const DEFAULT_DELAY = 3 * time.Second

// Start implements supervisor.Worker.
func (s SynchronizerCreateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	return s.WatchNewInputs(ctx)
}

// nolint
func FormatTransactionId(txId []byte) string {
	if len(txId) <= 8 {
		padded := make([]byte, 8)
		copy(padded[8-len(txId):], txId)
		n := binary.BigEndian.Uint64(padded)
		return strconv.FormatUint(n, 10)
	} else {
		return "0x" + common.Bytes2Hex(txId)
	}
}

func (s SynchronizerCreateWorker) WatchNewInputs(stdCtx context.Context) error {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	for {
		errCh := make(chan error)

		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				default:
					err := s.SynchronizerCreateInput.SyncInputs(ctx)
					if err != nil {
						errCh <- err
						return
					}
					err = s.SynchronizerUpdate.SyncInputStatus(ctx)
					if err != nil {
						errCh <- err
						return
					}
					err = s.SynchronizerReport.SyncReports(ctx)
					if err != nil {
						errCh <- err
						return
					}

					err = s.SynchronizerOutputCreate.SyncOutputs(ctx)
					if err != nil {
						errCh <- err
						return
					}

					err = s.SynchronizerOutputUpdate.SyncOutputs(ctx)
					if err != nil {
						errCh <- err
						return
					}

					<-time.After(DEFAULT_DELAY)
				}
			}
		}()

		wrong := <-errCh

		if wrong != nil {
			return wrong
		}

		slog.Debug("Retrying to fetch new inputs")
	}
}

// String implements supervisor.Worker.
func (s SynchronizerCreateWorker) String() string {
	return "SynchronizerCreateWorker"
}

func NewSynchronizerCreateWorker(
	inputRepository *repository.InputRepository,
	inputRefRepository *repository.RawInputRefRepository,
	dbRawUrl string,
	rawRepository *RawRepository,
	synchronizerUpdate *SynchronizerUpdate,
	decoder *decoder.OutputDecoder,
	synchronizerReport *SynchronizerReport,
	synchronizerOutputUpdate *SynchronizerOutputUpdate,
	outputRefRepository *repository.RawOutputRefRepository,
	synchronizerOutputCreate *SynchronizerOutputCreate,
	synchronizerCreateInput *SynchronizerCreateInput,
) supervisor.Worker {
	return SynchronizerCreateWorker{
		inputRepository:          inputRepository,
		inputRefRepository:       inputRefRepository,
		DbRawUrl:                 dbRawUrl,
		RawRepository:            rawRepository,
		SynchronizerUpdate:       synchronizerUpdate,
		Decoder:                  decoder,
		SynchronizerReport:       synchronizerReport,
		SynchronizerOutputUpdate: synchronizerOutputUpdate,
		outputRefRepository:      outputRefRepository,
		SynchronizerOutputCreate: synchronizerOutputCreate,
		SynchronizerCreateInput:  synchronizerCreateInput,
	}
}
