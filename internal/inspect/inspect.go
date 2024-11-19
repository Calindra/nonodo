// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the bindings for the inspect OpenAPI spec.
package inspect

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	cModel "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/model"
	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
)

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ../../api/inspect.yaml

// 2^20 bytes, which is the length of the RX buffer in the Cartesi machine.
const PayloadSizeLimit = 1_048_576

// Model is the inspect interface for the nonodo model.
type Model interface {
	AddInspectInput(payload []byte) int
	GetInspectInput(index int) (model.InspectInput, error)
}

// Register the rollup API to echo
func Register(e *echo.Echo, model Model) {
	var inspectAPI ServerInterface = &inspectAPI{model}
	RegisterHandlers(e, inspectAPI)
}

// Shared struct for request handlers.
type inspectAPI struct {
	model Model
}

// Handle POST requests to /.
func (a *inspectAPI) InspectPost(c echo.Context, _ string) error {
	body := c.Request().Body
	defer body.Close()
	payload, err := io.ReadAll(body)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if len(payload) > PayloadSizeLimit {
		return c.String(http.StatusBadRequest, "Payload reached size limit")
	}
	return a.inspect(c, payload)
}

// Handle GET requests to /{payload}.
func (a *inspectAPI) Inspect(c echo.Context, appAddress string, _ string) error {
	toRemove := len(fmt.Sprintf("/inspect/%s/", appAddress))
	uri := c.Request().RequestURI[toRemove:] // remove '/inspect/<app-address>'
	payload, err := url.QueryUnescape(uri)
	slog.Debug("/inspect", "payload", payload)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	return a.inspect(c, []byte(payload))
}

// Send the inspect input to the model and wait until it is completed.
func (a *inspectAPI) inspect(c echo.Context, payload []byte) error {
	// Send inspect to the model
	index := a.model.AddInspectInput(payload)

	// Poll the model for response
	const pollFrequency = 33 * time.Millisecond
	ticker := time.NewTicker(pollFrequency)
	defer ticker.Stop()
	for {
		input, err := a.model.GetInspectInput(index)

		if err != nil {
			return err
		}

		if input.Status != cModel.CompletionStatusUnprocessed {
			resp, err := convertInput(input)

			if err != nil {
				slog.Error("Error converting input", "Error", err)
				return err
			}
			return c.JSON(http.StatusOK, &resp)
		}
		select {
		case <-c.Request().Context().Done():
			return c.Request().Context().Err()
		case <-ticker.C:
		}
	}
}

// Convert model input to API type.
func convertInput(input model.InspectInput) (InspectResult, error) {
	var status CompletionStatus
	switch input.Status {
	case cModel.CompletionStatusUnprocessed:
		return InspectResult{}, errors.New("impossible")
	case cModel.CompletionStatusAccepted:
		status = Accepted
	case cModel.CompletionStatusRejected:
		status = Rejected
	case cModel.CompletionStatusException:
		status = Exception
	case cModel.CompletionStatusMachineHalted:
		status = MachineHalted
	case cModel.CompletionStatusCycleLimitExceeded:
		status = CycleLimitExceeded
	case cModel.CompletionStatusTimeLimitExceeded:
		status = TimeLimitExceeded
	case cModel.CompletionStatusPayloadLengthLimitExceeded:
		status = "PAYLOAD_LENGTH_LIMIT_EXCEEDED"
	default:
		return InspectResult{}, errors.New("invalid completion status")
	}

	var reports []Report
	for _, report := range input.Reports {
		reports = append(reports, Report{
			Payload: hexutil.Encode(report.Payload),
		})
	}

	return InspectResult{
		Status:              status,
		Reports:             reports,
		ExceptionPayload:    hexutil.Encode(input.Exception),
		ProcessedInputCount: input.ProcessedInputCount,
	}, nil
}
