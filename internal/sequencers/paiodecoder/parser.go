package paiodecoder

import (
	"fmt"
	"math/big"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type PaioDecoder struct {
}

func (t *PaioDecoder) DecodePaioBatch(bytes string) (string, error) {
	// call the paio decoder binary
	return "", nil
}

func CreateTypedData(
	app common.Address,
	nonce uint64,
	maxGasPrice *big.Int,
	dataBytes []byte,
	chainId *big.Int,
) apitypes.TypedData {
	var typedData apitypes.TypedData
	cid := math.NewHexOrDecimal256(chainId.Int64())
	typedData.Domain = commons.NewCartesiDomain(cid)
	typedData.Types = apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"CartesiMessage": {
			{Name: "app", Type: "address"},
			{Name: "nonce", Type: "uint64"},
			{Name: "max_gas_price", Type: "uint128"},
			{Name: "data", Type: "bytes"},
		}}
	typedData.PrimaryType = "CartesiMessage"
	typedData.Message = apitypes.TypedDataMessage{
		"app":           app.String(),
		"nonce":         nonce,
		"max_gas_price": maxGasPrice.String(),
		"data":          fmt.Sprintf("0x%s", common.Bytes2Hex(dataBytes)),
	}
	return typedData
}
