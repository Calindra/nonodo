// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the bindings for the rollup OpenAPI spec.
package rollup

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ../../api/rollup.yaml

import (
	"fmt"
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
)

func computeEpoch(blockNumber *big.Int, epochDuration *big.Int) (*big.Int, error) {
	// TODO: try to mimic current Authority epoch computation
	if epochDuration == nil {
		return nil, fmt.Errorf("Invalid epochDuration")
	} else {
		result := new(big.Int).Div(blockNumber, epochDuration)
		return result, nil
	}
}

func fetchInputBoxNumber(inputIndex *big.Int) (*big.Int, error) {
	return nil, nil
}

func FetchCurrentInput() (*big.Int, error) {
	// retrieve total number of inputs
	return nil, nil
}

func waitForBlock(blockNumber *big.Int) error {
	fmt.Println("Waiting for block", blockNumber)

	// poll until block is reached

	return nil
}

func getEpochDuration() (*big.Int, error) {

	EPOCH_DURATION := os.Getenv("EPOCH_DURATION")
	var epochDuration *big.Int
	if EPOCH_DURATION != "" {
		i, err := strconv.ParseInt(EPOCH_DURATION, 10, 64)
		if err != nil {
			return nil, err
		}
		epochDuration = big.NewInt(i)
	} else {
		epochDuration = big.NewInt(86400)
	}

	return epochDuration, nil
}

func FetchContext(blockNumber *big.Int) <-chan FetchInputBoxContextOrError {
	result := make(chan FetchInputBoxContextOrError)
	defer close(result)

	epochDuration, err := getEpochDuration()
	if err != nil {
		panic(err)
	}

	currentInput, err := FetchCurrentInput()

	if err != nil {
		result <- FetchInputBoxContextOrError{context: nil, err: err}
		return result
	}

	currentInputBlockNumber, err := fetchInputBoxNumber(currentInput)
	if err != nil {
		result <- FetchInputBoxContextOrError{context: nil, err: err}
		return result
	}
	currentEpoch, err := computeEpoch(currentInputBlockNumber, epochDuration)
	if err != nil {
		result <- FetchInputBoxContextOrError{context: nil, err: err}
		return result
	}
	epoch, err := computeEpoch(blockNumber, epochDuration)
	if err != nil {
		result <- FetchInputBoxContextOrError{context: nil, err: err}
		return result
	}

	var context FetchInputBoxContext = FetchInputBoxContext{
		blockNumber:             *blockNumber,
		epoch:                   *epoch,
		currentInput:            *currentInput,
		currentInputBlockNumber: *currentInputBlockNumber,
		currentEpoch:            *currentEpoch,
	}

	if epoch.Cmp(currentEpoch) != 1 {
		waitForBlock(blockNumber)
	}

	result <- FetchInputBoxContextOrError{context: &context, err: nil}
	return result
}

func FetchInputBox(id string) (*FetchResponse, error) {
	if len(id) != INPUT_BOX_SIZE || id[:2] != "0x" {
		error := fmt.Sprintf("Invalid id %s box id", id)
		fmt.Println(error)
		return &FetchResponse{status: http.StatusNotFound, data: nil}, nil
	}
	maxBlockNumber := big.NewInt(0).SetBytes([]byte(id[2:66]))
	inputIndex := big.NewInt(0).SetBytes([]byte(id[66:130]))

	contextCh := <-FetchContext(maxBlockNumber)
	if contextCh.err != nil {
		return nil, contextCh.err
	}
	context := contextCh.context

	// check if out of epoch's scope
	if context.epoch.Cmp(&context.currentEpoch) == 1 {
		error := fmt.Sprintf("Requested data beyond current epoch '%s' (data estimated to belong to epoch '%s')", context.currentEpoch.String(), context.epoch.String())
		fmt.Println(error)
		return &FetchResponse{status: http.StatusForbidden, data: nil}, nil
	}

	// check if input exists at specified block

	// fetch specified input
	// - input is already known to exist: poll GraphQL until we find it there

	return nil, nil
}

func (r *rollupAPI) Fetcher(request GioJSONRequestBody) (*FetchResponse, error) {

	return nil, fmt.Errorf("Unsupported domain")
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

	payload, err := hexutil.Decode(request.Id)
	if err != nil {
		return ctx.String(
			http.StatusBadRequest,
			"Error decoding gio request payload,"+
				"payload must be in Ethereum hex binary format",
		)
	}

	// data := make([]byte, len(payload))
	// copy(data, payload)

	fmt.Println("Gio request received with payload:", payload)

	return fmt.Errorf("not implemented")
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
