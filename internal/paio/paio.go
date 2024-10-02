package paio

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/labstack/echo/v4"
)

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ./oapi-paio.yaml

//go:embed paio.json
var DEFINITION string

type PaioTypedData struct {
	apitypes.TypedData
	Account common.Address `json:"account"`
}

type PaioAPI struct {
	availClient     *avail.AvailClient
	inputRepository *repository.InputRepository
	EvmRpcUrl       string
	chainID         *big.Int
}

func (p *PaioAPI) getChainID() (*big.Int, error) {
	if p.chainID != nil {
		return p.chainID, nil
	}
	stdCtx := context.Background()
	client, err := ethclient.DialContext(stdCtx, p.EvmRpcUrl)
	if err != nil {
		return nil, fmt.Errorf("ethclient dial error: %w", err)
	}
	chainId, err := client.ChainID(stdCtx)
	if err != nil {
		return nil, fmt.Errorf("ethclient chainId error: %w", err)
	}
	slog.Info("Using", "chainId", chainId.Uint64())
	return chainId, nil
}

// SendTransaction implements ServerInterface.
func (p *PaioAPI) SendTransaction(ctx echo.Context) error {
	var request SendTransactionJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return err
	}
	slog.Debug("Sending Avail transaction", "request", request)
	sigAndData := commons.SigAndData{
		Signature: request.Signature,
		TypedData: request.TypedData,
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		slog.Error("Error json.Marshal message:", "err", err)
		return err
	}
	hash, err := p.availClient.DefaultSubmit(stdCtx, string(jsonPayload))
	if err != nil {
		slog.Error("Error DefaultSubmit message:", "err", err)
		return err
	}
	_ = ctx.String(http.StatusOK, hash.Hex())
	return nil
}

func (p *PaioAPI) GetNonce(ctx echo.Context) error {
	var request GetNonceJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	if request.MsgSender == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "msg_sender is required"})
	}

	filters := []*model.ConvenienceFilter{}
	msgSenderField := "MsgSender"
	msgSender := common.HexToAddress(request.MsgSender).Hex()
	filters = append(filters, &model.ConvenienceFilter{
		Field: &msgSenderField,
		Eq:    &msgSender,
	})

	typeField := "Type"
	inputBoxType := "inputbox"
	filters = append(filters, &model.ConvenienceFilter{
		Field: &typeField,
		Ne:    &inputBoxType,
	})

	appContractField := "AppContract"
	appContract := common.HexToAddress(request.AppContract).Hex()
	filters = append(filters, &model.ConvenienceFilter{
		Field: &appContractField,
		Eq:    &appContract,
	})

	slog.Debug("GetNonce", "AppContract", request.AppContract, "MsgSender", request.MsgSender)

	inputs, err := p.inputRepository.FindAll(stdCtx, nil, nil, nil, nil, filters)

	if err != nil {
		slog.Error("Error querying for inputs:", "err", err)
		return err
	}

	nonce := int(inputs.Total + 1)
	response := NonceResponse{
		Nonce: &nonce,
	}

	return ctx.JSON(http.StatusOK, response)
}

