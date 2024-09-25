package avail

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/supervisor"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
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
	rpcURL, haveURL := os.LookupEnv("AVAIL_RPC_URL")
	if !haveURL {
		rpcURL = DEFAULT_AVAIL_RPC_URL
	}

	errCh := make(chan error)
	clientCh := make(chan *gsrpc.SubstrateAPI)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
			default:
				client, err := NewSubstrateAPICtx(ctx, rpcURL)
				if err != nil {
					slog.Error("Avail", "Error connecting to Avail client", err)
					slog.Info("Avail reconnecting client", "retryInterval", retryConnectionInterval)
					time.Sleep(retryConnectionInterval)
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

const retryConnectionInterval = 5 * time.Second
const intervalNextBlock = 500 * time.Millisecond

func (a AvailListener) handleData(header types.Header, client *gsrpc.SubstrateAPI, index uint64, fromBlock uint64) error {
	var (
		timestampSectionIndex       = 3
		timestampMethodIndex        = 0
		coreAppID             int64 = 0
	)

	if fromBlock < uint64(header.Number) {
		slog.Debug("Avail Catching up", "Chain is at block", header.Number, "fetching block", fromBlock)
	} else {
		slog.Debug("Avail", "index", index, "Chain is at block", header.Number, "fetching block", fromBlock)
	}

	blockHash, err := client.RPC.Chain.GetBlockHash(fromBlock)
	if err != nil {
		return err
	}
	block, err := client.RPC.Chain.GetBlock(blockHash)
	if err != nil {
		return err
	}
	timestamp := uint64(0)
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()
		mi := ext.Method.CallIndex.MethodIndex
		si := ext.Method.CallIndex.SectionIndex
		if appID == coreAppID && si == uint8(timestampSectionIndex) && mi == uint8(timestampMethodIndex) {
			timestamp = DecodeTimestamp(common.Bytes2Hex(ext.Method.Args))
		}
		slog.Debug("Block", "timestamp", timestamp, "blockNumber", fromBlock)

		if appID != DEFAULT_APP_ID {
			slog.Debug("Skipping", "appID", appID, "MethodIndex", mi, "SessionIndex", si)
			return nil
		}
		// json, err := ext.MarshalJSON()
		// if err != nil {
		// 	slog.Error("avail: Error marshalling extrinsic to JSON", "err", err)
		// 	continue
		// }
		// strJSON := string(json)
		args := string(ext.Method.Args)
		msgSender, typedData, err := commons.ExtractSigAndData(args)
		if err != nil {
			slog.Error("avail: error extracting signature and typed data", "err", err)
			return nil
		}
		dappAddress := typedData.Message["app"].(string)
		nonce := typedData.Message["nonce"].(string)
		maxGasPrice := typedData.Message["max_gas_price"].(string)
		payload, ok := typedData.Message["data"].(string)
		if !ok {
			slog.Error("avail: error extracting data from message")
			return nil
		}
		slog.Debug("Avail input",
			"dappAddress", dappAddress,
			"msgSender", msgSender,
			"nonce", nonce,
			"maxGasPrice", maxGasPrice,
			"payload", payload,
		)
		// slog.Debug("avail extrinsic:", "appID", appID, "index", index, "extId", extId, "args", args, "json", strJSON)
	}

	return nil
}

func (a AvailListener) ReadPastTransactions(ctx context.Context, client *gsrpc.SubstrateAPI, endBlock uint64, finished chan<- struct{}, errCh chan<- error) {
	for block := a.FromBlock; block < endBlock; block++ {
		index := block - a.FromBlock
		slog.Debug("Avail", "Reading past block", block)
		blockHash, err := client.RPC.Chain.GetBlockHash(block)
		if err != nil {
			errCh <- fmt.Errorf("Error getting block hash %v", err)
			continue
		}

		header, err := client.RPC.Chain.GetHeader(blockHash)
		if err != nil {
			errCh <- fmt.Errorf("Error getting block %v", err)
			continue
		}

		err = a.handleData(*header, client, index, a.FromBlock)

		if err != nil {
			errCh <- fmt.Errorf("Error handling data %v", err)
			continue
		}

	}

	finished <- struct{}{}
}

