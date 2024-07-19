package adapter

import (
	"fmt"
	"log/slog"
	"math/big"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
)

type Adapter struct{}

func ConvertVoucherPayloadToV2(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.VOUCHER_SELECTOR, payloadV1)
}

func (a *Adapter) ConvertVoucherPayloadToV3(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.VOUCHER_SELECTOR, payloadV1)
}

func ConvertNoticePayloadToV2(payloadV1 string) string {
	return fmt.Sprintf("0x%s%s", model.NOTICE_SELECTOR, payloadV1)
}

// for a while we will remove the prefix
// until the v2 does not arrives
func RemoveSelector(payload string) string {
	return fmt.Sprintf("0x%s", payload[10:])
}

func GetDestination(payload string) (common.Address, error) {
	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return common.Address{}, err
	}

	slog.Info("payload", "payload", payload)

	values, err := abiParsed.Methods["Voucher"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return common.Address{}, err
	}

	return values[0].(common.Address), nil
}

func (a *Adapter) GetDestinationV2(payload string) (common.Address, error) {
	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return common.Address{}, err
	}

	slog.Info("payload", "payload", payload)

	values, err := abiParsed.Methods["Voucher"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return common.Address{}, err
	}

	return values[0].(common.Address), nil
}

func GetConvertedInput(payload string) ([]interface{}, error) {
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return make([]interface{}, 0), err
	}

	values, err := abiParsed.Methods["EvmAdvance"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return make([]interface{}, 0), err
	}

	return values, nil

}

func (a *Adapter) GetConvertedInputV2(payload string) (model.ConvertedInput, error) {
	var emptyConvertedInput model.ConvertedInput
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return emptyConvertedInput, err
	}

	values, err := abiParsed.Methods["EvmAdvance"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return emptyConvertedInput, err
	}
	convertedInput := model.ConvertedInput{
		MsgSender:      values[2].(common.Address),
		Payload:        string(values[7].([]uint8)),
		BlockNumber:    values[3].(*big.Int),
		BlockTimestamp: values[4].(*big.Int).Int64(),
		PrevRandao:     values[5].(*big.Int).String(),
	}

	return convertedInput, nil

}
