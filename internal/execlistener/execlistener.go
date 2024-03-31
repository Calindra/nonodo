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

func NewExecListener(
	model *model.NonodoModel,
	provider string,
	applicationAddress common.Address,
) ExecListener {
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

func toStringPtr(num *big.Int) *string {
	s := num.String()
	return &s
}

// on event callback
func (x ExecListener) OnEvent(eventValues []interface{}, timestamp, blockNumber uint64) error {
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

	filterList := []*model.MetadataFilter{}
	filterList = append(filterList, &model.MetadataFilter{
		Field: "InputIndex",
		Eq:    toStringPtr(input),
	})
	filterList = append(filterList, &model.MetadataFilter{
		Field: "OutputIndex",
		Eq:    toStringPtr(voucher),
	})
	vouchers, err := x.Model.GetVouchersMetadata(filterList)
	if err != nil {
		slog.Error(err.Error())
	} else if len(vouchers) < 1 {
		slog.Warn("Voucher not found",
			"InputIndex", input.String(),
			"OutputIndex", voucher.String(),
		)
	} else {
		slog.Debug("Voucher Metadata updated",
			"blockNumber", blockNumber,
			"timestamp", timestamp,
		)
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
