package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type SynchronizerCreateWorker struct {
	inputRepository    *repository.InputRepository
	inputRefRepository *repository.RawInputRefRepository
	DbRawUrl           string
	DbRaw              *RawNode
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

func (s SynchronizerCreateWorker) GetDataRawData(abi *abi.ABI, rawData []byte) (map[string]any, error) {
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

	latestRawID, err := s.inputRefRepository.GetLatestRawId(ctx)
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
					inputs, err := s.DbRaw.FindAllInputsByFilter(ctx, FilterInput{IDgt: latestRawID}, page)
					if err != nil {
						errCh <- err
						return
					}

					for _, input := range inputs {
						data, err := s.GetDataRawData(abi, input.RawData)
						if err != nil {
							errCh <- err
							return
						}

						chainID, ok := data["chainID"].(string)

						if !ok {
							errCh <- fmt.Errorf("chainID not found")
							return
						}

						payload, ok := data["payload"].([]byte)
						if !ok {
							errCh <- fmt.Errorf("payload not found")
							return
						}

						rawInputRef := repository.RawInputRef{
							RawID:       uint64(input.ID),
							InputIndex:  input.Index,
							AppContract: common.BytesToAddress(input.ApplicationAddress).Hex(),
							Status:      input.Status,
							ChainID:     chainID,
						}
						advanceInput := model.AdvanceInput{
							Index:   int(input.Index),
							Status:  commons.ConvertStatusStringToCompletionStatus(input.Status),
							Payload: payload,
							ChainId: chainID,
						}

						err = s.inputRefRepository.Create(ctx, rawInputRef)
						if err != nil {
							errCh <- err
							return
						}

						_, err = s.inputRepository.Create(ctx, advanceInput)
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

func NewSynchronizerCreateWorker(inputRepository *repository.InputRepository, inputRefRepository *repository.RawInputRefRepository, dbRawUrl string) supervisor.Worker {
	return SynchronizerCreateWorker{inputRepository: inputRepository, inputRefRepository: inputRefRepository, DbRawUrl: dbRawUrl}
}
