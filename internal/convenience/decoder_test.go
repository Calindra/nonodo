package convenience

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type OutputDecoderSuite struct {
	suite.Suite
}

func (s *OutputDecoderSuite) SetupTest() {
}

func TestDecoderSuite(t *testing.T) {
	suite.Run(t, new(OutputDecoderSuite))
}

func (s *OutputDecoderSuite) TestGetAbiFromEtherscan() {
	s.T().Skip()
	decoder := OutputDecoder{}
	address := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	abi, err := decoder.GetAbi(address)
	checkError(s, err)
	selectorBytes, err2 := hex.DecodeString("a9059cbb")
	checkError(s, err2)
	abiMethod, err3 := abi.MethodById(selectorBytes)
	checkError(s, err3)
	s.Equal("transfer", abiMethod.RawName)
}

func (s *OutputDecoderSuite) TestDecode() {
	json := `[{
		"constant": false,
		"inputs": [
			{
				"name": "_to",
				"type": "address"
			},
			{
				"name": "_value",
				"type": "uint256"
			}
		],
		"name": "transfer",
		"outputs": [
			{
				"name": "",
				"type": "bool"
			}
		],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	}]`
	abi, err := jsonToAbi(json)
	checkError(s, err)
	selectorBytes, err2 := hex.DecodeString("a9059cbb")
	checkError(s, err2)
	abiMethod, err3 := abi.MethodById(selectorBytes)
	checkError(s, err3)
	s.Equal("transfer", abiMethod.RawName)
}

func checkError(s *OutputDecoderSuite, err error) {
	if err != nil {
		s.T().Fatal(err.Error())
	}
}
