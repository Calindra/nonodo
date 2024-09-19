package avail

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type AvailClientSuite struct {
	suite.Suite
}

func (s *AvailClientSuite) TestSendTransaction() {
	data := "deadbeef"
	ApiURL := "wss://turing-testnet.avail-rpc.com"
	Seed := os.Getenv("AVAIL_MNEMONIC")
	AppID := 91
	if Seed != "" {
		err := SubmitData(data, ApiURL, Seed, AppID)
		s.NoError(err)
	}
}

func TestEspressoListenerSuite(t *testing.T) {
	suite.Run(t, &AvailClientSuite{})
}
