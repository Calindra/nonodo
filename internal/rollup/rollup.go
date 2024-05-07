// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the bindings for the rollup OpenAPI spec.
package rollup

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ../../api/rollup.yaml

import (
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
)

const FinishRetries = 50
const FinishPollInterval = time.Millisecond * 100

// Register the rollup API to echo
func Register(e *echo.Echo, model *model.NonodoModel) {
	var rollupAPI ServerInterface = &rollupAPI{model}
	RegisterHandlers(e, rollupAPI)
}

// Shared struct for request handlers.
type rollupAPI struct {
	model *model.NonodoModel
}

type FetchResponse struct {
	status uint
	data   *string
}

type FetchInputBoxContext struct {
	blockNumber             big.Int
	epoch                   big.Int
	currentInput            big.Int
	currentInputBlockNumber big.Int
	currentEpoch            big.Int
}

type FetchInputBoxContextOrError struct {
	context *FetchInputBoxContext
	err     error
}

const (
	INPUT_BOX_SIZE = 130
	INPUT_FETCH    = 130
)

var EPOCH_DURATION = getEpochDuration()

func computeEpoch(blockNumber *big.Int) (*big.Int, error) {
	// TODO: try to mimic current Authority epoch computation
	if EPOCH_DURATION == nil {
		return nil, fmt.Errorf("Invalid epochDuration")
	} else {
		result := new(big.Int).Div(blockNumber, EPOCH_DURATION)
		return result, nil
	}
}

func (r *rollupAPI) fetchCurrentInput() (*model.AdvanceInput, error) {
	// retrieve total number of inputs
	if r.model == nil {
		return nil, fmt.Errorf("Model is nil")
	}
	input := r.model.GetInputRepository()
	currInput, err := input.FindByStatusNeDesc(model.CompletionStatusUnprocessed)
	if err != nil {
		return nil, err
	}

	return currInput, nil
}

func waitForBlock(blockNumber *big.Int) error {
	slog.Info("Waiting for block", blockNumber)

	// poll until block is reached

	return nil
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
		epochDuration = big.NewInt(86400)
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

// func FetchInputBox(id string) (*FetchResponse, error) {
// 	if len(id) != INPUT_BOX_SIZE || id[:2] != "0x" {
// 		error := fmt.Sprintf("Invalid id %s box id", id)
// 		slog.Error(error)
// 		return &FetchResponse{status: http.StatusNotFound, data: nil}, nil
// 	}
// 	maxBlockNumber := big.NewInt(0).SetBytes([]byte(id[2:66]))
// 	// inputIndex := big.NewInt(0).SetBytes([]byte(id[66:130]))

// 	contextCh := <-FetchContext(maxBlockNumber)
// 	if contextCh.err != nil {
// 		return nil, contextCh.err
// 	}
// 	context := contextCh.context

// 	// check if out of epoch's scope
// 	if context.epoch.Cmp(&context.currentEpoch) == 1 {
// 		error := fmt.Sprintf(
// 			"Requested data beyond current epoch '%s'"+
// 				" (data estimated to belong to epoch '%s')",
// 			context.currentEpoch.String(),
// 			context.epoch.String(),
// 		)
// 		slog.Error(error)
// 		return &FetchResponse{status: http.StatusForbidden, data: nil}, nil
// 	}

// 	// check if input exists at specified block

// 	// fetch specified input
// 	// - input is already known to exist: poll GraphQL until we find it there

// 	return nil, nil
// }

type HttpCustomError struct {
	status uint
	body   *string
}

func (m *HttpCustomError) Error() string {
	return "HTTP error with status " + strconv.Itoa(int(m.status)) + " and body " + *m.body
}
func (m *HttpCustomError) Status() uint {
	return m.status
}
func (m *HttpCustomError) Body() *string {
	return m.body
}

func (r *rollupAPI) fetchExpresso(id string) (*string, *HttpCustomError) {
	if len(id) != INPUT_FETCH || id[:2] != "0x" {
		err := fmt.Sprintf("Invalid id %s: : must be a hex string with 32 bytes for maxBlockNumber and 32 bytes for espressoBlockHeight", id)
		slog.Error(err)
		return nil, &HttpCustomError{status: http.StatusBadRequest}
	}

	maxBlockNumber := big.NewInt(0).SetBytes([]byte(id[2:66]))
	espressoBlockHeight := big.NewInt(0).SetBytes([]byte(id[66:130]))

	context, err := r.fetchContext(maxBlockNumber)

	if err != nil {
		return nil, &HttpCustomError{status: http.StatusInternalServerError}
	}

	return nil, nil
}

func (r *rollupAPI) Fetcher(request GioJSONRequestBody) (*string, *HttpCustomError) {
	var expresso uint16 = 2222

	if request.Domain == expresso {
		return r.fetchExpresso(request.Id)
	}

	unsupported := "Unsupported domain"
	return nil, &HttpCustomError{status: http.StatusBadRequest, body: &unsupported}
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

	fetch, err := r.Fetcher(request)

	if err != nil {
		return ctx.String(int(err.Status()), err.Error())
	}

	return nil
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
		input := r.model.FinishAndGetNext(accepted)
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
			BlockNumber:    input.BlockNumber,
			InputIndex:     uint64(input.Index),
			MsgSender:      hexutil.Encode(input.MsgSender[:]),
			BlockTimestamp: uint64(input.Timestamp.Unix()),
			Payload:        hexutil.Encode(input.Payload),
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
