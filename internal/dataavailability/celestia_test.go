package dataavailability

import (
	"context"
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type CelestiaSuite struct {
	suite.Suite
}

func (s *CelestiaSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func (s *CelestiaSuite) createGioID(blockHeight uint64, shareStart uint64, shareEnd uint64) string {
	abiParsed, err := getABI()
	s.NoError(err)
	// stringNamespace := "00000000000000000000000000000000000000000000000000000000deadbeef"
	stringNamespace := "00000000000000000000000000000000000000000000000000deadbeef"
	namespace := common.Hex2Bytes(stringNamespace)
	var bytesNamespace [29]byte
	copy(bytesNamespace[:], namespace)
	height := big.NewInt(int64(blockHeight))
	start := big.NewInt(int64(shareStart))
	end := big.NewInt(int64(shareEnd))
	payload, err := abiParsed.Pack(
		"CelestiaRequest",
		bytesNamespace,
		&height,
		&start,
		&end,
	)
	s.NoError(err)
	gioID := "0x" + common.Bytes2Hex(payload[4:])
	return gioID
}

func (s *CelestiaSuite) XTestTendermint(t *testing.T) {
	ctx := context.Background()
	gioID := s.createGioID(2034386, 10, 11)
	slog.Debug("FetchFromTendermint", "gioID", gioID)
	dataAsHexStr, err := FetchFromTendermint(ctx, gioID)
	s.NoError(err)
	dataAsBytes := common.Hex2Bytes(*dataAsHexStr)
	s.Equal("Hello, World! Cartesi Rocks!", string(dataAsBytes))
}

func (s *CelestiaSuite) XTestSubmitBlob() {
	token := os.Getenv("TIA_AUTH_TOKEN")
	url := os.Getenv("TIA_URL")
	// url := "https://api.celestia-arabica-11.com" //os.Getenv("CELESTIA_URL")
	// url := "https://validator-3.celestia-arabica-11.com:26657"
	// url := "https://26658-calindra-celestianode-p9zxr391sw1.ws-us114.gitpod.io"
	// url := "https://26658-calindra-celestianode-p9zxr391sw1.ws-us114.gitpod.io"
	// url := "https://rpc.celestia-mocha.com:26658" // not working
	if token == "" || url == "" {
		slog.Debug("missing celestia configuration")
		return
	}
	slog.Debug("Configs", "url", url, "token", token)
	ctx := context.Background()
	strData := `Hello, World! Cartesi Rocks!
	Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello`
	rawData := []byte(strData)
	blockHeight, shareStart, shareEnd, err := SubmitBlob(ctx, url, token, "DEADBEEF", rawData)
	s.NoError(err)

	// test the fetch
	gioID := s.createGioID(blockHeight, shareStart, shareEnd)
	slog.Debug("ID",
		"id", gioID,
		"namespace", namespace,
	)
	data, err := GetBlob(ctx, gioID, url, token)
	s.NoError(err)
	slog.Debug("GetBlob",
		"data", string(data),
	)
	s.Fail("123")
}

func (s *CelestiaSuite) XTestGioRequest() {
	gioID := s.createGioID(1490181, 1, 2)

	ctx := context.Background()
	token := os.Getenv("TIA_AUTH_TOKEN")
	url := os.Getenv("TIA_URL")
	// url := "https://api.celestia-arabica-11.com" //os.Getenv("CELESTIA_URL")
	// url := "https://validator-3.celestia-arabica-11.com:26657"
	// url := "https://26658-calindra-celestianode-p9zxr391sw1.ws-us114.gitpod.io"
	// url := "https://26658-calindra-celestianode-p9zxr391sw1.ws-us114.gitpod.io"
	// url := "https://rpc.celestia-mocha.com:26658" // not working
	if token == "" || url == "" {
		slog.Debug("missing celestia configuration")
		return
	}
	data, err := GetBlob(ctx, gioID, url, token)
	s.NoError(err)
	slog.Debug("GetBlob", "data", string(data))
	// s.Fail("x")
}

func TestCelestiaSuite(t *testing.T) {
	suite.Run(t, &CelestiaSuite{})
}
