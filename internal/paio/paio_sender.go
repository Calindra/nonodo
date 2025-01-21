package paio

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type PaioSender2Server struct {
	PaioServerUrl string
}

func EncodePaioFormat(sigAndData commons.SigAndData) (string, error) {
	// nolint
	typedData := apitypes.TypedData{}
	typedDataBytes, err := base64.StdEncoding.DecodeString(sigAndData.TypedData)
	if err != nil {
		return "", fmt.Errorf("decode typed data: %w", err)
	}
	if err := json.Unmarshal(typedDataBytes, &typedData); err != nil {
		return "", fmt.Errorf("unmarshal typed data: %w", err)
	}
	address := typedData.Message["app"].(string)
	data := typedData.Message["data"].(string)
	nonce, err := ToUint64(typedData.Message["nonce"])
	if err != nil {
		return "", fmt.Errorf("nonce error")
	}
	maxGasPrice, err := ToBig(typedData.Message["max_gas_price"])
	if err != nil {
		return "", fmt.Errorf("max_gas_price error")
	}
	slog.Debug("Decode", "address", address, "data", data,
		"nonce", nonce, "maxGasPrice", maxGasPrice)
	abiEncoder, err := abi.JSON(strings.NewReader(DEFINITION))
	if err != nil {
		return "", nil
	}
	method, ok := abiEncoder.Methods["signingMessage"]
	if !ok {
		slog.Error("error getting method signingMessage", "err", err)
		return "", fmt.Errorf("paio: error getting method signingMessage")
	}
	dappAddress := common.HexToAddress(address)
	encodedBytes, err := method.Inputs.Pack(
		dappAddress,
		nonce,
		maxGasPrice,
		common.Hex2Bytes(data[2:]),
	)
	if err != nil {
		slog.Error("ABI error", "err", err)
		return "", err
	}
	encoded := common.Bytes2Hex(encodedBytes)
	msg := PaioReqMessage{
		Signature: sigAndData.Signature,
		Message:   fmt.Sprintf("0x%s", encoded),
	}
	json, err := json.Marshal(msg)
	if err != nil {
		slog.Error("json.Marshal error", "err", err)
		return "", err
	}
	return string(json), nil
}

type PaioReqMessage struct {
	Signature string `json:"signature"`
	Message   string `json:"message"`
}

func ToUint64(value interface{}) (uint64, error) {
	b, err := ToBig(value)
	if err != nil {
		return 0, err
	}
	return b.Uint64(), nil
}

func ToBig(value interface{}) (*big.Int, error) {
	nonce := big.NewInt(0)
	nonceStr, ok := value.(string)
	if !ok {
		nonceFloat, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("converting to big error")
		}
		nonce = nonce.SetUint64(uint64(nonceFloat))
	} else {
		nonce, ok = nonce.SetString(nonceStr, 10) // nolint
		if !ok {
			return nil, fmt.Errorf("converting to big error 2")
		}
	}
	return nonce, nil
}

// SubmitSigAndData implements Sender.
func (p PaioSender2Server) SubmitSigAndData(sigAndData commons.SigAndData) (string, error) {
	jsonData, err := EncodePaioFormat(sigAndData)
	if err != nil {
		return "", err
	}
	transactionUrl := fmt.Sprintf("%s/transaction", p.PaioServerUrl)
	req, err := http.NewRequest("POST", transactionUrl, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		slog.Error("Unexpected paio response",
			"paioTransactionUrl", transactionUrl,
			"statusCode", resp.StatusCode,
			"json", jsonData,
		)
		return "", fmt.Errorf("unexpected paio server status code %d", resp.StatusCode)
	}
	slog.Debug("POST to Paio", "body", jsonData)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return "", err
	}

	slog.Debug("Response", "status", resp.StatusCode, "body", string(body))
	return "", nil
}

func NewPaioSender2Server(url string) Sender {
	return PaioSender2Server{
		PaioServerUrl: url,
	}
}
