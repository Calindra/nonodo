package synchronizer

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type GraphileFetcherTestSuite struct {
	suite.Suite
	graphileFetcher GraphileFetcher
	graphileClient  *MockHttpClient
}

type MockHttpClient struct {
	PostFunc func(body []byte) ([]byte, error)
}

func (m *MockHttpClient) Post(body []byte) ([]byte, error) {
	// If PostFUnc is defined, call it
	if m.PostFunc != nil {
		return m.PostFunc(body)
	}
	// Otherwise return error
	return nil, errors.New("PostFunc not set in the mock")
}

func (s *GraphileFetcherTestSuite) SetupTest() {
	s.graphileClient = &MockHttpClient{}
	s.graphileFetcher = GraphileFetcher{GraphileClient: s.graphileClient}
}

func TestAdapterV2Suite(t *testing.T) {
	suite.Run(t, new(GraphileFetcherTestSuite))
}

func (s *GraphileFetcherTestSuite) TestFetchWithoutCursor() {
	blob := GenerateOutputBlob()
	s.graphileClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(fmt.Sprintf(`{
 "data": {
   "outputs": {
     "edges": [
       {
         "cursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsxXV0=",
         "node": {
           "index": 1,
           "blob": "%s",
           "inputIndex": 1
         }
       }
     ],
	 "pageInfo": {
		"endCursor": "",
		"hasNextPage": false,
		"hasPreviousPage": false,
		"startCursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsxLDFdXQ=="
	  }
   }
 }
}`, blob)), nil
	}

	s.graphileFetcher.CursorAfter = "WyJwcmltYXJ5X2tleV9hc2MiLFsyLDJdXQ"

	resp, err := s.graphileFetcher.Fetch()

	s.NoError(err)
	s.NotNil(resp)
}

func (s *GraphileFetcherTestSuite) TestFetchWithCursor() {
	blob := GenerateOutputBlob()
	s.graphileClient.PostFunc = func(body []byte) ([]byte, error) {
		return []byte(fmt.Sprintf(`{
 "data": {
   "outputs": {
     "edges": [
       {
         "cursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsxXV0=",
         "node": {
           "index": 1,
           "blob": "%s",
           "inputIndex": 1
         }
       }
     ],
	 "pageInfo": {
		"endCursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsyLDJdXQ==",
		"hasNextPage": false,
		"hasPreviousPage": false,
		"startCursor": "WyJwcmltYXJ5X2tleV9hc2MiLFsxLDFdXQ=="
	  }
   }
 }
}`, blob)), nil
	}

	s.graphileFetcher.CursorAfter = ""

	resp, err := s.graphileFetcher.Fetch()

	s.NoError(err)
	s.NotNil(resp)
}

func (s *GraphileFetcherTestSuite) TestPrintInputBlob() {
	slog.Info("Blob", "Input Blob", GenerateInputBlob())
}

func (s *GraphileFetcherTestSuite) TestPrintOutputBlob() {
	slog.Info("Blob", "Output Blob", GenerateOutputBlob())
}

func GenerateOutputBlob() string {
	// Parse the ABI JSON
	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		log.Fatal(err)
	}

	value := big.NewInt(1000000000000000000)
	payload := common.Hex2Bytes("11223344556677889900")
	destination := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	inputData, _ := abiParsed.Pack("Vouchers",
		destination,
		value,
		payload,
	)

	return fmt.Sprintf("0x%s", common.Bytes2Hex(inputData))
}

func GenerateInputBlob() string {
	// Parse the ABI JSON
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		log.Fatal(err)
	}

	chainId := big.NewInt(1000000000000000000)
	appContract := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	blockNumber := big.NewInt(1000000000000000000)
	blockTimestamp := big.NewInt(1720701841)
	payload := common.Hex2Bytes("11223344556677889900")
	prevRandao := big.NewInt(1000000000000000000)
	index := big.NewInt(1000000000000000000)
	msgSender := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	inputData, _ := abiParsed.Pack("EvmAdvance",
		chainId,
		appContract,
		msgSender,
		blockNumber,
		blockTimestamp,
		prevRandao,
		index,
		payload,
	)

	return fmt.Sprintf("0x%s", common.Bytes2Hex(inputData))
}
