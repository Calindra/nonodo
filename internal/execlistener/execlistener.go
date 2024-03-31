package execlistener

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"strings"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ExecListener struct {
	Model              *model.NonodoModel
	Provider           string
	ApplicationAddress common.Address
	AbiString          string
	EventName          string
}

func NewExecListener(model *model.NonodoModel, provider string, applicationAddress common.Address) ExecListener {
	return ExecListener{
		Model:              model,
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
func (x ExecListener) OnEvent(eventValues []interface{}, timestamp, blockNumber uint64) error {
	event := struct {
		VoucherId *big.Int
	}{}
	// Assign the unpacked value to the VoucherId field
	if len(eventValues) > 0 {
		event.VoucherId, _ = eventValues[0].(*big.Int)
	}
	bitMask := new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF)
	// Extract voucher and input using bit masking and shifting
	voucher := new(big.Int).Rsh(event.VoucherId, 128)
	input := new(big.Int).And(event.VoucherId, bitMask)

	// Print the extracted voucher and input
	fmt.Println("Voucher:", voucher)
	fmt.Println("Input:", input)
	// Print decoded event data
	fmt.Println("Voucher Executed - Voucher ID:", event.VoucherId.String())
	context := context.Background()

	fmt.Println("Context", context, timestamp)
	filterList := []*model.MetadataFilter{}
	strInputIndex := input.String()
	filterList = append(filterList, &model.MetadataFilter{
		Field: "InputIndex",
		Eq:    &strInputIndex,
	})
	strOutputIndex := voucher.String()
	filterList = append(filterList, &model.MetadataFilter{
		Field: "OutputIndex",
		Eq:    &strOutputIndex,
	})
	vouchers, err := x.Model.GetVouchersMetadata(filterList)
	if err != nil {
		slog.Error(err.Error())
	} else if len(vouchers) < 1 {
		slog.Warn("Voucher not found", "strInputIndex", strInputIndex, "strOutputIndex", strOutputIndex)
	} else {
		slog.Info("Voucher execution updated", "blockNumber", blockNumber, "timestamp", timestamp)
		vouchers[0].ExecutedBlock = blockNumber
		vouchers[0].ExecutedAt = timestamp
	}
	return nil
}

// String implements supervisor.Worker.
func (x ExecListener) String() string {
	return "ExecListener"
}

func (x ExecListener) Start(ctx context.Context, ready chan<- struct{}) error {
	slog.Info("Connecting to", "provider", x.Provider)
	client, err := ethclient.DialContext(ctx, x.Provider)
	if err != nil {
		return fmt.Errorf("execlistener: dial: %w", err)
	}
	ready <- struct{}{}
	return x.WatchExecutions(ctx, client)
}

func (x *ExecListener) WatchExecutions(ctx context.Context, client *ethclient.Client) error {

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
		case vLog := <-logs:
			fmt.Println(vLog)

			// Get the block number of the event
			blockNumber := vLog.BlockNumber
			blockNumberBigInt := big.NewInt(int64(blockNumber))
			// Fetch the block information
			block, err := client.BlockByNumber(context.Background(), blockNumberBigInt)
			if err != nil {
				log.Fatal(err)
			}

			// Extract the timestamp from the block
			timestamp := block.Time()

			values, err := contractABI.Unpack(x.EventName, vLog.Data)
			if err != nil {
				log.Fatal(err)
			} else {
				x.OnEvent(values, timestamp, blockNumber)
			}
		}
	}
}
