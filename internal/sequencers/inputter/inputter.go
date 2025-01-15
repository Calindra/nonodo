// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package inputter

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	cRepos "github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Model interface {
	AddAdvanceInput(
		sender common.Address,
		payload string,
		blockNumber uint64,
		timestamp time.Time,
		index int,
		prevRandao string,
		appContract common.Address,
		chainId string,
	) error
}

// This worker reads inputs from Ethereum and puts them in the model.
type InputterWorker struct {
	Model              Model
	Provider           string
	InputBoxAddress    common.Address
	InputBoxBlock      uint64
	ApplicationAddress common.Address
	Repository         cRepos.InputRepository
	EthClient          *ethclient.Client
}

func (w InputterWorker) String() string {
	return "inputter"
}

func (w InputterWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	client, err := w.GetEthClient()
	if err != nil {
		return fmt.Errorf("inputter: dial: %w", err)
	}
	inputBox, err := contracts.NewInputBox(w.InputBoxAddress, client)
	if err != nil {
		return fmt.Errorf("inputter: bind input box: %w", err)
	}
	ready <- struct{}{}
	return w.watchNewInputs(ctx, client, inputBox)
}

func (w *InputterWorker) GetEthClient() (*ethclient.Client, error) {
	if w.EthClient == nil {
		ctx := context.Background()
		client, err := ethclient.DialContext(ctx, w.Provider)
		if err != nil {
			return nil, fmt.Errorf("inputter: dial: %w", err)
		}
		w.EthClient = client
	}
	return w.EthClient, nil
}

func (w *InputterWorker) ChainID() (*big.Int, error) {
	client, err := w.GetEthClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	return client.ChainID(ctx)
}

// Read inputs starting from the input box deployment block until the latest block.
func (w *InputterWorker) ReadPastInputs(
	ctx context.Context,
	client *ethclient.Client,
	inputBox *contracts.InputBox,
	startBlockNumber uint64,
	endBlockNumber *uint64,
) error {
	if endBlockNumber != nil {
		slog.Debug("readPastInputs",
			"startBlockNumber", startBlockNumber,
			"endBlockNumber", *endBlockNumber,
			"dappAddress", w.ApplicationAddress,
		)
	} else {
		slog.Debug("readPastInputs",
			"startBlockNumber", startBlockNumber,
			"dappAddress", w.ApplicationAddress,
		)
	}
	opts := bind.FilterOpts{
		Context: ctx,
		Start:   startBlockNumber,
		End:     endBlockNumber,
	}
	filter := []common.Address{w.ApplicationAddress}
	it, err := inputBox.FilterInputAdded(&opts, filter, nil)
	if err != nil {
		return fmt.Errorf("inputter: filter input added: %v", err)
	}
	defer it.Close()
	for it.Next() {
		w.InputBoxBlock = it.Event.Raw.BlockNumber - 1
		if err := w.addInput(ctx, client, it.Event); err != nil {
			return err
		}
	}
	return nil
}

// Watch new inputs added to the input box.
// This function continues to run forever until there is an error or the context is canceled.
func (w InputterWorker) watchNewInputs(
	ctx context.Context,
	client *ethclient.Client,
	inputBox *contracts.InputBox,
) error {
	seconds := 5
	reconnectDelay := time.Duration(seconds) * time.Second
	currentBlock := w.InputBoxBlock

	for {
		// First, read the event logs to get the past inputs; then, watch the event logs to get the
		// new ones. There is a race condition where we might lose inputs sent between the
		// readPastInputs call and the watchNewInputs call. Given that nonodo is a development node,
		// we accept this race condition.
		err := w.ReadPastInputs(ctx, client, inputBox, currentBlock, nil)
		if err != nil {
			slog.Error("Inputter", "error", err)
			slog.Info("Inputter reconnecting", "reconnectDelay", reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}

		// Create a new subscription
		logs := make(chan *contracts.InputBoxInputAdded)
		opts := bind.WatchOpts{
			Context: ctx,
		}
		filter := []common.Address{}
		sub, err := inputBox.WatchInputAdded(&opts, logs, filter, nil)
		if err != nil {
			slog.Error("Inputter", "error", err)
			slog.Info("Inputter reconnecting", "reconnectDelay", reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}

		// Handle the subscription in a separate goroutine
		errCh := make(chan error, 1)
		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case err := <-sub.Err():
					errCh <- err
					return
				case event := <-logs:
					currentBlock = event.Raw.BlockNumber - 1
					if err := w.addInput(ctx, client, event); err != nil {
						errCh <- err
						return
					}
				}
			}
		}()

		err = <-errCh
		sub.Unsubscribe()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			slog.Error("Inputter", "error", err)
			slog.Info("Inputter reconnecting", "reconnectDelay", reconnectDelay)
			time.Sleep(reconnectDelay)
		} else {
			return nil
		}
	}
}

