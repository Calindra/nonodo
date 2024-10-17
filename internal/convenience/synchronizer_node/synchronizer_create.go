package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
)

type SynchronizerCreateWorker struct {
	Container *convenience.Container
	DbRawUrl  string
	DbRaw     *RawNode
}

const DEFAULT_TIMEOUT = 1 * time.Second

// Start implements supervisor.Worker.
func (s SynchronizerCreateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	s.DbRaw = NewRawNode(s.DbRawUrl)
	db, err := s.DbRaw.Connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	return s.WatchNewInputs(ctx, db)
}

func (s SynchronizerCreateWorker) GetChainRawData(abi *abi.ABI, rawData []byte) (map[string]any, error) {
	data := make(map[string]any)

	methodId := rawData[:4]
	method, err := abi.MethodById(methodId)

	if err != nil {
		return nil, err
	}

	err = method.Inputs.UnpackIntoMap(data, rawData[4:])

	return data, err
}

func (s SynchronizerCreateWorker) WatchNewInputs(stdCtx context.Context, db *sqlx.DB) error {
	ctx, cancel := context.WithCancel(stdCtx)
	defer cancel()

	rawInputRep := s.Container.GetRawInputRepository()
	latestRawID, err := rawInputRep.GetLatestRawId(ctx)

	if err != nil {
		return err
	}

	abi, err := contracts.InputsMetaData.GetAbi()
	if err != nil {
		return err
	}

	page := &Pagination{Limit: LIMIT}

	for {
		errCh := make(chan error)

		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case <-time.After(DEFAULT_TIMEOUT):
				default:
					inputs, err := s.DbRaw.FindAllInputsByFilter(ctx, FilterInput{IDgt: latestRawID, IsStatusNone: true}, page)
					if err != nil {
						errCh <- err
						return
					}

					for _, input := range inputs {
						data, err := s.GetChainRawData(abi, input.RawData)

						if err != nil {
							errCh <- err
							return
						}

						chainID, ok := data["chainID"].(string)

						if !ok {
							errCh <- fmt.Errorf("chainID not found")
							return
						}

						rawInputRef := repository.RawInputRef{
							RawID:       uint64(input.ID),
							InputIndex:  input.Index,
							AppContract: common.BytesToAddress(input.ApplicationAddress).Hex(),
							Status:      input.Status,
							ChainID:     chainID,
						}

						err = rawInputRep.Create(ctx, rawInputRef)
						if err != nil {
							errCh <- err
							return
						}

						rawInputRefID, err := strconv.ParseUint(rawInputRef.ID, 10, 64)
						if err != nil {
							errCh <- err
							return
						}
						latestRawID = rawInputRefID
					}
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

func NewSynchronizerCreateWorker(container *convenience.Container, dbRawUrl string) supervisor.Worker {
	return SynchronizerCreateWorker{Container: container, DbRawUrl: dbRawUrl}
}
