package avail

import (
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/devnet"
	"github.com/stretchr/testify/suite"
)

type AvailClientSuite struct {
	suite.Suite
}

func (s *AvailClientSuite) XTestSendTransaction() {
	availClient, err := NewAvailClient("", DEFAULT_CHAINID_HARDHAT, DEFAULT_APP_ID)
	if err != nil {
		s.NoError(err)
		return
	}
	data := "deadbeef"
	ApiURL := "wss://turing-testnet.avail-rpc.com"
	Seed := os.Getenv("AVAIL_MNEMONIC")
	AppID := 91
	if Seed != "" {
		_, err := availClient.SubmitData(data, ApiURL, Seed, AppID)
		s.NoError(err)
	}
}

func (s *AvailClientSuite) TestSubmit712() {
	availClient, err := NewAvailClient("", DEFAULT_CHAINID_HARDHAT, DEFAULT_APP_ID)
	if err != nil {
		s.NoError(err)
		return
	}
	Seed := os.Getenv("AVAIL_MNEMONIC")
	if Seed != "" {
		_, err := availClient.Submit712("Cartesi Rocks!", devnet.ApplicationAddress, uint64(10))
		s.NoError(err)
		s.Fail("XXX")
	}
}

func TestEspressoListenerSuite(t *testing.T) {
	suite.Run(t, &AvailClientSuite{})
}
