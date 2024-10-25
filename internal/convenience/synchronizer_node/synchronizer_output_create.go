package synchronizernode

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerOutputCreate struct {
	VoucherRepository      *repository.VoucherRepository
	NoticeRepository       *repository.NoticeRepository
	RawNodeV2Repository    *RawRepository
	RawOutputRefRepository *repository.RawOutputRefRepository
	AbiDecoder             *AbiDecoder
}

func NewSynchronizerOutputCreate(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	rawRepository *RawRepository,
	rawOutputRefRepository *repository.RawOutputRefRepository,
	abiDecoder *AbiDecoder,
) *SynchronizerOutputCreate {
	return &SynchronizerOutputCreate{
		VoucherRepository:      voucherRepository,
		NoticeRepository:       noticeRepository,
		RawNodeV2Repository:    rawRepository,
		RawOutputRefRepository: rawOutputRefRepository,
		AbiDecoder:             abiDecoder,
	}
}

func (s *SynchronizerOutputCreate) SyncOutputs(ctx context.Context) error {
	latestOutputRawID, err := s.RawOutputRefRepository.GetLatestOutputRawId(ctx)
	if err != nil {
		return err
	}
	outputs, err := s.RawNodeV2Repository.FindAllOutputsByFilter(ctx, FilterID{IDgt: latestOutputRawID})
	if err != nil {
		return err
	}
	for _, rawOutput := range outputs {
		rawOutputRef, err := s.GetRawOutputRef(rawOutput)
		if err != nil {
			return err
		}
		err = s.RawOutputRefRepository.Create(ctx, *rawOutputRef)
		if err != nil {
			return err
		}
		if rawOutputRef.Type == repository.RAW_VOUCHER_TYPE {
			cVoucher, err := s.GetConvenienceVoucher(rawOutput)
			if err != nil {
				return err
			}
			_, err = s.VoucherRepository.CreateVoucher(ctx, cVoucher)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SynchronizerOutputCreate) GetConvenienceVoucher(rawOutput Output) (*model.ConvenienceVoucher, error) {
	data, err := s.AbiDecoder.GetMapRaw(rawOutput.RawData)
	if err != nil {
		return nil, err
	}
	destination, ok := data["destination"].(common.Address)
	if !ok {
		return nil, fmt.Errorf("destination not found %v", data)
	}

	voucherValue, ok := data["value"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("value not found %v", data)
	}
	outputIndex, err := strconv.ParseUint(rawOutput.Index, 10, 64)
	if err != nil {
		return nil, err
	}
	inputIndex, err := strconv.ParseUint(rawOutput.InputIndex, 10, 64)
	if err != nil {
		return nil, err
	}
	strPayload := "0x" + common.Bytes2Hex(rawOutput.RawData)
	cVoucher := model.ConvenienceVoucher{
		Destination: destination,
		Payload:     strPayload,
		Executed:    false,
		InputIndex:  inputIndex,
		OutputIndex: outputIndex,
		AppContract: common.BytesToAddress(rawOutput.AppContract),
		Value:       voucherValue.String(),
	}
	return &cVoucher, nil
}

func (s *SynchronizerOutputCreate) GetRawOutputRef(rawOutput Output) (*repository.RawOutputRef, error) {
	outputIndex, err := strconv.ParseUint(rawOutput.Index, 10, 64)
	if err != nil {
		return nil, err
	}
	inputIndex, err := strconv.ParseUint(rawOutput.InputIndex, 10, 64)
	if err != nil {
		return nil, err
	}
	outputType, err := getOutputType(rawOutput.RawData)
	if err != nil {
		return nil, err
	}
	return &repository.RawOutputRef{
		RawID:       rawOutput.ID,
		InputIndex:  inputIndex,
		OutputIndex: outputIndex,
		AppContract: common.BytesToAddress(rawOutput.AppContract).Hex(),
		Type:        outputType,
	}, nil
}

func getOutputType(rawData []byte) (string, error) {
	var strPayload = "0x" + common.Bytes2Hex(rawData)
	if strPayload[2:10] == model.VOUCHER_SELECTOR {
		return repository.RAW_VOUCHER_TYPE, nil
	} else if strPayload[2:10] == model.NOTICE_SELECTOR {
		return repository.RAW_NOTICE_TYPE, nil
	} else {
		return "", fmt.Errorf("unsupported output selector type: %s", strPayload[2:10])
	}
}
