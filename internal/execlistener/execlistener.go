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
}

// String implements supervisor.Worker.
func (x ExecListener) String() string {
	return "ExecListener"
}

func (x ExecListener) Start(ctx context.Context, ready chan<- struct{}) error {
	slog.Info("Connecting to", "provider", x.Provider)
	client, err := ethclient.DialContext(ctx, x.Provider)
	if err != nil {
		return fmt.Errorf("inputter: dial: %w", err)
	}
	ready <- struct{}{}
	return x.WatchExecutions(client)
}

func (x *ExecListener) WatchExecutions(client *ethclient.Client) error {

	contractAddress := x.ApplicationAddress

	abiString := `[
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
	]`
	// ABI of your contract
	contractABI, err := abi.JSON(strings.NewReader(abiString))
	if err != nil {
		slog.Error(err.Error())
	}

	// Subscribe to event
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{contractABI.Events["VoucherExecuted"].ID}},
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
		case err := <-sub.Err():
			log.Fatal(err)
		case vLog := <-logs:
			fmt.Println(vLog)
			event := struct {
				VoucherId *big.Int
			}{}
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

			values, err := contractABI.Unpack("VoucherExecuted", vLog.Data)
			if err != nil {
				log.Fatal(err)
			}
			// Assign the unpacked value to the VoucherId field
			if len(values) > 0 {
				event.VoucherId, _ = values[0].(*big.Int)
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
		}
	}
}
