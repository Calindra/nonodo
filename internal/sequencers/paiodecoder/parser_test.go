package paiodecoder

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/suite"
)

type ParserSuite struct {
	suite.Suite
}

func (s *ParserSuite) SetupTest() {
	// Log
	commons.ConfigureLog(slog.LevelDebug)
}

func TestParserSuite(t *testing.T) {
	suite.Run(t, new(ParserSuite))
}

func (s *ParserSuite) TestDecodeBytes() {
	ctx := context.Background()
	binLocation, err := DownloadPaioDecoderExecutableAsNeeded()
	s.Require().NoError(err)
	// nolint
	bytes := `0x1400000000000000000000000000000000000000000114ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000a07deadbeeffab10920205f3aa429e8ea753d2e799fa4bf9166264d4114745fb4670eaed856f0dae8e5204e74225ca715f951fed4bec7b0bc635f067183ac3ec44139533b0715e04b4da808000000000000001c`
	decoder := NewPaioDecoder(binLocation)
	json, err := decoder.DecodePaioBatch(ctx, common.Hex2Bytes(bytes[2:]))
	s.Require().NoError(err)
	slog.Debug("decoded", "json", json)
	// nolint
	expected := `{"sequencer_payment_address":"0x0000000000000000000000000000000000000000","txs":[{"app":"0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e","nonce":0,"max_gas_price":10,"data":[222,173,190,239,250,177,9],"signature":{"r":"0x205f3aa429e8ea753d2e799fa4bf9166264d4114745fb4670eaed856f0dae8e5","s":"0x4e74225ca715f951fed4bec7b0bc635f067183ac3ec44139533b0715e04b4da8","v":"0x1c"}}]}`
	s.Equal(expected, json)
}

func (s *ParserSuite) TestParsePaioFrom712Message() {
	typedData := apitypes.TypedData{
		Message: apitypes.TypedDataMessage{},
	}
	typedData.Message["app"] = "0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e"
	typedData.Message["nonce"] = "1"
	typedData.Message["max_gas_price"] = "10"
	typedData.Message["data"] = "0xdeadff"
	message, err := ParsePaioFrom712Message(typedData)
	s.NoError(err)
	s.Equal("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e", message.App)
	s.Equal("0xdeadff", string(message.Payload))
}

func (s *ParserSuite) TestParsePaioBatchToInputs() {
	// nolint
	jsonStr := `{"sequencer_payment_address":"0x63F9725f107358c9115BC9d86c72dD5823E9B1E6","txs":[{"app":"0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e","nonce":0,"max_gas_price":10,"data":[72,101,108,108,111,44,32,87,111,114,108,100,63],"signature":{"r":"0x76a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17","s":"0x7ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da72","v":"0x1c"}}]}`

	chainId := big.NewInt(11155111)
	inputs, err := ParsePaioBatchToInputs(jsonStr, chainId)
	s.NoError(err)
	s.Equal(1, len(inputs))

	// changed to new msg_sender because domain name changed from CartesiPaio to Cartesi,
	// so hash changed and then public key also changed
	s.Equal("0x631e372a9Ed7808Cbf55117f3263d3e1c9Bc3710", inputs[0].MsgSender.Hex())
	s.Equal("0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e", inputs[0].AppContract.Hex())
	s.Equal("Hello, World?", string(common.Hex2Bytes(inputs[0].Payload)))
}

func (s *ParserSuite) TestParsePaioBatchToInputsWeirdSignature() {
	// signature.s has an odd length
	// nolint
	jsonStr := `{
		"sequencer_payment_address":"0x63F9725f107358c9115BC9d86c72dD5823E9B1E6",
		"txs":[
			{
				"app":"0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e",
				"nonce":0,
				"max_gas_price":10,
				"data":[222,173,190,239,167],
				"signature":{
					"r":"0x6c4a7d4453adafa14fd9d79c3f6bab2f482f88f4f212971cd1fdc9c1a2bbdc88",
					"s":"0x17b79806e5af212437f3a2dd7a8c9f4575156b082b130369e29032792ff41e5",
					"v":"0x1c"
				}
			}
		]
	}`

	chainId := big.NewInt(31337)
	inputs, err := ParsePaioBatchToInputs(jsonStr, chainId)
	s.NoError(err)
	s.Equal(1, len(inputs))

	// changed to new msg_sender because domain name changed from CartesiPaio to Cartesi,
	// so hash changed and then public key also changed
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", inputs[0].MsgSender.Hex())
	s.Equal("0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e", inputs[0].AppContract.Hex())
	s.Equal("deadbeefa7", inputs[0].Payload)
}