func (p *PaioAPI) SaveTransaction(ctx echo.Context) error {
	var request SaveTransactionJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	if request.Signature == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "signature is required"})
	}

	if request.Message == "" {
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "message is required"})
	}

	// decode the ABI from message
	// https://github.com/fabiooshiro/frontend-web-cartesi/blob/16913e945ef687bd07b6c3900d63cb23d69390b1/src/Input.tsx#L195C13-L212C15
	decoder, err := abi.JSON(strings.NewReader(DEFINITION))
	if err != nil {
		slog.Error("error decoding ABI:", "err", err)
		return ctx.JSON(http.StatusInternalServerError, echo.Map{"error": "avail: error decoding ABI"})
	}
	method, ok := decoder.Methods["signingMessage"]
	if !ok {
		slog.Error("error getting method signingMessage", "err", err)
		return ctx.JSON(http.StatusInternalServerError, echo.Map{"error": "avail: error getting method signingMessage"})
	}

	// decode the message, message don't have 4 bytes of method id
	message := common.Hex2Bytes(strings.TrimPrefix(request.Message, "0x"))
	data := make(map[string]any)
	err = method.Inputs.UnpackIntoMap(data, message)
	if err != nil {
		slog.Error("error unpacking message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error unpacking message"})
	}

	// Validate the data from the message
	app, ok := data["app"].(common.Address)
	if !ok {
		slog.Error("error extracting app from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting app from message"})
	}
	nonce, ok := data["nonce"].(uint64)
	if !ok {
		slog.Error("error extracting nonce from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting nonce from message"})
	}
	maxGasPrice, ok := data["max_gas_price"].(*big.Int)
	if !ok {
		slog.Error("error extracting max_gas_price from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting max_gas_price from message"})
	}
	dataBytes, ok := data["data"].([]byte)
	if !ok {
		slog.Error("error extracting data from message", "err", err)
		return ctx.JSON(http.StatusBadRequest, echo.Map{"error": "avail: error extracting data from message"})
	}

	chainId, err := p.getChainID()
	if err != nil {
		return fmt.Errorf("ethclient dial error: %w", err)
	}

	// fill the typedData
	// https://github.com/fabiooshiro/frontend-web-cartesi/blob/16913e945ef687bd07b6c3900d63cb23d69390b1/src/Input.tsx#L65
	typedData := paiodecoder.CreateTypedData(
		app, nonce, maxGasPrice, dataBytes, chainId,
	)

	typeJSON, err := json.Marshal(typedData)
	if err != nil {
		return fmt.Errorf("error marshalling typed data: %w", err)
	}

	// set the typedData as string json below
	sigAndData := commons.SigAndData{
		Signature: request.Signature,
		TypedData: base64.StdEncoding.EncodeToString(typeJSON),
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		slog.Error("Error json.Marshal message:", "err", err)
		return err
	}
	slog.Debug("SaveTransaction", "jsonPayload", string(jsonPayload))
	msgSender, _, signature, err := commons.ExtractSigAndData(string(jsonPayload))

	if err != nil {
		slog.Error("Error:", "err", err)
		return err
	}

	if request.MsgSender != nil && common.HexToAddress(*request.MsgSender) != msgSender {
		msg := "wrong signature"
		return ctx.JSON(http.StatusBadRequest, TransactionError{Message: &msg})
	}

	dappAddress := app.String()
	payload := string(dataBytes)

	slog.Info("Input saved",
		"dappAddress", dappAddress,
		"msgSender", msgSender,
		"nonce", nonce,
		"maxGasPrice", maxGasPrice,
		"payload", payload,
	)

	payloadBytes := []byte(payload)
	if strings.HasPrefix(payload, "0x") {
		payload = payload[2:] // remove 0x
		payloadBytes, err = hex.DecodeString(payload)
		if err != nil {
			return err
		}
	}

	inputCount, err := p.inputRepository.Count(stdCtx, nil)

	if err != nil {
		slog.Error("Error counting inputs:", "err", err)
		return err
	}

	txId := fmt.Sprintf("0x%s", common.Bytes2Hex(crypto.Keccak256(signature)))
	createdInput, err := p.inputRepository.Create(stdCtx, model.AdvanceInput{
		ID:            txId,
		Index:         int(inputCount + 1),
		MsgSender:     msgSender,
		Payload:       payloadBytes,
		AppContract:   common.HexToAddress(dappAddress),
		InputBoxIndex: -2,
		Type:          "Avail",
	})

	if err != nil {
		slog.Error("Error creating inputs:", "err", err)
		return err
	}

	slog.Info("Input created", "id", createdInput.ID)

	response := TransactionResponse{
		Id: &txId,
	}

	return ctx.JSON(http.StatusOK, response)
}

// SendCartesiTransaction implements ServerInterface.
func (p *PaioAPI) SendCartesiTransaction(ctx echo.Context) error {
	var request SendCartesiTransactionJSONRequestBody
	stdCtx, cancel := context.WithCancel(ctx.Request().Context())
	defer cancel()
	if err := ctx.Bind(&request); err != nil {
		return err
	}
	slog.Debug("SendCartesiTransaction", "x", stdCtx)

	typeJSON, err := json.Marshal(request.TypedData)
	if err != nil {
		return fmt.Errorf("error marshalling typed data: %w", err)
	}

	// set the typedData as string json below
	sigAndData := commons.SigAndData{
		Signature: *request.Signature,
		TypedData: base64.StdEncoding.EncodeToString(typeJSON),
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		slog.Error("Error json.Marshal message:", "err", err)
		return err
	}
	slog.Debug("SaveTransaction", "jsonPayload", string(jsonPayload))
	msgSender, _, signature, err := commons.ExtractSigAndData(string(jsonPayload))
	if err != nil {
		slog.Error("Error ExtractSigAndData message:", "err", err)
		return err
	}
	if common.HexToAddress(request.TypedData.Account) != msgSender {
		errorMessage := "wrong signature"
		return ctx.JSON(http.StatusBadRequest, TransactionError{Message: &errorMessage})
	}
	appContract := common.HexToAddress(request.TypedData.Message.App[2:])
	slog.Debug("SaveTransaction",
		"msgSender", msgSender,
		"appContract", appContract.Hex(),
		"message", request.TypedData.Message,
	)
	inputCount, err := p.inputRepository.Count(stdCtx, nil)
	if err != nil {
		slog.Error("Error counting inputs:", "err", err)
		return err
	}
	txId := fmt.Sprintf("0x%s", common.Bytes2Hex(crypto.Keccak256(signature)))
	_, err = p.inputRepository.Create(stdCtx, model.AdvanceInput{
		ID:            txId,
		Index:         int(inputCount + 1),
		MsgSender:     msgSender,
		Payload:       common.Hex2Bytes(request.TypedData.Message.Data[2:]),
		AppContract:   appContract,
		InputBoxIndex: -2,
		Type:          "L2",
	})
	if err != nil {
		slog.Error("Error saving input:", "err", err)
		return err
	}
	response := TransactionResponse{
		Id: &txId,
	}
	return ctx.JSON(http.StatusOK, response)
}

// Register the Paio API to echo
func Register(e *echo.Echo, availClient *avail.AvailClient, inputRepository *repository.InputRepository, rpcUrl string) {
	var paioAPI ServerInterface = &PaioAPI{availClient, inputRepository, rpcUrl, nil}
	RegisterHandlers(e, paioAPI)
}
