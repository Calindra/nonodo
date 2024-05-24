package espresso

import (
	"context"
	"log/slog"
)

type EspressoListener struct {
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func (e EspressoListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
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
	return nil
}
