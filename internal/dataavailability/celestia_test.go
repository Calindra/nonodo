package dataavailability

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/stretchr/testify/suite"
)

type CelestiaSuite struct {
	suite.Suite
}

func (s *CelestiaSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func (s *CelestiaSuite) TestSubmitBlob() {
	// token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJwdWJsaWMiLCJyZWFkIiwid3JpdGUiLCJhZG1pbiJdfQ.jKg7hqk14Bh4QW_KoFHJ1Pb7h9buq2c42EX6q_QZ5gU"
	token := os.Getenv("CELESTIA_AUTH_TOKEN")
	// url := os.Getenv("CELESTIA_URL")
	// url := "https://api.celestia-arabica-11.com" //os.Getenv("CELESTIA_URL")
	// url := "https://validator-3.celestia-arabica-11.com:26657"
	// url := "https://26658-calindra-celestianode-p9zxr391sw1.ws-us114.gitpod.io"
	url := "https://26658-calindra-celestianode-p9zxr391sw1.ws-us114.gitpod.io"
	// url := "https://rpc.celestia-mocha.com:26658" // not working
	if token == "" || url == "" {
		slog.Debug("missing celestia configuration")
		return
	}
	ctx := context.Background()
	err := SubmitBlob(ctx, url, token)
	s.NoError(err)
	s.Fail("123")
}

func (s *CelestiaSuite) TestSubmitProof() {

}

func TestCelestiaSuite(t *testing.T) {
	suite.Run(t, &CelestiaSuite{})
}
