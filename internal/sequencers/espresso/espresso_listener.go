package espresso

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	cModel "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/dataavailability"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/common"
)

type EspressoListener struct {
	espressoAPI     *dataavailability.EspressoAPI
	namespace       uint64
	InputRepository *cRepos.InputRepository
	fromBlock       uint64
	InputterWorker  *inputter.InputterWorker
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func NewEspressoListener(namespace uint64, repository *cRepos.InputRepository, fromBlock uint64, w *inputter.InputterWorker) *EspressoListener {
	return &EspressoListener{namespace: namespace, InputRepository: repository, fromBlock: fromBlock, InputterWorker: w}
}

func (e EspressoListener) getBaseUrl() string {
	url := os.Getenv("ESPRESSO_URL")
	if url == "" {
		url = "https://query.cappuccino.testnet.espresso.network/"
	}
	return url
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
				l1FinalizedPrevHeight := getL1FinalizedHeight(previousBlockHeight)
				l1FinalizedCurrentHeight := getL1FinalizedHeight(currentBlockHeight)
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
				msgSender, typedData, err := ExtractSigAndData(string(transaction))
				if err != nil {
					return err
				}
				// type EspressoMessage struct {
				// 	nonce   uint32 `json:"nonce"`
				// 	payload string `json:"payload"`
				// }
				nonce := int64(typedData.Message["nonce"].(float64)) // by default, JSON number is float64
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

				_, err = e.InputRepository.Create(ctx, cModel.AdvanceInput{
					Index:          int(index),
					MsgSender:      msgSender,
					Payload:        []byte(payload),
					BlockNumber:    getL1FinalizedHeight(currentBlockHeight),
					BlockTimestamp: getL1FinalizedTimestamp(currentBlockHeight),
				})
				if err != nil {
					return err
				}
			}
		}
	}
}
