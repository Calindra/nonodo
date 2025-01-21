package synchronizernode

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerInputCreator struct {
	InputRepository       *repository.InputRepository
	RawInputRefRepository *repository.RawInputRefRepository
	RawNodeV2Repository   *RawRepository
	AbiDecoder            *AbiDecoder
}

func NewSynchronizerInputCreator(
	inputRepository *repository.InputRepository,
	rawInputRefRepository *repository.RawInputRefRepository,
	rawRepository *RawRepository,
	abiDecoder *AbiDecoder,
) *SynchronizerInputCreator {
	return &SynchronizerInputCreator{
		InputRepository:       inputRepository,
		RawInputRefRepository: rawInputRefRepository,
		RawNodeV2Repository:   rawRepository,
		AbiDecoder:            abiDecoder,
	}
}

func (s SynchronizerInputCreator) SyncInputs(ctx context.Context) error {
	txCtx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	err = s.syncInputs(txCtx)
	if err != nil {
		s.rollbackTransaction(txCtx)
		return err
	}
	err = s.commitTransaction(txCtx)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerInputCreator) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.InputRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, &db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerInputCreator) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}

func (s *SynchronizerInputCreator) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerInputCreator) syncInputs(ctx context.Context) error {
	latestRawID, err := s.RawInputRefRepository.GetLatestRawId(ctx)
	if err != nil {
		return err
	}

	page := &Pagination{Limit: LIMIT}

	inputs, err := s.RawNodeV2Repository.FindAllInputsByFilter(ctx, FilterInput{IDgt: latestRawID}, page)
	if err != nil {
		return err
	}

	for _, input := range inputs {

		err = s.CreateInput(ctx, input)
		if err != nil {
			return err
		}

	}
	return nil
}

func (s *SynchronizerInputCreator) CreateInput(ctx context.Context, rawInput RawInput) error {
	advanceInput, err := s.GetAdvanceInputFromMap(rawInput)
	if err != nil {
		return err
	}

	inputBox, err := s.InputRepository.Create(ctx, *advanceInput)
	if err != nil {
		return err
	}

	rawInputRef := repository.RawInputRef{
		ID:          inputBox.ID,
		RawID:       uint64(rawInput.ID),
		InputIndex:  rawInput.Index,
		AppContract: common.BytesToAddress(rawInput.ApplicationAddress).Hex(),
		Status:      rawInput.Status,
		ChainID:     advanceInput.ChainId,
	}

	err = s.RawInputRefRepository.Create(ctx, rawInputRef)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerInputCreator) GetAdvanceInputFromMap(rawInput RawInput) (*model.AdvanceInput, error) {
	decodedData, err := s.AbiDecoder.GetMapRaw(rawInput.RawData)
	if err != nil {
		return nil, err
	}

	chainId, ok := decodedData["chainId"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("chainId not found")
	}

	payload, ok := decodedData["payload"].([]byte)
	if !ok {
		return nil, fmt.Errorf("payload not found")
	}

	msgSender, ok := decodedData["msgSender"].(common.Address)
	if !ok {
		return nil, fmt.Errorf("msgSender not found")
	}

	blockNumber, ok := decodedData["blockNumber"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("blockNumber not found")
	}

	blockTimestamp, ok := decodedData["blockTimestamp"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("blockTimestamp not found")
	}

	appContract, ok := decodedData["appContract"].(common.Address)
	if !ok {
		return nil, fmt.Errorf("appContract not found")
	}

	prevRandao, ok := decodedData["prevRandao"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("prevRandao not found")
	}

	inputBoxIndex, ok := decodedData["index"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("inputBoxIndex not found")
	}

	slog.Debug("GetAdvanceInputFromMap", "chainId", chainId)
	advanceInput := model.AdvanceInput{
		ID:                     FormatTransactionId(rawInput.TransactionId),
		AppContract:            appContract,
		Index:                  int(rawInput.Index),
		InputBoxIndex:          int(inputBoxIndex.Int64()),
		MsgSender:              msgSender,
		BlockNumber:            blockNumber.Uint64(),
		BlockTimestamp:         time.Unix(blockTimestamp.Int64(), 0),
		Payload:                common.Bytes2Hex(payload),
		ChainId:                chainId.String(),
		Status:                 commons.ConvertStatusStringToCompletionStatus(rawInput.Status),
		PrevRandao:             "0x" + prevRandao.Text(16), // nolint
		EspressoBlockTimestamp: time.Unix(-1, 0),
		AvailBlockTimestamp:    time.Unix(-1, 0),
	}
	// advanceInput.Status = model.CompletionStatusUnprocessed
	return &advanceInput, nil
}
