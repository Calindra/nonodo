// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the bindings for the rollup OpenAPI spec.
package rollup

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ../../api/rollup.yaml

import (
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"

	"strings"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	mdl "github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
)

const FinishRetries = 50
const FinishPollInterval = time.Millisecond * 100

// Register the rollup API to echo
func Register(e *echo.Echo, model *mdl.NonodoModel, sequencer Sequencer, applicationAddress common.Address) {
	var rollupAPI ServerInterface = &RollupAPI{model, sequencer, applicationAddress}
	RegisterHandlers(e, rollupAPI)
}

// Shared struct for request handlers.
type RollupAPI struct {
	model              *mdl.NonodoModel
	sequencer          Sequencer
	ApplicationAddress common.Address
}

type Sequencer interface {
	FinishAndGetNext(accept bool) (cModel.Input, error)
}

// Gio implements ServerInterface.
func (r *RollupAPI) Gio(ctx echo.Context) error {

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
		slog.Debug("Error in Fetcher: %s %d", err.Error(), err.Status())
		return ctx.String(int(err.Status()), err.Error())
	}

	if fetch == nil {
		return ctx.String(http.StatusNotFound, "Not found")
	}

	return ctx.JSON(http.StatusOK, fetch)
}

// Handle requests to /finish.
func (r *RollupAPI) Finish(c echo.Context) error {
	slog.Debug("/finish start handling...")
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request FinishJSONRequestBody
	if err := c.Bind(&request); err != nil {
		slog.Error("/finish bind request error", "error", err)
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
	if r.sequencer == nil {
		return c.String(http.StatusInternalServerError, "sequencer not available")
	}
	for i := 0; i < FinishRetries; i++ {
		input, err := r.sequencer.FinishAndGetNext(accepted)

		if err != nil {
			slog.Error("/finish and get next", "error", err)
			return err
		}

		if input != nil {
			resp, err := convertInput(input)

			if err != nil {
				slog.Error("/finish convert input", "error", err)
				return err
			}

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
func (r *RollupAPI) AddVoucher(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request AddVoucherJSONRequestBody
	if err := c.Bind(&request); err != nil {
		slog.Error("AddVoucher error", "error", err)
		return err
	}

	// validate fields
	destination, err := hexutil.Decode(request.Destination)
	if err != nil {
		slog.Error("invalid hex payload", "error", err)
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}
	if len(destination) != common.AddressLength {
		return c.String(http.StatusBadRequest, "invalid address length")
	}
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		slog.Error("invalid hex payload", "error", err)
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return err
	}

	if len(request.Value) != 66 { // nolint
		return fmt.Errorf("the value must be a 32-byte hex string. eg: 0x0000000000000000000000000000000000000000000000000000000000000001")
	}

	value, ok := new(big.Int).SetString(request.Value[2:], 16) // nolint
	if !ok {
		slog.Error("wrong number format", "value", request.Value[2:])
		return fmt.Errorf("wrong number format")
	}
	destinationContract := common.HexToAddress(request.Destination)
	encodedPayload, err := abiParsed.Pack("Voucher", destinationContract, value, payload)
	if err != nil {
		slog.Error("encoded payload error", "err", err)
		return err
	}

	index, err := r.model.AddVoucher(r.ApplicationAddress, common.Address(destination), request.Value, encodedPayload)
	if err != nil {
		slog.Error("AddVoucher", "err", err)
		return c.String(http.StatusInternalServerError, err.Error())
	}
	resp := IndexResponse{
		Index: uint64(index),
	}
	return c.JSON(http.StatusOK, &resp)
}

// Handle requests to /notice.
func (r *RollupAPI) AddNotice(c echo.Context) error {
	if !checkContentType(c) {
		slog.Error("invalid notice content type")
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request AddNoticeJSONRequestBody
	if err := c.Bind(&request); err != nil {
		slog.Error("invalid notice body", "error", err)
		return err
	}

	// validate fields
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		slog.Error("invalid hex payload", "payload", request.Payload)
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return err
	}

	encodedPayload, err := abiParsed.Pack("Notice", payload)
	if err != nil {
		slog.Error("Error encoding notice as abi", "err", err)
		return err
	}
	// talk to model
	slog.Debug("RollupAPI", "encodedPayload", common.Bytes2Hex(encodedPayload))
	index, err := r.model.AddNotice(encodedPayload, r.ApplicationAddress)
	if err != nil {
		slog.Error("add notice error", "err", err)
		return c.String(http.StatusForbidden, err.Error())
	}
	resp := IndexResponse{
		Index: uint64(index),
	}
	return c.JSON(http.StatusOK, &resp)
}

// Handle requests to /report.
func (r *RollupAPI) AddReport(c echo.Context) error {
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
		slog.Error("payload decoded error", "err", err)
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	// talk to model
	err = r.model.AddReport(r.ApplicationAddress, payload)
	if err != nil {
		slog.Error("add report error", "err", err)
		return c.String(http.StatusForbidden, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// Handle requests to /exception.
func (r *RollupAPI) RegisterException(c echo.Context) error {
	if !checkContentType(c) {
		return c.String(http.StatusUnsupportedMediaType, "invalid content type")
	}

	// parse body
	var request RegisterExceptionJSONRequestBody
	if err := c.Bind(&request); err != nil {
		slog.Error("parse body error", "err", err)
		return err
	}

	// validate fields
	payload, err := hexutil.Decode(request.Payload)
	if err != nil {
		slog.Error("payload decoded error", "err", err)
		return c.String(http.StatusBadRequest, "invalid hex payload")
	}

	// talk to model
	err = r.model.RegisterException(payload)
	if err != nil {
		slog.Error("register exception error", "err", err)
		return c.String(http.StatusForbidden, err.Error())
	}
	return c.NoContent(http.StatusOK)
}

// Check whether the content type is application/json.
func checkContentType(c echo.Context) bool {
	cType := c.Request().Header.Get(echo.HeaderContentType)
	return strings.HasPrefix(cType, echo.MIMEApplicationJSON)
}

func parseChainID(value string) (*big.Int, error) {
	if strings.HasPrefix(value, "0x") {
		chainId, ok := new(big.Int).SetString(value, 16) // nolint
		if ok {
			return chainId, nil
		}
		slog.Error("failed to convert chain id", "value", value)
		return nil, errors.New("failed to convert chain id")
	}
	chainId, ok := new(big.Int).SetString(value, 10) // nolint
	if ok {
		return chainId, nil
	}
	slog.Error("failed to convert chain id", "value", value)
	return nil, errors.New("failed to convert chain id")
}

// Convert model input to API type.
func convertInput(input cModel.Input) (RollupRequest, error) {
	var resp RollupRequest
	switch input := input.(type) {
	case cModel.AdvanceInput:
		chainId, err := parseChainID(input.ChainId)
		if err != nil {
			return RollupRequest{}, err
		}
		advance := Advance{
			Metadata: Metadata{
				AppContract:    input.AppContract.Hex(),
				BlockNumber:    input.BlockNumber,
				InputIndex:     uint64(input.Index),
				MsgSender:      hexutil.Encode(input.MsgSender[:]),
				BlockTimestamp: uint64(input.BlockTimestamp.Unix()),
				PrevRandao:     input.PrevRandao,
				ChainId:        chainId.Uint64(),
			},
			Payload: input.Payload,
		}
		err = resp.Data.FromAdvance(advance)
		if err != nil {
			slog.Error("failed to convert advance", "err", err)
			return RollupRequest{}, errors.New("failed to convert advance")
		}
		resp.RequestType = AdvanceState
	case mdl.InspectInput:
		inspect := Inspect{
			Payload: hexutil.Encode(input.Payload),
		}
		err := resp.Data.FromInspect(inspect)
		if err != nil {
			slog.Error("failed to convert inspect", "err", err)
			return RollupRequest{}, errors.New("failed to convert inspect")
		}
		resp.RequestType = InspectState
	default:
		return RollupRequest{}, errors.New("invalid input from model")
	}
	return resp, nil
}
