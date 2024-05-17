package model

import (
	"io"
	"log/slog"
	"net/http"

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
	slog.Debug("Called FetchSyscoinPoDa")

	full_url := "https://poda.syscoin.org/vh/" + id

	res, err := http.Get(full_url)

	if err != nil {
		return nil, NewHttpCustomError(http.StatusInternalServerError, nil)
	}

	defer res.Body.Close()

	// Read the response body
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, NewHttpCustomError(http.StatusInternalServerError, nil)
	}

	// Convert the body to string
	str := string(body)

	slog.Debug("Called syscoin PoDa: ")

	return &str, nil
}
