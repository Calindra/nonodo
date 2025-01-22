package paiodecoder

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/common"
)

type ZzzzHuiDecoder struct {
}

func (z ZzzzHuiDecoder) DecodePaioBatch(ctx context.Context, bytes []byte) (string, error) {
	_, typedData, signature, err := commons.ExtractSigAndData(string(bytes))
	if err != nil {
		return "", err
	}
	signature[64] += 27
	slog.Debug("DecodePaioBatch", "signature", common.Bytes2Hex(signature))
	txs := []PaioTransaction{}
	txs = append(txs, PaioTransaction{
		Signature: PaioSignature{
			R: fmt.Sprintf("0x%s", common.Bytes2Hex(signature[0:32])),
			S: fmt.Sprintf("0x%s", common.Bytes2Hex(signature[32:64])),
			V: fmt.Sprintf("0x%s", common.Bytes2Hex(signature[64:])),
		},
		App:         typedData.Message["app"].(string),
		Nonce:       uint64(typedData.Message["nonce"].(float64)),
		Data:        common.Hex2Bytes(typedData.Message["data"].(string)[2:]),
		MaxGasPrice: uint64(typedData.Message["max_gas_price"].(float64)),
	})
	paioBatch := PaioBatch{
		Txs: txs,
	}
	paioJson, err := json.Marshal(paioBatch)
	if err != nil {
		return "", err
	}
	return string(paioJson), nil
}
