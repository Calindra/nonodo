package model

import (
	"net/http"
	"os"

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
	// Read of file
	file, err := os.ReadFile("syscoin-poda.json")

	if err != nil {
		return nil, NewHttpCustomError(http.StatusNotFound, nil)
	}

	// Convert to string
	str := string(file)

	return &str, nil
}
