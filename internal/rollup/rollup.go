// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the bindings for the rollup OpenAPI spec.
package rollup

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ../../api/rollup.yaml

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
)

const FinishRetries = 50
const FinishPollInterval = time.Millisecond * 100

// Register the rollup API to echo
func Register(e *echo.Echo, model *model.NonodoModel) {
	sequencer := InputBoxSequencer{model: model}
	var rollupAPI ServerInterface = &rollupAPI{model, &sequencer}
	RegisterHandlers(e, rollupAPI)
}

// Shared struct for request handlers.
type rollupAPI struct {
	model     *model.NonodoModel
	sequencer Sequencer
}

type InputBoxSequencer struct {
	model *model.NonodoModel
}

type EspressoSequencer struct {
	//??
}

func (es *EspressoSequencer) FinishAndGetNext(accept bool) model.Input {
	return nil
}

func (ibs *InputBoxSequencer) FinishAndGetNext(accept bool) model.Input {
	return ibs.model.FinishAndGetNext(accept)
}

type Sequencer interface {
	FinishAndGetNext(accept bool) model.Input
}

type FetchInputBoxContext struct {
	blockNumber             big.Int
	epoch                   big.Int
	currentInput            big.Int
	currentInputBlockNumber big.Int
	currentEpoch            big.Int
}

// type FetchInputBoxContextOrError struct {
// 	context *FetchInputBoxContext
// 	err     error
// }

const (
	INPUT_BOX_SIZE   = 130
	INPUT_FETCH_SIZE = 130
)

var EPOCH_DURATION = getEpochDuration()
var VM_ID = devnet.ApplicationAddress[0:18]

func computeEpoch(blockNumber *big.Int) (*big.Int, error) {
	// try to mimic current Authority epoch computation
	if EPOCH_DURATION == nil {
		return nil, fmt.Errorf("invalid epoch duration")
	} else {
		result := new(big.Int).Div(blockNumber, EPOCH_DURATION)
		return result, nil
	}
}

func (r *rollupAPI) fetchCurrentInput() (*model.AdvanceInput, error) {
	// retrieve total number of inputs
	input := r.model.GetInputRepository()
	currentInput, err := input.FindByStatusNeDesc(model.CompletionStatusUnprocessed)
	if err != nil {
		return nil, err
	}

	return currentInput, nil
}

func getEpochDuration() *big.Int {
	EPOCH_DURATION := os.Getenv("EPOCH_DURATION")
	var epochDuration *big.Int
	if EPOCH_DURATION != "" {
		i, err := strconv.ParseInt(EPOCH_DURATION, 10, 64)
		if err != nil {
			panic(err)
		}
		epochDuration = big.NewInt(i)
	} else {
		oneDay := 86400
		epochDuration = big.NewInt(int64(oneDay))
	}

	return epochDuration
}

func (r *rollupAPI) fetchContext(blockNumber *big.Int) (*FetchInputBoxContext, error) {
	currentInput, err := r.fetchCurrentInput()
	currentInputIndex := big.NewInt(0).SetInt64(int64(currentInput.Index))

	if err != nil {
		return nil, err
	}

	currentInputBlockNumber := big.NewInt(0).SetInt64(int64(currentInput.BlockNumber))

	currentEpoch, err := computeEpoch(currentInputBlockNumber)
	if err != nil {
		return nil, err
	}
	epoch, err := computeEpoch(blockNumber)
	if err != nil {
		return nil, err
	}

	if epoch.Cmp(currentEpoch) != 1 {
		err := fmt.Sprintf(
			"Requested data beyond current epoch '%s'"+
				" (data estimated to belong to epoch '%s')",
			currentEpoch.String(),
			epoch.String(),
		)
		slog.Error(err)
		return nil, fmt.Errorf(err)
	}

	var context FetchInputBoxContext = FetchInputBoxContext{
		blockNumber:             *blockNumber,
		epoch:                   *epoch,
		currentInput:            *currentInputIndex,
		currentInputBlockNumber: *currentInputBlockNumber,
		currentEpoch:            *currentEpoch,
	}

	return &context, nil
}

