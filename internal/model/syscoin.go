package model

import (
	"github.com/ethereum-optimism/optimism/op-service/client"
	ethLog "github.com/ethereum/go-ethereum/log"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slog"
)

// "github.com/EspressoSystems/espresso-sequencer-go/client"

func Call() error {
	handler := slog.Default().Handler()
	log := ethLog.NewLogger(handler)

	client.NewBasicHTTPClient("http://localhost:8080", log)
	return nil
}

func FetchSyscoinPoDa(ctx echo.Context, id string) (*string, *HttpCustomError) {
	return nil, nil
}
