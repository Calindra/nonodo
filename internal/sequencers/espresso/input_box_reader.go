package espresso

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tidwall/gjson"
)

func getL1FinalizedTimestamp(espressoBlockHeight uint64) time.Time {
	requestURL := fmt.Sprintf("https://query.cappuccino.testnet.espresso.network/availability/header/%d", espressoBlockHeight)
	res, err := http.Get(requestURL)
	if err != nil {
		slog.Error("error making http request", "err", err)
		os.Exit(1)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("could not read response body", "err", err)
		os.Exit(1)
	}

	value := gjson.Get(string(resBody), "l1_finalized.timestamp")
	timestampStr := value.Str
	timestampInt, err := strconv.ParseInt(timestampStr[2:], 16, 64)
	if err != nil {
		slog.Error("hex to int conversion failed", "err", err)
		os.Exit(1)
	}
	return time.Unix(timestampInt, 0)
}

func getL1FinalizedHeight(espressoBlockHeight uint64) uint64 {
	requestURL := fmt.Sprintf("https://query.cappuccino.testnet.espresso.network/availability/header/%d", espressoBlockHeight)
	res, err := http.Get(requestURL)
	if err != nil {
		slog.Error("error making http request", "err", err)
		os.Exit(1)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("could not read response body", "err", err)
		os.Exit(1)
	}

	value := gjson.Get(string(resBody), "l1_finalized.number")
	return value.Uint()
}

func readInputBox(ctx context.Context, l1FinalizedPrevHeight uint64, l1FinalizedCurrentHeight uint64, w *inputter.InputterWorker) error {
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
