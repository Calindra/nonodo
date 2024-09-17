package avail

import (
	"context"
	"log/slog"

	"github.com/calindra/nonodo/internal/supervisor"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

type AvailListener struct{}

func NewAvailListener() supervisor.Worker {
	return AvailListener{}
}

func (a AvailListener) String() string {
	return "avail_listener"
}

func (a AvailListener) Start(ctx context.Context, ready chan<- struct{}) error {
	client, err := a.connect()
	if err != nil {
		return err
	}
	ready <- struct{}{}
	return a.watchNewTransactions(ctx, client)
}

func (a AvailListener) connect() (*gsrpc.SubstrateAPI, error) {
	// uses env RPC_URL for connecting
	// cfg := config.Default()
	RPCUrl := "https://turing-rpc.avail.so/rpc"

	client, err := gsrpc.NewSubstrateAPI(RPCUrl)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (a AvailListener) watchNewTransactions(ctx context.Context, client *gsrpc.SubstrateAPI) error {
	waitForBlocks := 5
	count := 0

	subscription, err := client.RPC.Chain.SubscribeNewHeads()

	if err != nil {
		return err
	}

	for i := range subscription.Chan() {
		count++
		slog.Info("Avail", "Block number: %v\n", i.Number)
		if count == waitForBlocks {
			break
		}
	}

	return nil
}
