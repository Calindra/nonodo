package avail

import (
	"context"
	"log/slog"
	"time"

	"github.com/calindra/nonodo/internal/supervisor"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

type AvailListener struct {
	FromBlock uint64
}

func NewAvailListener(fromBlock uint64) supervisor.Worker {
	return AvailListener{
		FromBlock: fromBlock,
	}
}

func (a AvailListener) String() string {
	return "avail_listener"
}

func (a AvailListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	client, err := a.connect(ctx)
	if err != nil {
		slog.Error("Avail", "Error connecting to Avail", err)
		return err
	}
	return a.watchNewTransactions(ctx, client)
}

func (a AvailListener) connect(ctx context.Context) (*gsrpc.SubstrateAPI, error) {
	// uses env RPC_URL for connecting
	// cfg := config.Default()

	// cfg := config.Config{}
	// err := cfg.GetConfig("config.json")
	// if err != nil {
	// 	return nil, err
	// }
	rpcURL := "wss://turing-rpc.avail.so/ws"

	errCh := make(chan error)
	clientCh := make(chan *gsrpc.SubstrateAPI)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
			default:
				client, err := gsrpc.NewSubstrateAPI(rpcURL)
				if err != nil {
					slog.Error("Avail", "Error connecting to Avail client", err)
					slog.Info("Avail reconnecting client", "retryInterval", retryInterval)
					time.Sleep(retryInterval)
				} else {
					clientCh <- client
					return
				}

			}
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
	case client := <-clientCh:
		return client, nil
	}
}

const retryInterval = 5 * time.Second

func (a AvailListener) watchNewTransactions(ctx context.Context, client *gsrpc.SubstrateAPI) error {
	latestBlock := a.FromBlock
	var index uint = 0

	for {
		if latestBlock == 0 {
			block, err := client.RPC.Chain.GetHeaderLatest()
			if err != nil {
				slog.Error("Avail", "Error getting latest block hash", err)
				slog.Info("Avail reconnecting", "retryInterval", retryInterval)
				time.Sleep(retryInterval)
				continue
			}

			slog.Info("Avail", "Set last block", block.Number)
			latestBlock = uint64(block.Number)
		}

		subscription, err := client.RPC.Chain.SubscribeNewHeads()
		if err != nil {
			slog.Error("Avail", "Error subscribing to new heads", err)
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
			continue
		}
		defer subscription.Unsubscribe()

		errCh := make(chan error)

		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case err := <-subscription.Err():
					errCh <- err
					return
				case i := <-subscription.Chan():
					index++

					slog.Info("Avail", "index", index, "Chain is at block", i.Number)

					blockHash, err := client.RPC.Chain.GetBlockHash(latestBlock)
					if err != nil {
						errCh <- err
						return
					}
					block, err := client.RPC.Chain.GetBlock(blockHash)
					if err != nil {
						errCh <- err
						return
					}

					for extId, ext := range block.Block.Extrinsics {
						appID := ext.Signature.AppID.Int64()

						json, err := ext.MarshalJSON()
						if err != nil {
							slog.Error("Avail", "Error marshalling extrinsic to JSON", err)
							continue
						}
						strJSON := string(json)
						args := string(ext.Method.Args)

						slog.Debug("Avail Extrinsic", "appID", appID, "index", index, "extId", extId, "args", args, "json", strJSON)
					}

					latestBlock += 1
				}
			}
		}()

		err = <-errCh
		subscription.Unsubscribe()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			slog.Error("Avail", "Error", err)
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
		} else {
			return nil
		}
	}
}
