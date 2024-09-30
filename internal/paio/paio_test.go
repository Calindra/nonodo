package paio

import (
	"log/slog"
	"math/big"
	"strings"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type PaioSuite struct {
	suite.Suite
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
	// decoded := PaioDefinition{
	// 	Address:     common.HexToAddress("0x0"),
	// 	Nonce:       0,
	// 	MaxGasPrice: new(big.Int).SetUint64(0),
	// 	Data:        []byte{},
	// }
	method, err := abi.MethodById(data)
	p.NoError(err)
	args := method.Inputs
	decoded, err := args.Unpack(data[4:])
	p.NoError(err)
	slog.Info("decoded", "decoded", decoded, "method", method, "args", args)
}

func TestPaioSuite(t *testing.T) {
	suite.Run(t, new(PaioSuite))
}
