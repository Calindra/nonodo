// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This package contains the bindings for the inspect OpenAPI spec.
package inspect

import (
	"io"
	"net/http"
	"net/url"
	"time"

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
	GetInspectInput(index int) model.InspectInput
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
func (a *inspectAPI) InspectPost(c echo.Context) error {
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
func (a *inspectAPI) Inspect(c echo.Context, _ string) error {
	uri := c.Request().RequestURI[9:] // remove '/inspect/'
	payload, err := url.QueryUnescape(uri)
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
		input := a.model.GetInspectInput(index)
		if input.Status != model.CompletionStatusUnprocessed {
			resp := convertInput(input)
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
func convertInput(input model.InspectInput) InspectResult {
	var status CompletionStatus
	switch input.Status {
	case model.CompletionStatusUnprocessed:
		panic("impossible")
	case model.CompletionStatusAccepted:
		status = Accepted
	case model.CompletionStatusRejected:
		status = Rejected
	case model.CompletionStatusException:
		status = Exception
	default:
		panic("invalid completion status")
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
	}
}
