package dataavailability

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/tendermint/rpc/client/http"
)

type CelestiaSuite struct {
	suite.Suite
}

func (s *CelestiaSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func (s *CelestiaSuite) TestTendermint() {
	trpcEndpoint := "https://celestia-mocha-rpc.publicnode.com:443"
	trpc, err := http.New(trpcEndpoint, "/websocket")
	if err != nil {
		panic(fmt.Errorf("failed to connect to the Tendermint RPC: %w", err))
	}
	ctx := context.Background()
	blockHeight := 2034386
	shareStart := 10
	shareEnd := 11
	// blockHeight := 2048473
	// shareStart := 16
	// shareEnd := 17

	shareProof, err := trpc.ProveShares(ctx, uint64(blockHeight), uint64(shareStart), uint64(shareEnd))
	if err != nil {
		s.Fail(fmt.Sprintf("failed to get share proof: %s", err))
	}
	namespace := shareProof.Data[0][:29]
	dataRawLen := shareProof.Data[0][30:34]
	dataLen := binary.BigEndian.Uint32(dataRawLen)
	data := shareProof.Data[0][34 : 34+dataLen]
	slog.Debug("Result",
		"namespace", common.Bytes2Hex(namespace),
		"dataLen", binary.BigEndian.Uint32(dataRawLen),
		"data", common.Bytes2Hex(data),
		"data", string(data),
	)
	s.Fail("x")
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
	rawData := []byte(`Hello, World! Cartesi Rocks!
	Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!Hello, World! Cartesi Rocks!`)
	heightU, startU, err := SubmitBlob(ctx, url, token, rawData)
	s.NoError(err)

	// test the fetch
	abiParsed, err := getABI()
	s.NoError(err)

	// dead beef
	stringNamespace := "00000000000000000000000000000000000000000000000000000000deadbeef"
	namespace := common.Hex2Bytes(stringNamespace)
	var bytes32Value [32]byte
	copy(bytes32Value[:], namespace)
	height := big.NewInt(int64(heightU))
	start := big.NewInt(int64(startU))
	payload, err := abiParsed.Pack(
		"CelestiaRequest",
		bytes32Value,
		&height,
		&start,
	)
	s.NoError(err)
	gioID := "0x" + common.Bytes2Hex(payload)
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
	abiParsed, err := getABI()
	s.NoError(err)
	// dead beef
	stringNamespace := "00000000000000000000000000000000000000000000000000000000deadbeef"
	namespace := common.Hex2Bytes(stringNamespace)
	var bytes32Value [32]byte
	copy(bytes32Value[:], namespace)
	height := big.NewInt(1490181)
	start := big.NewInt(1)
	// height := big.NewInt(2040311)
	// start := big.NewInt(14)

	// height := big.NewInt(731137)
	// start := big.NewInt(0)
	payload, err := abiParsed.Pack(
		"CelestiaRequest",
		bytes32Value,
		&height,
		&start,
	)
	s.NoError(err)
	id := "0x" + common.Bytes2Hex(payload)
	slog.Debug("ID",
		"id", id,
		"namespace", namespace,
	)
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
	data, err := GetBlob(ctx, id, url, token)
	s.NoError(err)
	slog.Debug("GetBlob", "data", string(data))
	// s.Fail("x")
}

func TestCelestiaSuite(t *testing.T) {
	suite.Run(t, &CelestiaSuite{})
}