func (r *rollupAPI) fetchEspresso(ctx echo.Context, id string) (*string, *model.HttpCustomError) {
	// check if id is valid and parse id as maxBlockNumber and espressoBlockHeight
	if len(id) != INPUT_FETCH_SIZE || id[:2] != "0x" {
		err := fmt.Sprintf("Invalid id %s: : must be a hex string with 32 bytes for maxBlockNumber and 32 bytes for espressoBlockHeight", id)
		slog.Error(err)
		return nil, model.NewHttpCustomError(http.StatusBadRequest, nil)
	}
	maxBlockNumber := big.NewInt(0).SetBytes([]byte(id[2:66]))
	espressoBlockHeight := big.NewInt(0).SetBytes([]byte(id[66:130]))

	context, err := r.fetchContext(maxBlockNumber)

	if err != nil {
		return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)
	}

	// check if out of epoch's scope
	if context.epoch.Cmp(&context.currentEpoch) == 1 {
		error := fmt.Sprintf(
			"Requested data beyond current epoch '%s'"+
				" (data estimated to belong to epoch '%s')",
			context.currentEpoch.String(),
			context.epoch.String(),
		)
		slog.Error(error)
		return nil, model.NewHttpCustomError(http.StatusForbidden, nil)
	}

	ctxHttp := ctx.Request().Context()
	urlBase := "https://query.cappuccino.testnet.espresso.network/"
	espressoService := NewExpressoService(ctxHttp, &urlBase)

	for {
		lastEspressoBlockHeight, err := espressoService.GetLatestBlockHeight()
		if err != nil {
			msg := fmt.Sprintf("Failed to get latest block height: %s", err)
			slog.Error(msg)
			return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

		}
		if espressoBlockHeight.Cmp(lastEspressoBlockHeight) == 1 {
			// requested Espresso block not available yet: just check if we are still within L1 blockNumber scope
			header, err := espressoService.GetHeaderByBlockByHeight(lastEspressoBlockHeight)
			if err != nil {
				msg := fmt.Sprintf("Failed to get header by block height: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			l1FinalizedNumber := header.L1Finalized.Number
			l1Finalized := big.NewInt(0).SetUint64(l1FinalizedNumber)
			if l1Finalized.Cmp(maxBlockNumber) == 1 {
				msg := fmt.Sprintf("Espresso block height %s is not finalized", espressoBlockHeight)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			// call again at some later time to see if we reach the block
			var timeInMs time.Duration = 500
			time.Sleep(timeInMs * time.Millisecond)
		} else {
			// requested Espresso block available: fetch it
			filteredBlock, err := espressoService.GetTransactionByHeight(espressoBlockHeight)
			if err != nil {
				msg := fmt.Sprintf("Failed to get block by height: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			header, err := espressoService.GetHeaderByBlockByHeight(espressoBlockHeight)

			if err != nil {
				msg := fmt.Sprintf("Failed to get header by block height: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			// check if within L1 blockNumber scope
			l1FinalizedNumber := header.L1Finalized.Number
			l1Finalized := big.NewInt(0).SetUint64(l1FinalizedNumber)
			if l1Finalized == nil {
				msg := fmt.Sprintf("Espresso block %s with undefined L1 blockNumber", espressoBlockHeight)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusNotFound, nil)
			}

			if l1Finalized.Cmp(maxBlockNumber) == 1 {
				msg := fmt.Sprintf("Espresso block height %s beyond requested L1 blockNumber", espressoBlockHeight)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusNotFound, nil)
			}

			serializedBlock, err := json.Marshal(filteredBlock)
			if err != nil {
				msg := fmt.Sprintf("Failed to marshal block: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}
			encodedBlockHex := hexutil.Encode(serializedBlock)
			// nTransactions := len(blockFiltered.Payload.TransactionNMT)
			nTransactions := len(filteredBlock.Transactions)
			slog.Info(fmt.Sprintf("Fetched Espresso block %s with %d transactions", espressoBlockHeight, nTransactions))
			return &encodedBlockHex, nil
		}
	}
}

func (r *rollupAPI) Fetcher(ctx echo.Context, request GioJSONRequestBody) (*GioResponseRollup, *model.HttpCustomError) {
	var espresso uint16 = 2222
	var syscoin uint16 = 5700
	var its_ok uint16 = 42

	deb, err := json.Marshal(request)

	if err != nil {
		slog.Debug("Failed to marshal request", "error", err)
	} else {
		slog.Debug("Fetcher called", "json", string(deb))
	}

	switch request.Domain {
	case espresso:
		data, err := r.fetchEspresso(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	case syscoin:
		data, err := model.FetchSyscoinPoDa(ctx, request.Id)

		if err != nil {
			return nil, err
		}

		return &GioResponseRollup{Data: *data, Code: its_ok}, nil
	default:
		unsupported := "Unsupported domain"
		return nil, model.NewHttpCustomError(http.StatusBadRequest, &unsupported)
	}
}

// Gio implements ServerInterface.
func (r *rollupAPI) Gio(ctx echo.Context) error {

	if !checkContentType(ctx) {
		return ctx.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request GioJSONRequestBody
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	fetch, err := r.Fetcher(ctx, request)

	if err != nil {
		return ctx.String(int(err.Status()), err.Error())
	}

	if fetch == nil {
		return ctx.String(http.StatusNotFound, "Not found")
	}

	return ctx.JSON(http.StatusOK, fetch)
}

// Handle requests to /finish.
func (r *rollupAPI) Finish(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request FinishJSONRequestBody
	if err := c.Bind(&request); err != nil {
		return err
	}

	// validate fields
	var accepted bool
	switch request.Status {
	case Accept:
		accepted = true
	case Reject:
		accepted = false
	default:
		return c.String(http.StatusBadRequest, "invalid value for status")
	}

	// talk to model
	for i := 0; i < FinishRetries; i++ {
		input := r.sequencer.FinishAndGetNext(accepted)
		if input != nil {
			resp := convertInput(input)
			return c.JSON(http.StatusOK, &resp)
		}
		ctx := c.Request().Context()
		select {
		case <-ctx.Done():
			return c.String(http.StatusInternalServerError, ctx.Err().Error())
		case <-time.After(FinishPollInterval):
		}
	}
	return c.String(http.StatusAccepted, "no rollup request available")
}

// Handle requests to /voucher.
func (r *rollupAPI) AddVoucher(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request AddVoucherJSONRequestBody
	if err := c.Bind(&request); err != nil {
		return err
	}

	// validate fields
	destination, err := hexutil.Decode(request.Destination)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}
	if len(destination) != common.AddressLength {
		return c.String(http.StatusBadRequest, "invalid address length")
	}
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	// talk to model
	index, err := r.model.AddVoucher(common.Address(destination), payload)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	resp := IndexResponse{
		Index: uint64(index),
	}
	return c.JSON(http.StatusOK, &resp)
}

// Handle requests to /notice.
func (r *rollupAPI) AddNotice(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request AddNoticeJSONRequestBody
	if err := c.Bind(&request); err != nil {
		return err
	}

	// validate fields
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	// talk to model
	index, err := r.model.AddNotice(payload)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	resp := IndexResponse{
		Index: uint64(index),
	}
	return c.JSON(http.StatusOK, &resp)
}

// Handle requests to /report.
func (r *rollupAPI) AddReport(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request AddReportJSONRequestBody
	if err := c.Bind(&request); err != nil {
		return err
	}

	// validate fields
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	// talk to model
	err = r.model.AddReport(payload)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// Handle requests to /exception.
func (r *rollupAPI) RegisterException(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request RegisterExceptionJSONRequestBody
	if err := c.Bind(&request); err != nil {
		return err
	}

	// validate fields
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	// talk to model
	err = r.model.RegisterException(payload)
	if err != nil {
		return c.String(http.StatusForbidden, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// Check whether the content type is application/json.
func checkContentType(c echo.Context) bool {
	cType := c.Request().Header.Get(echo.HeaderContentType)
	return strings.HasPrefix(cType, echo.MIMEApplicationJSON)
}

// Convert model input to API type.
func convertInput(input model.Input) RollupRequest {
	var resp RollupRequest
	switch input := input.(type) {
	case model.AdvanceInput:
		advance := Advance{
			Metadata: Metadata{
				BlockNumber:    input.BlockNumber,
				InputIndex:     uint64(input.Index),
				MsgSender:      hexutil.Encode(input.MsgSender[:]),
				BlockTimestamp: uint64(input.BlockTimestamp.Unix()),
			},
			Payload: hexutil.Encode(input.Payload),
		}
		err := resp.Data.FromAdvance(advance)
		if err != nil {
			panic("failed to convert advance")
		}
		resp.RequestType = AdvanceState
	case model.InspectInput:
		inspect := Inspect{
			Payload: hexutil.Encode(input.Payload),
		}
		err := resp.Data.FromInspect(inspect)
		if err != nil {
			panic("failed to convert inspect")
		}
		resp.RequestType = InspectState
	default:
		panic("invalid input from model")
	}
	return resp
}
