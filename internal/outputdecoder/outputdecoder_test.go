package outputdecoder

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

func TestModelSuite(t *testing.T) {
	suite.Run(t, new(OutputDecoderSuite))
}

func (s *OutputDecoderSuite) TestItAddsAndGetsAdvanceInputs() {
	// s.T().Skip()
	decoder := OutputDecoder{}
	address := common.HexToAddress("0x26A61aF89053c847B4bd5084E2caFe7211874a29")
	abi, err := decoder.GetAbi(address)
	if err != nil {
		s.Fail(err.Error())
		return
	}

	selectorHex := "a9059cbb"
	selectorBytes, err2 := hex.DecodeString(selectorHex)
	if err2 != nil {
		s.Fail(err2.Error())
		return
	}
	abiMethod, err3 := abi.MethodById(selectorBytes)
	if err3 != nil {
		s.Fail(err3.Error())
		return
	}
	s.Equal("transfer", abiMethod.RawName)
}

func (s *OutputDecoderSuite) TestDecode() {
	json := `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	abi, err := jsonToAbi(json)
	if err != nil {
		s.Fail(err.Error())
		return
	}
	selectorHex := "a9059cbb"
	selectorBytes, err2 := hex.DecodeString(selectorHex)
	if err2 != nil {
		s.Fail(err2.Error())
		return
	}
	abiMethod, err3 := abi.MethodById(selectorBytes)
	if err3 != nil {
		s.Fail(err3.Error())
		return
	}
	s.Equal("transfer", abiMethod.RawName)
}
