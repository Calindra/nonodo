package model

import (
	"net/http"
	"os"

	// opt_client "github.com/ethereum-optimism/optimism/op-service/client"
	// eth_log "github.com/ethereum/go-ethereum/log"
	// nxt_log "golang.org/x/exp/slog"
	"github.com/labstack/echo/v4"
)

type SyscoinClient struct {
	client   *http.Client
	endpoint string
}

func NewSyscoinClient() *SyscoinClient {
	// example: https://poda.syscoin.org/vh/06310294ee0af7f1ae4c8e19fa509264565fa82ba8c82a7a9074b2abf12a15d9
	url := "https://poda.syscoin.org/vh"

	return &SyscoinClient{
		client:   http.DefaultClient,
		endpoint: url,
	}
}

// func (n *NonodoModel) ShowTransaction() {
// 	handler := nxt_log.Default().Handler()
// 	log := eth_log.NewLogger(handler)

// 	log.Info("Hello, World!")

// 	// Create a new client if it doesn't exist
// 	// if n.http_syscoin_client == nil {
// 	// 	client := opt_client.NewBasicHTTPClient("http://localhost:8080", log)
// 	// 	n.http_syscoin_client = client
// 	// }

// 	// return client
// }

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
