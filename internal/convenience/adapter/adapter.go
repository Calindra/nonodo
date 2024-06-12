package adapter

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const abiJSON = `[
	{
	  "inputs": [
		{
		  "internalType": "bytes",
		  "name": "notice",
		  "type": "bytes"
		}
	  ],
	  "name": "Notice",
	  "outputs": [],
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "inputs": [
		{
		  "internalType": "address",
		  "name": "destination",
		  "type": "address"
		},
		{
		  "internalType": "bytes",
		  "name": "payload",
		  "type": "bytes"
		}
	  ],
	  "name": "Voucher",
	  "outputs": [],
	  "stateMutability": "nonpayable",
	  "type": "function"
	}
  ]`

func ConvertVoucherPayloadToV2(payloadV1 string) string {
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

func GetDestination(payload string) (string, error) {
	abiParsed, err := abi.JSON(strings.NewReader(abiJSON))

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return "", err
	}

	values, err := abiParsed.Methods["Voucher"].Inputs.UnpackValues(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return "", err
	}

	slog.Info("values", "batata", values)

	return values[0].(string), nil
}
