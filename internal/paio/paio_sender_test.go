package paio

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type PaioSender2ServerSuite struct {
	suite.Suite
}

func TestPaioSender2ServerSuite(t *testing.T) {
	suite.Run(t, new(PaioSender2ServerSuite))
}

func (s *PaioSender2ServerSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func (s *PaioSender2ServerSuite) TestPaioEncoder() {
	// nolint
	expected := `{"signature":"0x76a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c177ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da721c","message":"0x000000000000000000000000ab7528bb862fb57e8a2bcd567a2e929a0be56a5e0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000d48656c6c6f2c20576f726c643f00000000000000000000000000000000000000"}`

	dappAddress := common.HexToAddress(devnet.ApplicationAddress)
	payload := common.Hex2Bytes("48656c6c6f2c20576f726c643f")
	typedData := paiodecoder.CreateTypedData(
		dappAddress,
		uint64(0),
		big.NewInt(int64(10)),
		payload,
		big.NewInt(int64(11155111)),
	)
	typedDataJSON, err := json.Marshal(typedData)
	s.NoError(err)
	typedDataBase64 := base64.StdEncoding.EncodeToString(typedDataJSON)
	sigAndData := commons.SigAndData{
		Signature: "0x76a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c177ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da721c",
		TypedData: typedDataBase64,
	}
	encoded, err := EncodePaioFormat(sigAndData)
	s.NoError(err)
	s.Equal(expected, encoded)
}
