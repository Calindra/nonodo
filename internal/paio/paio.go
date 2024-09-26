package paio

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/labstack/echo/v4"
)

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ./oapi-paio.yaml

type PaioAPI struct {
	availClient     *avail.AvailClient
	inputRepository *repository.InputRepository
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
	filters = append(filters, &model.ConvenienceFilter{
		Field: &msgSenderField,
		Eq:    &request.MsgSender,
	})

	typeField := "Type"
	availType := "Avail"
	filters = append(filters, &model.ConvenienceFilter{
		Field: &typeField,
		Eq:    &availType,
	})
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

// Register the Paio API to echo
func Register(e *echo.Echo, availClient *avail.AvailClient, inputRepository *repository.InputRepository) {
	var paioAPI ServerInterface = &PaioAPI{availClient, inputRepository}
	RegisterHandlers(e, paioAPI)
}
