package paio

import (
	"encoding/json"
	"log/slog"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/sequencers/avail"
	"github.com/labstack/echo/v4"
)

//go:generate go run github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen -config=oapi.yaml ./oapi-paio.yaml

type PaioAPI struct {
	availClient *avail.AvailClient
}

// SendTransaction implements ServerInterface.
func (p *PaioAPI) SendTransaction(ctx echo.Context) error {
	var request SendTransactionJSONRequestBody
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
	hash, err := p.availClient.DefaultSubmit(string(jsonPayload))
	if err != nil {
		slog.Error("Error DefaultSubmit message:", "err", err)
		return err
	}
	ctx.String(200, hash.Hex())
	return nil
}

// Register the Paio API to echo
func Register(e *echo.Echo, availClient *avail.AvailClient) {
	var paioAPI ServerInterface = &PaioAPI{availClient}
	RegisterHandlers(e, paioAPI)
}
