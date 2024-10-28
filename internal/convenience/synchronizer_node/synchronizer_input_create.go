package synchronizernode

import (
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerCreateInput struct {
	InputRepository       *repository.InputRepository
	RawInputRefRepository *repository.RawInputRefRepository
	RawNodeV2Repository   *RawRepository
	AbiDecoder            *AbiDecoder
}

func NewSynchronizerCreateInput(
	inputRepository *repository.InputRepository,
	rawInputRefRepository *repository.RawInputRefRepository,
	rawRepository *RawRepository,
	abiDecoder *AbiDecoder,
) *SynchronizerCreateInput {
	return &SynchronizerCreateInput{
		InputRepository:       inputRepository,
		RawInputRefRepository: rawInputRefRepository,
		RawNodeV2Repository:   rawRepository,
		AbiDecoder:            abiDecoder,
	}
}

func (s *SynchronizerCreateInput) GetAdvanceInputFromMap(rawInput RawInput) (*model.AdvanceInput, error) {
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
		// nolint
		// TODO: check if the ID is correct
		ID:                     FormatTransactionId(rawInput.TransactionId),
		AppContract:            appContract,
		Index:                  int(rawInput.Index),
		InputBoxIndex:          int(inputBoxIndex.Int64()),
		MsgSender:              msgSender,
		BlockNumber:            blockNumber.Uint64(),
		BlockTimestamp:         time.Unix(0, blockTimestamp.Int64()),
		Payload:                payload,
		ChainId:                chainId.String(),
		Status:                 commons.ConvertStatusStringToCompletionStatus(rawInput.Status),
		PrevRandao:             "0x" + prevRandao.Text(16), // nolint
		EspressoBlockTimestamp: time.Unix(-1, 0),
		AvailBlockTimestamp:    time.Unix(-1, 0),
	}
	// advanceInput.Status = model.CompletionStatusUnprocessed
	return &advanceInput, nil
}
