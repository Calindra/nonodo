// (c) Cartesi and individual authors (see AUTHORS)
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package task_reader

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	cRepos "github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
type TaskReaderWorker struct {
	Model                 Model
	Provider              string
	MockCoprocessor       common.Address
	MockCoprocessorBlock  uint64
	CoprocessorPrivateKey string
	Repository            cRepos.InputRepository
	EthClient             *ethclient.Client
}

func (w TaskReaderWorker) String() string {
	return "task_reader"
}

func (w TaskReaderWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	client, err := w.GetEthClient()
	if err != nil {
		return fmt.Errorf("task reader: dial: %w", err)
	}
	mockCoprocessor, err := contracts.NewMockCoprocessor(w.MockCoprocessor, client)
	if err != nil {
		return fmt.Errorf("task_reader: bind input box: %w", err)
	}
	ready <- struct{}{}
	return w.watchNewInputs(ctx, client, mockCoprocessor)
}

func (w *TaskReaderWorker) GetEthClient() (*ethclient.Client, error) {
	if w.EthClient == nil {
		ctx := context.Background()
		client, err := ethclient.DialContext(ctx, w.Provider)
		if err != nil {
			return nil, fmt.Errorf("task_reader: dial: %w", err)
		}
		w.EthClient = client
	}
	return w.EthClient, nil
}

func (w *TaskReaderWorker) ChainID() (*big.Int, error) {
	client, err := w.GetEthClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	return client.ChainID(ctx)
}

func (w *TaskReaderWorker) GetClientOpts() (*bind.TransactOpts, error) {
	chainId, err := w.ChainID()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain id: %v", err)
	}
	privateKey, err := crypto.HexToECDSA(w.CoprocessorPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %v", err)
	}
	opts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %v", err)
	}
	return opts, nil
}

// Watch new inputs added to the input box.
// This function continues to run forever until there is an error or the context is canceled.
func (w TaskReaderWorker) watchNewInputs(
	ctx context.Context,
	client *ethclient.Client,
	mockCoprocessor *contracts.MockCoprocessor,
) error {
	seconds := 5
	reconnectDelay := time.Duration(seconds) * time.Second

	for {
		// Create a new subscription
		logs := make(chan *contracts.MockCoprocessorTaskIssued)
		opts := bind.WatchOpts{
			Context: ctx,
		}
		sub, err := mockCoprocessor.WatchTaskIssued(&opts, logs)
		if err != nil {
			slog.Error("TaskReader", "error", err)
			slog.Info("TaskReader reconnecting", "reconnectDelay", reconnectDelay)
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
					if err := w.processInput(ctx, client, event); err != nil {
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
			slog.Error("TaskReader", "error", err)
			slog.Info("TaskReader reconnecting", "reconnectDelay", reconnectDelay)
			time.Sleep(reconnectDelay)
		} else {
			return nil
		}
	}
}

var inputRequestCounter int = 0

func (w TaskReaderWorker) processInput(
	ctx context.Context,
	client *ethclient.Client,
	event *contracts.MockCoprocessorTaskIssued,
) error {
	inputRequestCounter++
	header, err := client.HeaderByHash(ctx, event.Raw.BlockHash)
	if err != nil {
		return fmt.Errorf("task_reader: failed to get tx header: %w", err)
	}
	timestamp := time.Unix(int64(header.Time), 0)

	chainId, err := w.ChainID()
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	err = w.Model.AddAdvanceInput(
		event.Callback,
		common.Bytes2Hex(event.Input),
		event.Raw.BlockNumber,
		timestamp,
		inputRequestCounter, // since the coprocessor contracts don´t have an input index, we use the inputRequestCounter as the input index
		common.Bytes2Hex(event.MachineHash[:]), // this field is now used to store the machine hash instead of the prevRandao value
		event.Callback,
		chainId.String(),
	)

	if err != nil {
		return err
	}

	acceptedInput, err := w.waitForInput(ctx, inputRequestCounter)
	if err != nil {
		return err
	}

	err = w.executeOutput(ctx, client, acceptedInput)
	if err != nil {
		if strings.Contains(err.Error(), "execution reverted") {
			slog.Warn("Execution reverted, make sure the CoprocessorAdapter address is correct")
		}
		slog.Error("Failed to call coprocessor callback", "error", err)
		return err
	}

	slog.Debug("Input accepted", "input", acceptedInput)

	return nil
}

func (w TaskReaderWorker) waitForInput(ctx context.Context, index int) (*model.AdvanceInput, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		input, err := w.Repository.FindByIndexAndAppContract(ctx, index, &w.MockCoprocessor)
		if err != nil {
			return nil, fmt.Errorf("failed to get input status: %w", err)
		}
		if input.Status == model.CompletionStatusAccepted {
			return input, nil
		}
		if input.Status == model.CompletionStatusRejected {
			return nil, fmt.Errorf("input rejected")
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (w TaskReaderWorker) executeOutput(ctx context.Context, client *ethclient.Client, input *model.AdvanceInput) error {
	mockCoprocessor, err := contracts.NewMockCoprocessor(w.MockCoprocessor, client)
	if err != nil {
		return fmt.Errorf("task_reader: bind input box: %w", err)
	}

	opts, err := w.GetClientOpts()
	if err != nil {
		return fmt.Errorf("failed to get client opts: %w", err)
	}

	outputs := make([][]byte, len(input.Notices))
	for i, notice := range input.Notices {
		outputs[i] = common.Hex2Bytes(notice.Payload)
	}

	tx, err := mockCoprocessor.SolverCallbackOutputsOnly(
		opts,
		[32]byte(common.Hex2Bytes(input.PrevRandao)),
		crypto.Keccak256Hash(common.Hex2Bytes(input.Payload)),
		outputs,
		input.AppContract,
	)
	slog.Info("Outputs executed", "payload hash", crypto.Keccak256Hash(common.Hex2Bytes(input.Payload)), "tx", tx.Hash().Hex())
	return nil
}
