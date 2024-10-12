package paiodecoder

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"os/exec"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const TimeoutExecutionPaioDecoder = 1 * time.Minute

type DecoderPaio interface {
	DecodePaioBatch(ctx context.Context, bytes []byte) (string, error)
}

type PaioDecoder struct {
	location string
}

func NewPaioDecoder(location string) *PaioDecoder {
	return &PaioDecoder{location}
}

// call the paio decoder binary
func (pd *PaioDecoder) DecodePaioBatch(stdCtx context.Context, rawBytes []byte) (string, error) {
	first, err := pd.DecodePaioBatchSkip(stdCtx, 0, rawBytes) // nolint
	if err == nil {
		return first, nil
	}
	slog.Warn("failed to decode, we will try again removing 2 bytes")
	second, err := pd.DecodePaioBatchSkip(stdCtx, 2, rawBytes) // nolint
	if err != nil {
		return "", err
	}
	return second, nil
}

func (pd *PaioDecoder) DecodePaioBatchSkip(stdCtx context.Context, skip int, rawBytes []byte) (string, error) {
	ctx, cancel := context.WithTimeout(stdCtx, TimeoutExecutionPaioDecoder)
	defer cancel()
	cmd := exec.CommandContext(ctx, pd.location)
	var stdinData bytes.Buffer
	bytesStr := common.Bytes2Hex(rawBytes[skip:])
	stdinData.WriteString(bytesStr)
	cmd.Stdin = &stdinData
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to decode", "bytes", bytesStr)
		return "", fmt.Errorf("failed to run command: %w", err)
	}
	slog.Debug("Output decoded", "output", string(output))
	return string(output), nil
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
		},
	}
	typedData.PrimaryType = "CartesiMessage"
	typedData.Message = apitypes.TypedDataMessage{
		"app":           app.String(),
		"nonce":         nonce,
		"max_gas_price": maxGasPrice.String(),
		"data":          fmt.Sprintf("0x%s", common.Bytes2Hex(dataBytes)),
	}
	return typedData
}
