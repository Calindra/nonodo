package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience/decoder"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type SynchronizerCreateWorker struct {
	inputRepository    *repository.InputRepository
	inputRefRepository *repository.RawInputRefRepository
	DbRawUrl           string
	RawRepository      *RawRepository
	SynchronizerUpdate *SynchronizerUpdate
	Decoder            *decoder.OutputDecoder
}

const DEFAULT_DELAY = 3 * time.Second

// Start implements supervisor.Worker.
func (s SynchronizerCreateWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	return s.WatchNewInputs(ctx)
}

func (s SynchronizerCreateWorker) GetMapRaw(abi *abi.ABI, rawData []byte) (map[string]any, error) {
	data := make(map[string]any)

	methodId := rawData[:4]
	method, err := abi.MethodById(methodId)
	if err != nil {
		return nil, err
	}

	err = method.Inputs.UnpackIntoMap(data, rawData[4:])

	slog.Debug("DecodedData", "map", data)

	return data, err
}

func (s SynchronizerCreateWorker) GetAdvanceInputFromMap(data map[string]any, input RawInput) (*model.AdvanceInput, error) {
	chainId, ok := data["chainId"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("chainId not found")
	}

	payload, ok := data["payload"].([]byte)
	if !ok {
		return nil, fmt.Errorf("payload not found")
	}

	msgSender, ok := data["msgSender"].(common.Address)
	if !ok {
		return nil, fmt.Errorf("msgSender not found")
	}

	blockNumber, ok := data["blockNumber"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("blockNumber not found")
	}

	blockTimestamp, ok := data["blockTimestamp"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("blockTimestamp not found")
	}

	appContract, ok := data["appContract"].(common.Address)
	if !ok {
		return nil, fmt.Errorf("appContract not found")
	}

	slog.Debug("GetAdvanceInputFromMap", "chainId", chainId)
	advanceInput := model.AdvanceInput{
		// nolint
		// TODO: check if the ID is correct
		ID:             strconv.FormatUint(input.ID, 10),
		AppContract:    appContract,
		Index:          int(input.Index),
		MsgSender:      msgSender,
		BlockNumber:    blockNumber.Uint64(),
		BlockTimestamp: time.Unix(0, blockTimestamp.Int64()),
		Payload:        payload,
		ChainId:        chainId.String(),
		Status:         commons.ConvertStatusStringToCompletionStatus(input.Status),
	}
	// advanceInput.Status = model.CompletionStatusUnprocessed
	return &advanceInput, nil

}

func (s SynchronizerCreateWorker) HandleInput(ctx context.Context, abi *abi.ABI, input RawInput) (id uint64, err error) {
	data, err := s.GetMapRaw(abi, input.RawData)
	if err != nil {
		return 0, err
	}

	advanceInput, err := s.GetAdvanceInputFromMap(data, input)
	if err != nil {
		return 0, err
	}

	inputBox, err := s.inputRepository.Create(ctx, *advanceInput)
	if err != nil {
		return 0, err
	}

	rawInputRef := repository.RawInputRef{
		ID:          inputBox.ID,
		RawID:       uint64(input.ID),
		InputIndex:  input.Index,
		AppContract: common.BytesToAddress(input.ApplicationAddress).Hex(),
		Status:      input.Status,
		ChainID:     advanceInput.ChainId,
	}
	// rawInputRef.Status = "NONE"
	err = s.inputRefRepository.Create(ctx, rawInputRef)
	if err != nil {
		return 0, err
	}

	return rawInputRef.RawID, nil
}

func (s SynchronizerCreateWorker) SyncInputCreation(ctx context.Context, latestRawID uint64, page *Pagination, abi *abi.ABI) (uint64, error) {
	inputs, err := s.RawRepository.FindAllInputsByFilter(ctx, FilterInput{IDgt: latestRawID}, page)
	if err != nil {
		return latestRawID, err
	}

	for _, input := range inputs {
		rawInputRefID, err := s.HandleInput(ctx, abi, input)
		if err != nil {
			return latestRawID, err
		}
		latestRawID = rawInputRefID + 1
	}
	return latestRawID, nil
}

func (s SynchronizerCreateWorker) WatchNewInputs(stdCtx context.Context) error {
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
				default:
					latestRawID, err = s.SyncInputCreation(ctx, latestRawID, page, abi)
					if err != nil {
						errCh <- err
						return
					}
					err = s.SynchronizerUpdate.SyncInputStatus(ctx)
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
) supervisor.Worker {
	return SynchronizerCreateWorker{
		inputRepository:    inputRepository,
		inputRefRepository: inputRefRepository,
		DbRawUrl:           dbRawUrl,
		RawRepository:      rawRepository,
		SynchronizerUpdate: synchronizerUpdate,
		Decoder:            decoder,
	}
}
