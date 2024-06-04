// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package inputter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Model interface {
	AddAdvanceInput(
		sender common.Address,
		payload []byte,
		blockNumber uint64,
		timestamp time.Time,
		index int,
	)
}

// This worker reads inputs from Ethereum and puts them in the model.
type InputterWorker struct {
	Model              Model
	Provider           string
	InputBoxAddress    common.Address
	InputBoxBlock      uint64
	ApplicationAddress common.Address
	Repository         model.InputRepository
}

func (w InputterWorker) String() string {
	return "inputter"
}

func (w InputterWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	client, err := ethclient.DialContext(ctx, w.Provider)
	if err != nil {
		return fmt.Errorf("inputter: dial: %w", err)
	}
	inputBox, err := contracts.NewInputBox(w.InputBoxAddress, client)
	if err != nil {
		return fmt.Errorf("inputter: bind input box: %w", err)
	}
	ready <- struct{}{}

	// First, read the event logs to get the past inputs; then, watch the event logs to get the
	// new ones. There is a race condition where we might lose inputs sent between the
	// readPastInputs call and the watchNewInputs call. Given that nonodo is a development node,
	// we accept this race condition.
	err = w.readPastInputs(ctx, client, inputBox)
	if err != nil {
		return err
	}
	return w.watchNewInputs(ctx, client, inputBox)
}

// Read inputs starting from the input box deployment block until the latest block.
func (w InputterWorker) readPastInputs(
	ctx context.Context,
	client *ethclient.Client,
	inputBox *contracts.InputBox,
) error {
	opts := bind.FilterOpts{
		Context: ctx,
		Start:   w.InputBoxBlock,
	}
	filter := []common.Address{w.ApplicationAddress}
	it, err := inputBox.FilterInputAdded(&opts, filter, nil)
	if err != nil {
		return fmt.Errorf("inputter: filter input added: %v", err)
	}
	defer it.Close()
	for it.Next() {
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
	logs := make(chan *contracts.InputBoxInputAdded)
	opts := bind.WatchOpts{
		Context: ctx,
	}
	filter := []common.Address{w.ApplicationAddress}
	sub, err := inputBox.WatchInputAdded(&opts, logs, filter, nil)
	if err != nil {
		return fmt.Errorf("inputter: watch input added: %w", err)
	}
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			return err
		case event := <-logs:
			if err := w.addInput(ctx, client, event); err != nil {
				return err
			}
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

	msgSender := values[2].(common.Address)
	payload := values[7].([]uint8)

	slog.Debug("inputter: read event",
		"dapp", event.AppContract,
		"input.index", event.Index,
		"sender", msgSender,
		"input", event.Input,
		"payload", payload,
		slog.Group("block",
			"number", header.Number,
			"timestamp", timestamp,
		),
	)

	w.Model.AddAdvanceInput(
		msgSender,
		payload,
		event.Raw.BlockNumber,
		timestamp,
		int(event.InputIndex.Int64()),
	)
	return nil
}
