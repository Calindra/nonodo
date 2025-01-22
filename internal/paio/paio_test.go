package paio

import (
	"encoding/json"
	"log/slog"
	"math/big"
	"strings"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type PaioSuite struct {
	suite.Suite
}

type PaioDefinition struct {
	Address     common.Address `json:"address"`
	Nonce       uint64         `json:"nonce"`
	MaxGasPrice *big.Int       `json:"max_gas_price"`
	Data        []byte         `json:"data"`
}

func (s *PaioSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func (p *PaioSuite) TestPaioEncoder() {
	// test encoding
	abi, err := abi.JSON(strings.NewReader(DEFINITION))
	p.NoError(err)

	app := common.HexToAddress("0xdeadbeef")
	nonce := uint64(123)
	maxGasPrice := new(big.Int).SetUint64(456)
	data := []byte("hello world")
	encoded, err := abi.Pack("signingMessage", app, nonce, maxGasPrice, data)
	p.NoError(err)
	hexa := common.Bytes2Hex(encoded)
	slog.Debug("encoded", "hexa", hexa)
}

func (p *PaioSuite) TestPaioDecoder() {
	// test decoding
	abi, err := abi.JSON(strings.NewReader(DEFINITION))
	p.NoError(err)

	dataEncoded := "0xd24f8fa800000000000000000000000000000000000000000000000000000000deadbeef000000000000000000000000000000000000000000000000000000000000007b00000000000000000000000000000000000000000000000000000000000001c80000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b68656c6c6f20776f726c64000000000000000000000000000000000000000000" // nolint

	data := common.Hex2Bytes(strings.TrimPrefix(dataEncoded, "0x"))
	decoded := PaioDefinition{
		Address:     common.HexToAddress("0x0"),
		Nonce:       0,
		MaxGasPrice: new(big.Int).SetUint64(0),
		Data:        []byte{},
	}
	method, err := abi.MethodById(data)
	p.NoError(err)
	args := method.Inputs
	decodedMap := make(map[string]any)
	err = args.UnpackIntoMap(decodedMap, data[4:])
	p.NoError(err)

	for key, val := range decodedMap {
		switch key {
		case "account":
			decoded.Address = val.(common.Address)
		case "nonce":
			decoded.Nonce = val.(uint64)
		case "max_gas_price":
			decoded.MaxGasPrice = val.(*big.Int)
		case "data":
			decoded.Data = val.([]byte)
		}
	}

	output, err := json.Marshal(decoded)
	p.NoError(err)

	slog.Debug("decoded", "method", method, "args", args, "decoded", string(output), "data", string(decoded.Data))
}

func TestPaioSuite(t *testing.T) {
	suite.Run(t, new(PaioSuite))
}
