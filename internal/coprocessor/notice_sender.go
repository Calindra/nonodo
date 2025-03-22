package coprocessor

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type CoProcessor struct {
	ethClient                 *ethclient.Client
	privateKey                *ecdsa.PrivateKey
	coprocessorAdapterAddress common.Address
	senderAddress             common.Address
}

func NewCoProcessorFromEnvs(ctx context.Context) (*CoProcessor, error) {
	nodeURL := os.Getenv("RPC_URL")
	if nodeURL == "" {
		return nil, errors.New("RPC_URL environment variable is not set")
	}

	client, err := ethclient.DialContext(ctx, nodeURL)
	if err != nil {
		return nil, err
	}

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		return nil, errors.New("PRIVATE_KEY environment variable is not set")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}

	contractAddress := os.Getenv("COPROCESSOR_ADAPTER_CONTRACT_ADDRESS")
	if contractAddress == "" {
		return nil, errors.New("COPROCESSOR_ADAPTER_CONTRACT_ADDRESS environment variable is not set")
	}

	noticeReceiver := common.HexToAddress(contractAddress)
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	return &CoProcessor{
		ethClient:                 client,
		privateKey:                privateKey,
		coprocessorAdapterAddress: noticeReceiver,
		senderAddress:             fromAddress,
	}, nil
}

func (r *CoProcessor) SendNoticeCallback(ctx context.Context, notice []byte) {
	contractABI := contracts.CoprocessorAdapterMetaData.ABI

	// Load the contract ABI
	parsedABI, err := abi.JSON(bytes.NewReader([]byte(contractABI)))
	if err != nil {
		log.Fatalf("Failed to parse contract ABI: %v", err)
	}

	// Prepare data for the function
	payloadHash := common.HexToHash("0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")

	// Pack the data to call the Solidity function
	data, err := parsedABI.Pack("nonodoHandleNotice", payloadHash, notice)
	if err != nil {
		log.Fatalf("Failed to pack data: %v", err)
	}

	// Get the nonce for the transaction
	nonce, err := r.ethClient.PendingNonceAt(ctx, r.senderAddress)
	if err != nil {
		log.Fatalf("Failed to get nonce: %v", err)
	}

	// Set the gas price
	gasPrice, err := r.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("Failed to get gas price: %v", err)
	}

	// Estimate the gas needed for the transaction
	gasLimit, err := r.ethClient.EstimateGas(ctx, ethereum.CallMsg{
		To:   &r.coprocessorAdapterAddress,
		Data: data,
	})
	if err != nil {
		log.Fatalf("Failed to estimate gas: %v", err)
	}

	// Prepare the transaction
	tx := types.NewTransaction(nonce, r.coprocessorAdapterAddress, big.NewInt(0), gasLimit, gasPrice, data)
	chainID, err := r.ethClient.NetworkID(ctx)
	if err != nil {
		log.Fatalf("Failed to get network ID: %v", err)
	}
	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), r.privateKey)
	if err != nil {
		log.Fatalf("Failed to sign the transaction: %v", err)
	}

	// Send the transaction
	err = r.ethClient.SendTransaction(ctx, signedTx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}

	fmt.Println("Transaction sent! Transaction Hash:", signedTx.Hash().Hex())
	receipt, err := bind.WaitMined(ctx, r.ethClient, signedTx)
	if err != nil {
		log.Fatalf("Failed to send transaction: %v", err)
	}

	if receipt.Status == 0 {
		log.Fatalf("Failed to send transaction")
	}
}
