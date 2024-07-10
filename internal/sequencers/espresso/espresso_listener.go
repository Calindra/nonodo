package espresso

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/calindra/nonodo/internal/dataavailability"
	"github.com/calindra/nonodo/internal/model"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/common"
)

type EspressoListener struct {
	espressoAPI     *dataavailability.EspressoAPI
	namespace       uint64
	InputRepository *model.InputRepository
	fromBlock       uint64
	InputterWorker  *inputter.InputterWorker
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func NewEspressoListener(namespace uint64, repository *model.InputRepository, fromBlock uint64, w *inputter.InputterWorker) *EspressoListener {
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
				fmt.Println("Fetching InputBox between Espresso block ", previousBlockHeight, " to ", currentBlockHeight)
				readInputBox(ctx, previousBlockHeight, currentBlockHeight, e.InputterWorker)
				previousBlockHeight = currentBlockHeight + 1
			}

			slog.Debug("Espresso:", "transactionsLen", tot)
			for i := 0; i < tot; i++ {
				transaction := transactions.Transactions[i]
				slog.Debug("transaction", "currentBlockHeight", currentBlockHeight, "transaction", transaction)

				// transform and add to InputRepository
				index, err := e.InputRepository.Count(nil)
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
				fmt.Println("msg sender ", msgSender.String())
				// type EspressoMessage struct {
				// 	nonce   uint32 `json:"nonce"`
				// 	payload string `json:"payload"`
				// }
				nonce := int64(typedData.Message["nonce"].(float64)) // by default, JSON number is float64
				payload, ok := typedData.Message["payload"].(string)
				fmt.Println("nonce ", nonce)
				fmt.Println("payload ", payload)
				if !ok {
					return fmt.Errorf("type assertion error")
				}

				// update nonce maps
				// no need to consider node exits abruptly and restarts from where it left
				// app has to start `nonce` from 1 and increment by 1 for each payload
				if nonceMap[msgSender] == nonce-1 {
					nonceMap[msgSender] = nonce
					// fmt.Println("nonce is now ", nonce)
				} else {
					// nonce repeated
					fmt.Println("duplicated or incorrect nonce")
					continue
				}

				_, err = e.InputRepository.Create(model.AdvanceInput{
					Index:       int(index),
					MsgSender:   msgSender,
					Payload:     []byte(payload),
					BlockNumber: currentBlockHeight,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}
