package espresso

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/dataavailability"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tidwall/gjson"
)

type EspressoListener struct {
	espressoAPI     *dataavailability.EspressoAPI
	espressoUrl     string
	namespace       uint64
	InputRepository *cRepos.InputRepository
	fromBlock       uint64
	InputterWorker  *inputter.InputterWorker
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func NewEspressoListener(espressoUrl string, namespace uint64, repository *cRepos.InputRepository, fromBlock uint64, w *inputter.InputterWorker) *EspressoListener {
	return &EspressoListener{espressoUrl: espressoUrl, namespace: namespace, InputRepository: repository, fromBlock: fromBlock, InputterWorker: w}
}

func (e EspressoListener) getBaseUrl() string {
	return e.espressoUrl
}

func (e EspressoListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	url := e.getBaseUrl()
	e.espressoAPI = dataavailability.NewEspressoAPI(ctx, &url)
	slog.Info("espresso listener started")
	return e.watchNewTransactions(ctx)
}

func (e EspressoListener) watchNewTransactions(ctx context.Context) error {
	slog.Debug("Espresso: watchNewTransactions", "fromBlock", e.fromBlock)
	currentBlockHeight := e.fromBlock
	previousBlockHeight := currentBlockHeight

	// keep track of msgSender -> nonce
	nonceMap := make(map[common.Address]int64)

	// main polling loop
	for {
		slog.Debug("Espresso: fetchLatestBlockHeight...")
		lastEspressoBlockHeight, err := e.espressoAPI.FetchLatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		slog.Debug("Espresso:", "lastEspressoBlockHeight", lastEspressoBlockHeight)
		if lastEspressoBlockHeight == currentBlockHeight {
			var delay time.Duration = 500
			time.Sleep(delay * time.Millisecond)
			continue
		}
		for ; currentBlockHeight < lastEspressoBlockHeight; currentBlockHeight++ {
			slog.Debug("Espresso:", "currentBlockHeight", currentBlockHeight)
			transactions, err := e.espressoAPI.FetchTransactionsInBlock(ctx, currentBlockHeight, e.namespace)
			if err != nil {
				return err
			}
			tot := len(transactions.Transactions)

			if tot > 0 {
				l1FinalizedPrevHeight := e.getL1FinalizedHeight(previousBlockHeight)
				l1FinalizedCurrentHeight := e.getL1FinalizedHeight(currentBlockHeight)
				slog.Debug("L1 finalized", "from", l1FinalizedPrevHeight, "to", l1FinalizedCurrentHeight)

				// read L1 if there might be update
				if l1FinalizedCurrentHeight > l1FinalizedPrevHeight || previousBlockHeight == e.fromBlock {
					slog.Debug("Fetching InputBox between Espresso blocks", "from", previousBlockHeight, "to", currentBlockHeight)
					err = readInputBox(ctx, l1FinalizedPrevHeight, l1FinalizedCurrentHeight, e.InputterWorker)
					if err != nil {
						return err
					}
				}
				previousBlockHeight = currentBlockHeight + 1
			}

			slog.Debug("Espresso:", "transactionsLen", tot)
			for i := 0; i < tot; i++ {
				transaction := transactions.Transactions[i]
				slog.Debug("transaction", "currentBlockHeight", currentBlockHeight, "transaction", transaction)

				ctx := context.Background()
				// transform and add to InputRepository
				index, err := e.InputRepository.Count(ctx, nil)
				if err != nil {
					return err
				}

				// assume the following encoding
				// transaction = JSON.stringify({
				//		 	signature,
				//		 	typedData: btoa(JSON.stringify(typedData)),
				//		 })
				msgSender, typedData, signature, err := commons.ExtractSigAndData(string(transaction))
				if err != nil {
					return err
				}
				// type EspressoMessage struct {
				// 	nonce   uint32 `json:"nonce"`
				// 	payload string `json:"payload"`
				// }
				nonceField := typedData.Message["nonce"]

				var nonce int64
				// Request tx is coming from nonodo, the nonce at this point returns as string always
				if _, ok := nonceField.(string); ok {
					strNonce := nonceField.(string)
					parsedNonce, err := strconv.ParseInt(strNonce, 10, 64)
					if err != nil {
						return fmt.Errorf("error converting nonce from string to int64: %v", err)
					}
					nonce = parsedNonce
				} else if _, ok := nonceField.(float64); ok {
					// When coming from other sources, the nonce at this point is a float
					nonce = int64(nonceField.(float64))
				} else {
					//
					return fmt.Errorf("error converting nonce: %T", nonceField)
				}

				payload, ok := typedData.Message["payload"].(string)
				if !ok {
					return fmt.Errorf("type assertion error")
				}
				slog.Debug("Espresso input", "msgSender", msgSender, "nonce", nonce, "payload", payload)

				// update nonce maps
				// no need to consider node exits abruptly and restarts from where it left
				// app has to start `nonce` from 1 and increment by 1 for each payload
				if nonceMap[msgSender] == nonce-1 {
					nonceMap[msgSender] = nonce
				} else {
					// nonce repeated
					slog.Debug("duplicated or incorrect nonce", "nonce", nonce)
					continue
				}

				payloadBytes := []byte(payload)
				if strings.HasPrefix(payload, "0x") {
					payload = payload[2:] // remove 0x
					payloadBytes, err = hex.DecodeString(payload)
					if err != nil {
						return err
					}
				}

				_, err = e.InputRepository.Create(ctx, cModel.AdvanceInput{
					Index:                  int(index),
					MsgSender:              msgSender,
					Payload:                payloadBytes,
					BlockNumber:            e.getL1FinalizedHeight(currentBlockHeight),
					BlockTimestamp:         e.getL1FinalizedTimestamp(currentBlockHeight),
					AppContract:            e.InputterWorker.ApplicationAddress,
					EspressoBlockNumber:    int(currentBlockHeight),
					EspressoBlockTimestamp: e.getEspressoTimestamp(currentBlockHeight),
					InputBoxIndex:          -1,
					Type:                   "Espresso",
					CartesiTransactionId:   string(crypto.Keccak256(signature)),
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func (e EspressoListener) readEspressoHeader(espressoBlockHeight uint64) string {
	requestURL := fmt.Sprintf("%s/availability/header/%d", e.espressoUrl, espressoBlockHeight)
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

	return string(resBody)
}

func (e EspressoListener) getL1FinalizedTimestamp(espressoBlockHeight uint64) time.Time {
	espressoHeader := e.readEspressoHeader(espressoBlockHeight)
	value := gjson.Get(espressoHeader, "l1_finalized.timestamp")
	timestampStr := value.Str
	timestampInt, err := strconv.ParseInt(timestampStr[2:], 16, 64)
	if err != nil {
		slog.Error("hex to int conversion failed", "err", err)
		os.Exit(1)
	}
	return time.Unix(timestampInt, 0)
}

func (e EspressoListener) getL1FinalizedHeight(espressoBlockHeight uint64) uint64 {
	espressoHeader := e.readEspressoHeader(espressoBlockHeight)
	value := gjson.Get(espressoHeader, "l1_finalized.number")
	return value.Uint()
}

func (e EspressoListener) getEspressoTimestamp(espressoBlockHeight uint64) time.Time {
	espressoHeader := e.readEspressoHeader(espressoBlockHeight)
	value := gjson.Get(espressoHeader, "timestamp")
	return time.Unix(value.Int(), 0)
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