func (a AvailListener) watchNewTransactions(ctx context.Context, client *gsrpc.SubstrateAPI) error {
	var (
		finishedPast = make(chan struct{})
		errPastCh    = make(chan error)
	)

	for {
		if a.FromBlock == 0 {
			return fmt.Errorf("avail: fromBlock is 0")
			// block, err := client.RPC.Chain.GetHeaderLatest()
			// if err != nil {
			// 	slog.Error("Avail", "Error getting latest block hash", err)
			// 	slog.Info("Avail reconnecting", "retryInterval", retryConnectionInterval)
			// 	// time.Sleep(retryConnectionInterval)
			// 	continue
			// }

			// slog.Info("Avail", "Set last block", block.Number)
			// fromBlock = uint64(block.Number)
		}

		latestBlock, err := client.RPC.Chain.GetHeaderLatest()
		if err != nil {
			slog.Error("Avail", "Error getting latest block hash", err)
			slog.Info("Avail reconnecting", "retryInterval", retryConnectionInterval)
			time.Sleep(retryConnectionInterval)
			continue
		}

		slog.Info("Avail", "Set last block", latestBlock.Number)

		go func() {
			a.ReadPastTransactions(ctx, client, uint64(latestBlock.Number), finishedPast, errPastCh)

			for {
				select {
				case <-ctx.Done():
					return
				case err := <-errPastCh:
					slog.Error("Avail", "Error reading past transactions", err)
				case <-finishedPast:
					slog.Info("Avail finished reading past transactions")
					return
				}
			}
		}()

		subscription, err := client.RPC.Chain.SubscribeFinalizedHeads()
		if err != nil {
			slog.Error("Avail", "Error subscribing to new heads", err)
			slog.Info("Avail reconnecting", "retryInterval", retryConnectionInterval)
			time.Sleep(retryConnectionInterval)
			continue
		}
		defer subscription.Unsubscribe()

		errCh := make(chan error, 1)

		go func() {
			for {
				select {
				case <-ctx.Done():
					slog.Warn(">>>>>>>>>>>>>>>> Avail signal shutting down")
					errCh <- ctx.Err()
					return
				case err := <-subscription.Err():
					errCh <- err
					return

				case <-time.After(intervalNextBlock):
				case header := <-subscription.Chan():
					index := uint64(header.Number) - a.FromBlock
					slog.Debug("Avail", "index", index, "New block", header.Number)
					err := a.handleData(header, client, index, a.FromBlock)

					if err != nil {
						errCh <- err
						return
					}
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
			slog.Info("Avail reconnecting", "retryInterval", retryConnectionInterval)
			time.Sleep(retryConnectionInterval)
		} else {
			return nil
		}
	}
}

func DecodeTimestamp(hexStr string) uint64 {
	decoded, err := hex.DecodeString(padHexStringRight(hexStr))
	if err != nil {
		fmt.Println("Error decoding hex:", err)
		return 0
	}
	return decodeCompactU64(decoded)
}

// nolint
func decodeCompactU64(data []byte) uint64 {
	firstByte := data[0]
	if firstByte&0b11 == 0b00 { // Single byte (6-bit value)
		return uint64(firstByte >> 2)
	} else if firstByte&0b11 == 0b01 { // Two bytes (14-bit value)
		return uint64(firstByte>>2) | uint64(data[1])<<6
	} else if firstByte&0b11 == 0b10 { // Four bytes (30-bit value)
		return uint64(firstByte>>2) | uint64(data[1])<<6 | uint64(data[2])<<14 | uint64(data[3])<<22
	} else { // Eight bytes (64-bit value)
		return uint64(data[1]) | uint64(data[2])<<8 | uint64(data[3])<<16 | uint64(data[4])<<24 |
			uint64(data[5])<<32 | uint64(data[6])<<40 | uint64(data[7])<<48
	}
}

func padHexStringRight(hexStr string) string {
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	// Right pad with zeros to ensure it's 16 characters long (8 bytes)
	for len(hexStr) < 16 {
		hexStr += "0"
	}

	return hexStr
}
