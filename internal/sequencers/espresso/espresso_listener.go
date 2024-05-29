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
	slog.Debug("ctx", "ctx", ctx)
	currentBlockHeight := uint64(0)
	// main polling loop
	for {
		lastEspressoBlockHeight, err := e.client.FetchLatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		if lastEspressoBlockHeight == currentBlockHeight {
			var delay time.Duration = 500
			time.Sleep(delay * time.Millisecond)
			continue
		}
		transactions, err := e.client.FetchTransactionsInBlock(ctx, lastEspressoBlockHeight, e.namespace)
		if err != nil {
			return err
		}
		tot := len(transactions.Transactions)
		for i := 0; i < tot; i++ {
			transaction := transactions.Transactions[i]
			slog.Debug("transaction", "t", transaction)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
