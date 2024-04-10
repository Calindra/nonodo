package convenience

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"strings"

	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type VoucherExecListener struct {
	Provider           string
	ApplicationAddress common.Address
	AbiString          string
	EventName          string
	ConvenienceService *services.ConvenienceService
}

func NewExecListener(
	provider string,
	applicationAddress common.Address,
	convenienceService *services.ConvenienceService,
) VoucherExecListener {
	return VoucherExecListener{
		ConvenienceService: convenienceService,
		Provider:           provider,
		ApplicationAddress: applicationAddress,
		EventName:          "VoucherExecuted",
		AbiString: `[
			{
			  "anonymous": false,
			  "inputs": [
				{
				  "indexed": false,
				  "internalType": "uint256",
				  "name": "voucherId",
				  "type": "uint256"
				}
			  ],
			  "name": "VoucherExecuted",
			  "type": "event"
			}
		]`,
	}
}

// on event callback
func (x VoucherExecListener) OnEvent(
	eventValues []interface{},
	timestamp,
	blockNumber uint64,
) error {
	if len(eventValues) != 1 {
		return fmt.Errorf("wrong event values length != 1")
	}
	voucherId, ok := eventValues[0].(*big.Int)
	if !ok {
		return fmt.Errorf("cannot cast voucher id to big.Int")
	}

	// Extract voucher and input using bit masking and shifting
	var bitsToShift uint = 128
	var maxHexBytes uint64 = 0xFFFFFFFFFFFFFFFF
	bitMask := new(big.Int).SetUint64(maxHexBytes)
	voucher := new(big.Int).Rsh(voucherId, bitsToShift)
	input := new(big.Int).And(voucherId, bitMask)

	// Print the extracted voucher and input
	slog.Debug("Decoded voucher params", "voucher", voucher, "input", input)

	// Print decoded event data
	slog.Debug("Voucher Executed", "voucherId", voucherId.String())

	ctx := context.Background()
	return x.ConvenienceService.UpdateExecuted(ctx, input.Uint64(), voucher.Uint64(), true)
}

// String implements supervisor.Worker.
func (x VoucherExecListener) String() string {
	return "ExecListener"
}

func (x VoucherExecListener) Start(ctx context.Context, ready chan<- struct{}) error {
	slog.Info("Connecting to", "provider", x.Provider)
	client, err := ethclient.DialContext(ctx, x.Provider)
	if err != nil {
		return fmt.Errorf("execlistener: dial: %w", err)
	}
	ready <- struct{}{}
	return x.WatchExecutions(ctx, client)
}

func (x *VoucherExecListener) WatchExecutions(ctx context.Context, client *ethclient.Client) error {

	// ABI of your contract
	contractABI, err := abi.JSON(strings.NewReader(x.AbiString))
	if err != nil {
		slog.Error(err.Error())
	}

	// Subscribe to event
	query := ethereum.FilterQuery{
		Addresses: []common.Address{x.ApplicationAddress},
		Topics:    [][]common.Hash{{contractABI.Events[x.EventName].ID}},
	}
	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		log.Fatal(err)
		panic("unexpected subscribe error")
	}

	slog.Info("Listening for execution events...")

	// Process events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			log.Fatal(err)
			return err
		case vLog := <-logs:
			fmt.Println(vLog)

			// Get the block number of the event
			blockNumber := vLog.BlockNumber
			blockNumberBigInt := big.NewInt(int64(blockNumber))
			// Fetch the block information
			block, err := client.BlockByNumber(context.Background(), blockNumberBigInt)
			if err != nil {
				slog.Error(err.Error())
				continue
			}

			// Extract the timestamp from the block
			timestamp := block.Time()

			values, err := contractABI.Unpack(x.EventName, vLog.Data)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			err = x.OnEvent(values, timestamp, blockNumber)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
		}
	}
}
