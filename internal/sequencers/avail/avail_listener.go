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
		timestampSectionIndex := uint8(3)
		timestampMethodIndex := uint8(0)
		coreAppID := int64(0)
		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case err := <-subscription.Err():
					errCh <- err
					return
				case <-time.After(500 * time.Millisecond):
				case i := <-subscription.Chan():
					for latestBlock <= uint64(i.Number) {
						index++

						if latestBlock < uint64(i.Number) {
							slog.Debug("Avail Catching up", "Chain is at block", i.Number, "fetching block", latestBlock)
						} else {
							slog.Debug("Avail", "index", index, "Chain is at block", i.Number, "fetching block", latestBlock)
						}

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
						timestamp := uint64(0)
						for extId, ext := range block.Block.Extrinsics {
							appID := ext.Signature.AppID.Int64()
							mi := ext.Method.CallIndex.MethodIndex
							si := ext.Method.CallIndex.SectionIndex
							if appID == coreAppID && si == timestampSectionIndex && mi == timestampMethodIndex {
								timestamp = DecodeTimestamp(common.Bytes2Hex(ext.Method.Args))
							}
							slog.Debug("Block", "timestamp", timestamp, "blockNumber", latestBlock)

							if appID != DEFAULT_APP_ID {
								slog.Debug("Skipping", "appID", appID, "MethodIndex", mi, "SessionIndex", si)
								continue
							}
							json, err := ext.MarshalJSON()
							if err != nil {
								slog.Error("avail: Error marshalling extrinsic to JSON", "err", err)
								continue
							}
							strJSON := string(json)
							args := string(ext.Method.Args)
							msgSender, typedData, err := commons.ExtractSigAndData(args[2:])
							if err != nil {
								slog.Error("avail: error extracting signature and typed data", "err", err)
								continue
							}
							nonce := typedData.Message["nonce"].(uint64)
							payload, ok := typedData.Message["data"].(string)
							if !ok {
								slog.Error("avail: error extracting payload from typed data")
								continue
							}
							slog.Debug("Avail input", "msgSender", msgSender, "nonce", nonce, "payload", payload)
							slog.Debug("avail extrinsic:", "appID", appID, "index", index, "extId", extId, "args", args, "json", strJSON)
						}

						latestBlock += 1
						time.Sleep(500 * time.Millisecond)
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
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
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
