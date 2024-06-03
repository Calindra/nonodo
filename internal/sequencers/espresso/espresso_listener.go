package espresso

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/calindra/nonodo/internal/dataavailability"
	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
)

type EspressoListener struct {
	espressoAPI     *dataavailability.EspressoAPI
	namespace       uint64
	InputRepository *model.InputRepository
	fromBlock       uint64
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func NewEspressoListener(namespace uint64, repository *model.InputRepository, fromBlock uint64) *EspressoListener {
	return &EspressoListener{namespace: namespace, InputRepository: repository, fromBlock: fromBlock}
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

	var mapToDeduplicate []map[string]bool
	mapToDeduplicate = make([]map[string]bool, 3)
	for i := range mapToDeduplicate {
		mapToDeduplicate[i] = make(map[string]bool)
	}
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
			iMap := currentBlockHeight % 3
			dMap := (currentBlockHeight + 2) % 3
			mapToDeduplicate[dMap] = make(map[string]bool)
			slog.Debug("Espresso:", "currentBlockHeight", currentBlockHeight)
			transactions, err := e.espressoAPI.FetchTransactionsInBlock(ctx, currentBlockHeight, e.namespace)
			if err != nil {
				return err
			}
			tot := len(transactions.Transactions)
			slog.Debug("Espresso:", "transactionsLen", tot)
			for i := 0; i < tot; i++ {
				transaction := transactions.Transactions[i]
				key := common.Bytes2Hex(transaction)
				slog.Debug("transaction", "currentBlockHeight", currentBlockHeight, "transaction", key)
				if mapToDeduplicate[iMap][key] || mapToDeduplicate[(iMap+1)%3][key] {
					slog.Debug("Espresso: duplicated", "transaction", transaction)
					continue
				}
				slog.Debug("not duplicated")
				mapToDeduplicate[(iMap+1)%3][key] = true
				// transform and add to InputRepository
				index, err := e.InputRepository.Count(nil)
				if err != nil {
					return err
				}
				_, err = e.InputRepository.Create(model.AdvanceInput{
					Index:       int(index),
					Payload:     transaction,
					BlockNumber: currentBlockHeight,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}
