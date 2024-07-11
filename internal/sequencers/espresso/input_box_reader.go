package espresso

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tidwall/gjson"
)

func getL1FinalizedHeight(espressoBlockHeight uint64) uint64 {
	requestURL := fmt.Sprintf("https://query.cappuccino.testnet.espresso.network/availability/header/%d", espressoBlockHeight)
	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		os.Exit(1)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	// fmt.Printf("client: response body: %s\n", resBody)

	value := gjson.Get(string(resBody), "l1_finalized.number")
	return value.Uint()
}

func readInputBox(ctx context.Context, espressoPrevHeight uint64, espressoCurrentHeight uint64, w *inputter.InputterWorker) error {
	l1FinalizedPrevHeight := getL1FinalizedHeight(espressoPrevHeight)
	l1FinalizedCurrentHeight := getL1FinalizedHeight(espressoCurrentHeight)

	client, err := ethclient.DialContext(ctx, w.Provider)
	if err != nil {
		return fmt.Errorf("espresso inputter: dial: %w", err)
	}
	inputBox, err := contracts.NewInputBox(w.InputBoxAddress, client)
	if err != nil {
		return fmt.Errorf("espresso inputter: bind input box: %w", err)
	}

	err = w.ReadPastInputs(ctx, client, inputBox, l1FinalizedPrevHeight, &l1FinalizedCurrentHeight)
	if err != nil {
		return fmt.Errorf("espresso inputter: read past inputs: %w", err)
	}

	return nil
}
