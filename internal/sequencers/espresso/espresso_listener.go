package espresso

import (
	"context"
	"log/slog"
	"time"

	"github.com/EspressoSystems/espresso-sequencer-go/client"
)

type EspressoListener struct {
	client    *client.Client
	namespace uint64
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func NewEspressoListener(namespace uint64) *EspressoListener {
	return &EspressoListener{namespace: namespace}
}

func (e EspressoListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	url := "https://query.cappuccino.testnet.espresso.network/"
	e.client = client.NewClient(url)
	e.namespace = 10008
	err := e.readPastTransactions(ctx)
	if err != nil {
		return err
	}
	slog.Info("espresso started!")
	return e.watchNewTransactions(ctx)
}

func (e EspressoListener) readPastTransactions(ctx context.Context) error {
	slog.Debug("ctx", "ctx", ctx)
	return nil
}

func (e EspressoListener) watchNewTransactions(ctx context.Context) error {
	slog.Info("Espresso: watchNewTransactions...")
	// currentBlockHeight := uint64(276228)
	currentBlockHeight := uint64(98299)

	// main polling loop
	for {
		slog.Info("Espresso: fetchLatestBlockHeight...")
		lastEspressoBlockHeight, err := e.client.FetchLatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		slog.Info("Espresso:", "lastEspressoBlockHeight", lastEspressoBlockHeight)
		if lastEspressoBlockHeight == currentBlockHeight {
			var delay time.Duration = 500
			time.Sleep(delay * time.Millisecond)
			continue
		}
		for ; currentBlockHeight < lastEspressoBlockHeight; currentBlockHeight++ {
			slog.Info("Espresso:", "currentBlockHeight", currentBlockHeight)
			transactions, err := e.client.FetchTransactionsInBlock(ctx, currentBlockHeight, e.namespace)
			if err != nil {
				return err
			}
			tot := len(transactions.Transactions)
			slog.Info("Espresso:", "transactionsLen", tot)
			for i := 0; i < tot; i++ {
				transaction := transactions.Transactions[i]
				slog.Info("transaction", "currentBlockHeight", currentBlockHeight, "transaction", transaction)
				// transform and add to InputRepository
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
