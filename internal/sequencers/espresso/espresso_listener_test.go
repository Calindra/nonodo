package espresso

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/EspressoSystems/espresso-sequencer-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type EspressoListenerSuite struct {
	suite.Suite
}

func (s *EspressoListenerSuite) XTestSendTransaction() {
	// this test is just to understand the api
	ctx := context.Background()
	url := "https://query.cappuccino.testnet.espresso.network/"
	// espressoClient := client.NewClient(url)
	tx := types.Transaction{
		Namespace: 10008,                        // any number...
		Payload:   common.Hex2Bytes("deadbeef"), // any payload...
	}
	// err := espressoClient.SubmitTransaction(ctx, tx)
	// s.NoError(err)

	// the func above is a copy from espressoClient.SubmitTransaction
	txHash, err := submitTransactionWithResp(ctx, http.DefaultClient, url, tx)
	s.NoError(err)
	s.NotEmpty(txHash)
}

func submitTransactionWithResp(ctx context.Context, client *http.Client, baseUrl string, tx types.Transaction) (string, error) {
	marshalled, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", baseUrl+"submit/submit", bytes.NewBuffer(marshalled))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return "", fmt.Errorf("received unexpected status code: %v", response.StatusCode)
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	responseBody := string(bodyBytes)
	return responseBody, nil
}

func TestEspressoListenerSuite(t *testing.T) {
	suite.Run(t, &EspressoListenerSuite{})
}
