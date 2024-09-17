package avail

import (
	"context"

	"github.com/calindra/nonodo/internal/supervisor"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	config "github.com/centrifuge/go-substrate-rpc-client/v4/config"
)

type AvailListener struct{}

func NewAvailListener() supervisor.Worker {
	return &AvailListener{}
}

func (a AvailListener) String() string {
	return "avail_listener"
}

func (a AvailListener) Start(ctx context.Context, ready chan<- struct{}) error {
	_, err := a.connect()
	if err != nil {
		return err
	}
	ready <- struct{}{}
	return nil
}

func (a AvailListener) connect() (*gsrpc.SubstrateAPI, error) {
	// uses env RPC_URL for connecting
	cfg := config.Default()

	client, err := gsrpc.NewSubstrateAPI(cfg.RPCURL)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (a AvailListener) watchNewTransactions(ctx context.Context) error {
	return nil
}
