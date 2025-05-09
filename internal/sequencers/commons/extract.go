package espresso

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type SigAndData struct {
	Signature string `json:"signature"`
	TypedData string `json:"typedData"`
}

func ExtractSigAndData(raw string) (common.Address, apitypes.TypedData, string, error) {
	var sigAndData SigAndData
	if err := json.Unmarshal([]byte(raw), &sigAndData); err != nil {
		slog.Error("unmarshal error", "error", err, "raw", raw)
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("unmarshal sigAndData: %w", err)
	}

	signature, err := hexutil.Decode(sigAndData.Signature)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("decode signature: %w", err)
	}
	hash := crypto.Keccak256Hash(signature)
	hashString := hash.Hex()

	typedDataBytes, err := base64.StdEncoding.DecodeString(sigAndData.TypedData)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("decode typed data: %w", err)
	}

	typedData := apitypes.TypedData{}
	if err := json.Unmarshal(typedDataBytes, &typedData); err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("unmarshal typed data: %w", err)
	}

	dataHash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("typed data hash: %w", err)
	}

	// update the recovery id
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	signature[64] -= 27

	// get the pubkey used to sign this signature
	sigPubkey, err := crypto.Ecrecover(dataHash, signature)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("ecrecover: %w", err)
	}
	pubkey, err := crypto.UnmarshalPubkey(sigPubkey)
	if err != nil {
		return common.HexToAddress("0x"), apitypes.TypedData{}, "", fmt.Errorf("unmarshal: %w", err)
	}
	address := crypto.PubkeyToAddress(*pubkey)

	return address, typedData, hashString, nil
}
