package paio

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
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
	ClientSender    Sender
	paioNonceUrl    string
}

// GetNonceDeprecated implements ServerInterface.
func (p *PaioAPI) GetNonceDeprecated(ctx echo.Context) error {
	slog.Warn("Deprecated endpoint, please use /transaction/nonce instead")
	return p.GetNonce(ctx)
}

// SendCartesiTransactionDeprecated implements ServerInterface.
func (p *PaioAPI) SendCartesiTransactionDeprecated(ctx echo.Context) error {
	slog.Warn("Deprecated endpoint, please use /transaction/submit instead")
	return p.SendCartesiTransaction(ctx)
}

func (p *PaioAPI) getBlockNumber(ctx context.Context) (uint64, error) {
	client, err := ethclient.DialContext(ctx, p.EvmRpcUrl)
	if err != nil {
		return 0, fmt.Errorf("ethclient dial error: %w", err)
	}
	defer client.Close()
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("ethclient block_number error: %w", err)
	}
	return blockNumber, nil
}

func (p *PaioAPI) getNonceFromPaio(user common.Address, app common.Address) (*NonceResponse, error) {
	url := p.paioNonceUrl
	payload := map[string]string{
		"application": app.Hex(),
		"user":        user.Hex(),
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		slog.Error("error marshaling json", "error", err)
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("error creating request", "error", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("error sending request", "error", err)
		return nil, err
	}
	defer resp.Body.Close()
	var response NonceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		slog.Error("error decoding paio's response", "error", err)
		return nil, err
	}
	return &response, nil
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
	msgSender := common.HexToAddress(request.MsgSender)
	appContract := common.HexToAddress(request.AppContract)
	if p.paioNonceUrl != "" {
		slog.Debug("Requesting Paio's nonce for",
			"msgSender", msgSender,
			"appContract", appContract,
		)
		response, err := p.getNonceFromPaio(msgSender, appContract)
		if err != nil {
			return err
		}
		return ctx.JSON(http.StatusOK, response)
	}

	slog.Debug("GetNonce", "AppContract", request.AppContract, "MsgSender", request.MsgSender)

	total, err := p.inputRepository.GetNonce(stdCtx, appContract, msgSender)
	if err != nil {
		slog.Error("Error querying for inputs:", "err", err)
		return err
	}
	nonce := int(total)
	response := NonceResponse{
		Nonce: &nonce,
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
	typeJSON, err := json.Marshal(request.TypedData)
	if err != nil {
		return fmt.Errorf("error marshalling typed data: %w", err)
	}
	sigAndData := commons.SigAndData{
		Signature: *request.Signature,
		TypedData: base64.StdEncoding.EncodeToString(typeJSON),
	}
	jsonPayload, err := json.Marshal(sigAndData)
	if err != nil {
		slog.Error("Error json.Marshal message:", "err", err)
		return err
	}
	slog.Debug("/submit", "jsonPayload", string(jsonPayload))
	msgSender, _, signature, err := commons.ExtractSigAndData(string(jsonPayload))
	if err != nil {
		slog.Error("Error ExtractSigAndData message:", "err", err)
		return err
	}
	if request.Address != nil && common.HexToAddress(*request.Address) != msgSender {
		errorMessage := "wrong signature"
		return ctx.JSON(http.StatusBadRequest, TransactionError{Message: &errorMessage})
	}
	appContract := common.HexToAddress(request.TypedData.Message.App[2:])
	slog.Debug("SaveTransaction",
		"msgSender", msgSender,
		"appContract", appContract.Hex(),
		"message", request.TypedData.Message,
	)
	txId := fmt.Sprintf("0x%s", common.Bytes2Hex(crypto.Keccak256(signature)))
	if p.ClientSender != nil {
		seqTxId, err := p.ClientSender.SubmitSigAndData(sigAndData)
		if err != nil {
			return err
		}
		slog.Info("Transaction sent to the sequencer", "txId", seqTxId)
		response := TransactionResponse{
			Id: &txId,
		}
		return ctx.JSON(http.StatusCreated, response)
	}
	blockNumber, err := p.getBlockNumber(stdCtx)
	if err != nil {
		slog.Error("Error reading current block number:", "err", err)
		return err
	}
	inputCount, err := p.inputRepository.Count(stdCtx, nil)
	if err != nil {
		slog.Error("Error counting inputs:", "err", err)
		return err
	}
	payload := request.TypedData.Message.Data[2:]
	_, err = p.inputRepository.Create(stdCtx, model.AdvanceInput{
		ID:             txId,
		Index:          int(inputCount),
		MsgSender:      msgSender,
		Payload:        payload,
		AppContract:    appContract,
		BlockNumber:    blockNumber,
		BlockTimestamp: time.Now(),
		InputBoxIndex:  -2,
		Type:           "L2",
		ChainId:        "31337",
		PrevRandao:     "0xaabb",
	})
	if err != nil {
		slog.Error("Error saving input:", "err", err)
		return err
	}
	msg, _ := json.Marshal(request.TypedData.Message)
	slog.Info("transaction saved",
		"txId", txId,
		"msgSender", msgSender,
		"appContract", appContract.Hex(),
		"data", payload,
		"message", string(msg),
	)
	response := TransactionResponse{
		Id: &txId,
	}
	return ctx.JSON(http.StatusOK, response)
}

// Register the Paio API to echo
func Register(e *echo.Echo, paioServerAPI ServerInterface) {
	RegisterHandlers(e, paioServerAPI)
}
