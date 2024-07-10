package convenience

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/big"

	"github.com/calindra/nonodo/internal/contracts"
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
	EventName          string
	ConvenienceService *services.ConvenienceService
	FromBlock          *big.Int
}

func NewExecListener(
	provider string,
	applicationAddress common.Address,
	convenienceService *services.ConvenienceService,
	fromBlock *big.Int,
) VoucherExecListener {
	return VoucherExecListener{
		FromBlock:          fromBlock,
		ConvenienceService: convenienceService,
		Provider:           provider,
		ApplicationAddress: applicationAddress,
		EventName:          "OutputExecuted",
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
	slog.Debug("Decoded voucher params",
		"voucher", voucher,
		"input", input,
		"blockNumber", blockNumber,
	)

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

func (x *VoucherExecListener) ReadPastExecutions(client *ethclient.Client, contractABI abi.ABI, query ethereum.FilterQuery) error {
	slog.Debug("ReadPastExecutions", "FromBlock", x.FromBlock)

	// Retrieve logs for the specified block range
	oldLogs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}
	// Process old logs
	for _, vLog := range oldLogs {
		err := x.HandleLog(vLog, client, contractABI)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
	}

	return nil
}

func (x *VoucherExecListener) WatchExecutions(ctx context.Context, client *ethclient.Client) error {

	// ABI of your contract
	abi, err := contracts.ApplicationMetaData.GetAbi()

	if abi == nil {
		return fmt.Errorf("error parsing abi")
	}
	if err != nil {
		slog.Error(err.Error())
	}
	contractABI := *abi

	// Subscribe to event
	query := ethereum.FilterQuery{
		FromBlock: x.FromBlock,
		Addresses: []common.Address{x.ApplicationAddress},
		Topics:    [][]common.Hash{{contractABI.Events[x.EventName].ID}},
	}

	err = x.ReadPastExecutions(client, contractABI, query)
	if err != nil {
		return err
	}

	// subscribe to new logs
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
			err := x.HandleLog(vLog, client, contractABI)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
		}
	}
}

func (x *VoucherExecListener) HandleLog(
	vLog types.Log,
	client *ethclient.Client,
	contractABI abi.ABI,
) error {
	timestamp, blockNumber, values, err := x.GetEventData(
		vLog,
		client,
		contractABI,
	)
	if err != nil {
		return err
	}
	err = x.OnEvent(values, timestamp, blockNumber)
	if err != nil {
		return err
	}
	return nil
}

func (x *VoucherExecListener) GetEventData(
	vLog types.Log,
	client *ethclient.Client,
	contractABI abi.ABI,
) (uint64, uint64, []interface{}, error) {
	// Get the block number of the event
	blockNumber := vLog.BlockNumber
	blockNumberBigInt := big.NewInt(int64(blockNumber))
	// Fetch the block information
	block, err := client.BlockByNumber(context.Background(), blockNumberBigInt)
	if err != nil {
		return 0, 0, nil, err
	}

	// Extract the timestamp from the block
	timestamp := block.Time()

	values, err := contractABI.Unpack(x.EventName, vLog.Data)
	if err != nil {
		return 0, 0, nil, err
	}
	return timestamp, blockNumber, values, nil
}
