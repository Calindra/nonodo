// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package devnet

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func DefaultTxOpts(ctx context.Context, client *ethclient.Client) (*bind.TransactOpts, error) {
	privateKey, err := crypto.ToECDSA(common.Hex2Bytes(SenderPrivateKey[2:]))
	if err != nil {
		return nil, fmt.Errorf("create private key: %w", err)
	}

	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain id: %w", err)
	}

	txOpts, err := bind.NewKeyedTransactorWithChainID(privateKey, chainId)
	if err != nil {
		return nil, fmt.Errorf("create transactor: %w", err)
	}
	nonce, err := client.PendingNonceAt(ctx, common.HexToAddress(SenderAddress))
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.Value = big.NewInt(0)
	txOpts.GasLimit = GasLimit
	txOpts.GasPrice, err = client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gas price: %w", err)
	}
	return txOpts, nil
}

// AddInput sends an input to Ethereum using the devnet sender.
// This function should be used in the devnet environment.
func AddInput(ctx context.Context, rpcUrl string, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("cannot send empty payload")
	}

	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return fmt.Errorf("dial to %v: %w", rpcUrl, err)
	}

	txOpts, err := DefaultTxOpts(ctx, client)
	if err != nil {
		return fmt.Errorf("create transactor: %w", err)
	}

	inputBox, err := contracts.NewInputBox(common.HexToAddress(InputBoxAddress), client)
	if err != nil {
		return fmt.Errorf("bind input box: %w", err)
	}
	slog.Debug("AddInput", "dapp", ApplicationAddress)
	tx, err := inputBox.AddInput(txOpts, common.HexToAddress(ApplicationAddress), payload)
	if err != nil {
		return fmt.Errorf("add input: %w", err)
	}

	receipt, err := waitMined(ctx, client, tx)
	if err != nil {
		return err
	}
	if receipt.Status == 0 {
		return fmt.Errorf("transaction was not accepted")
	}
	return nil
}

// GetInputAdded gets all input added events from the input box.
func GetInputAdded(ctx context.Context, rpcUrl string) ([]*contracts.InputBoxInputAdded, error) {
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("dial to %v: %w", rpcUrl, err)
	}
	inputBox, err := contracts.NewInputBox(common.HexToAddress(InputBoxAddress), client)
	if err != nil {
		return nil, fmt.Errorf("bind input box: %w", err)
	}
	it, err := inputBox.FilterInputAdded(nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("filter input added: %v", err)
	}
	defer it.Close()
	var inputs []*contracts.InputBoxInputAdded
	for it.Next() {
		inputs = append(inputs, it.Event)
	}
	return inputs, nil
}

// WaitMined waits for tx to be mined on the blockchain.
// It stops waiting when the context is canceled.
func waitMined(
	ctx context.Context,
	client *ethclient.Client,
	tx *types.Transaction,
) (*types.Receipt, error) {
	const pollFrequency = 33 * time.Millisecond
	queryTicker := time.NewTicker(pollFrequency)
	defer queryTicker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			return receipt, nil
		}
		if errors.Is(err, ethereum.NotFound) {
			slog.Debug("Transaction not yet mined")
		} else {
			return nil, fmt.Errorf("receipt retrieval failed: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-queryTicker.C:
		}
	}
}