// Add the input to the model.
func (w InputterWorker) addInput(
	ctx context.Context,
	client *ethclient.Client,
	event *contracts.InputBoxInputAdded,
) error {
	header, err := client.HeaderByHash(ctx, event.Raw.BlockHash)
	if err != nil {
		return fmt.Errorf("inputter: failed to get tx header: %w", err)
	}
	timestamp := time.Unix(int64(header.Time), 0)

	// use abi to decode the input
	eventInput := event.Input[4:]
	abi, err := contracts.InputsMetaData.GetAbi()
	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return err
	}

	values, err := abi.Methods["EvmAdvance"].Inputs.UnpackValues(eventInput)
	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return err
	}

	chainId := values[0].(*big.Int).String()
	msgSender := values[2].(common.Address)
	prevRandao := fmt.Sprintf("0x%s", common.Bytes2Hex(values[5].(*big.Int).Bytes()))
	payload := common.Bytes2Hex(values[7].([]uint8))
	inputIndex := int(event.Index.Int64())

	slog.Debug("inputter: read event",
		"dapp", event.AppContract,
		"input.index", event.Index,
		"sender", msgSender,
		"input", common.Bytes2Hex(event.Input),
		"payload", payload,
		slog.Group("block",
			"number", header.Number,
			"timestamp", timestamp,
			"prevRandao", prevRandao,
		),
	)

	if w.ApplicationAddress != event.AppContract {
		msg := fmt.Sprintf("The dapp address is wrong: %s. It should be %s",
			event.AppContract.Hex(),
			w.ApplicationAddress,
		)
		slog.Warn(msg)
		return nil
	}

	err = w.Model.AddAdvanceInput(
		msgSender,
		payload,
		event.Raw.BlockNumber,
		timestamp,
		inputIndex,
		prevRandao,
		event.AppContract,
		chainId,
	)

	if err != nil {
		return err
	}

	return nil
}

func (w InputterWorker) ReadInputsByBlockAndTimestamp(
	ctx context.Context,
	client *ethclient.Client,
	inputBox *contracts.InputBox,
	startBlockNumber uint64,
	endTimestamp uint64,
) (uint64, error) {
	slog.Debug("ReadInputsByBlockAndTimestamp",
		"startBlockNumber", startBlockNumber,
		"dappAddress", w.ApplicationAddress,
		"endTimestamp", endTimestamp,
	)
	lastL1BlockRead := startBlockNumber

	opts := bind.FilterOpts{
		Context: ctx,
		Start:   startBlockNumber,
	}
	filter := []common.Address{w.ApplicationAddress}
	it, err := inputBox.FilterInputAdded(&opts, filter, nil)

	if err != nil {
		return 0, fmt.Errorf("inputter: filter input added: %v", err)
	}
	defer it.Close()

	for it.Next() {
		header, err := client.HeaderByHash(ctx, it.Event.Raw.BlockHash)

		if err != nil {
			return 0, fmt.Errorf("inputter: failed to get tx header: %w", err)
		}
		timestamp := uint64(header.Time)

		if timestamp < endTimestamp {
			w.InputBoxBlock = it.Event.Raw.BlockNumber - 1
			if err := w.addInput(ctx, client, it.Event); err != nil {
				return 0, err
			}
			lastL1BlockRead = it.Event.Raw.BlockNumber
		} else {
			slog.Debug("InputAdded ignored", "timestamp", timestamp, "endTimestamp", endTimestamp)
		}
	}

	return lastL1BlockRead, nil
}

func (w InputterWorker) FindAllInputsByBlockAndTimestampLT(
	ctx context.Context,
	client *ethclient.Client,
	inputBox *contracts.InputBox,
	startBlockNumber uint64,
	endTimestamp uint64,
) ([]cModel.AdvanceInput, error) {
	slog.Debug("ReadInputsByBlockAndTimestamp",
		"startBlockNumber", startBlockNumber,
		"dappAddress", w.ApplicationAddress,
		"endTimestamp", endTimestamp,
	)

	opts := bind.FilterOpts{
		Context: ctx,
		Start:   startBlockNumber,
	}
	filter := []common.Address{w.ApplicationAddress}
	it, err := inputBox.FilterInputAdded(&opts, filter, nil)
	result := []cModel.AdvanceInput{}
	if err != nil {
		return result, fmt.Errorf("inputter: filter input added: %v", err)
	}
	defer it.Close()

	for it.Next() {
		header, err := client.HeaderByHash(ctx, it.Event.Raw.BlockHash)

		if err != nil {
			return result, fmt.Errorf("inputter: failed to get tx header: %w", err)
		}
		timestamp := uint64(header.Time)
		unixTimestamp := time.Unix(int64(header.Time), 0)
		if timestamp < endTimestamp {
			eventInput := it.Event.Input[4:]
			abi, err := contracts.InputsMetaData.GetAbi()
			if err != nil {
				slog.Error("Error parsing abi", "err", err)
				return result, err
			}

			values, err := abi.Methods["EvmAdvance"].Inputs.UnpackValues(eventInput)
			if err != nil {
				slog.Error("Error parsing abi", "err", err)
				return result, err
			}

			chainId := values[0].(*big.Int).String()
			appContract := values[1].(common.Address)
			msgSender := values[2].(common.Address)
			prevRandao := fmt.Sprintf("0x%s", common.Bytes2Hex(values[5].(*big.Int).Bytes()))
			payload := common.Bytes2Hex(values[7].([]uint8))
			inputIndex := int(it.Event.Index.Int64())

			input := cModel.AdvanceInput{
				ID:                     strconv.Itoa(inputIndex),
				Index:                  -1,
				Status:                 cModel.CompletionStatusUnprocessed,
				MsgSender:              msgSender,
				Payload:                payload,
				BlockTimestamp:         unixTimestamp,
				BlockNumber:            header.Number.Uint64(),
				EspressoBlockNumber:    -1,
				EspressoBlockTimestamp: time.Unix(-1, 0),
				InputBoxIndex:          inputIndex,
				PrevRandao:             prevRandao,
				AppContract:            appContract,
				ChainId:                chainId,
			}
			slog.Debug("append InputAdded", "timestamp", timestamp, "endTimestamp", endTimestamp)
			result = append(result, input)
		} else {
			slog.Debug("skip event InputAdded",
				"timestamp", timestamp,
				"endTimestamp", endTimestamp,
				"timeDiff", timestamp-endTimestamp,
			)
		}
	}

	return result, nil
}
