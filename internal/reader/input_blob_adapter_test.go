package reader

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
	"log"
	"log/slog"
	"math/big"
	"strings"
	"testing"
)

type InputBlobAdapterTestSuite struct {
	suite.Suite
	blobAdapter InputBlobAdapter
}

func (s *InputBlobAdapterTestSuite) SetupTest() {
	s.blobAdapter = InputBlobAdapter{}

}

func TestInputBlobAdapterSuite(t *testing.T) {
	suite.Run(t, new(InputBlobAdapterTestSuite))
}

func (s *InputBlobAdapterTestSuite) TestAdapt() {
	var nodeValue struct {
		Index  int    `json:"index"`
		Blob   string `json:"blob"`
		Status string `json:"status"`
	}

	nodeValue.Index = 1
	nodeValue.Blob = GenerateBlob()
	nodeValue.Status = "UNPROCESSED"

	input, err := s.blobAdapter.Adapt(nodeValue)
	slog.Info("input>>>>>", "ERR", input)
	s.NoError(err)
	s.NotNil(input)
}

func GenerateBlob() string {
	abiJSON := `
	[
		{
			"inputs": [
			{
				"internalType": "uint256",
				"name": "chainId",
				"type": "uint256"
			},
			{
				"internalType": "address",
				"name": "appContract",
				"type": "address"
			},
			{
				"internalType": "address",
				"name": "msgSender",
				"type": "address"
			},
			{
				"internalType": "uint256",
				"name": "blockNumber",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "blockTimestamp",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "prevRandao",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "index",
				"type": "uint256"
			},
			{
				"internalType": "bytes",
				"name": "payload",
				"type": "bytes"
			}
			],
			"name": "EvmAdvance",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		}
	]
	`

	// Parse the ABI JSON
	abiParsed, err := abi.JSON(strings.NewReader(abiJSON))

	if err != nil {
		log.Fatal(err)
	}

	chainId := big.NewInt(1)
	blockNumber := big.NewInt(20)
	blockTimestamp := big.NewInt(1234)
	index := big.NewInt(42)
	prevRandao := big.NewInt(21)
	payload := common.Hex2Bytes("11223344556677889900")
	appContract := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	msgSender := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	inputData, _ := abiParsed.Pack("EvmAdvance",
		&chainId,
		&appContract,
		&msgSender,
		&blockNumber,
		&blockTimestamp,
		&prevRandao,
		&index,
		payload,
	)

	return fmt.Sprintf("0x%s", common.Bytes2Hex(inputData))
}
