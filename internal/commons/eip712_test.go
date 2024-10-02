package commons

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

var Token = common.HexToAddress("0xc6e7DF5E7b4f2A278906862b61205850344D4e7d")

type EIP712Suite struct {
	suite.Suite
}

func (s *EIP712Suite) SetupTest() {

}

func TestDecoderSuite(t *testing.T) {
	suite.Run(t, new(EIP712Suite))
}

func (s *EIP712Suite) TestHandleOutput() {
	Main()
	// s.Fail("xxx")
}
