package paiodecoder

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
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
	json, err := decoder.DecodePaioBatch(ctx, bytes)
	s.Require().NoError(err)
	slog.Debug("decoded", "json", json)
	// nolint
	expected := `{"sequencer_payment_address":"0x0000000000000000000000000000000000000000","txs":[{"app":"0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e","nonce":0,"max_gas_price":10,"data":[222,173,190,239,250,177,9],"signature":{"r":"0x205f3aa429e8ea753d2e799fa4bf9166264d4114745fb4670eaed856f0dae8e5","s":"0x4e74225ca715f951fed4bec7b0bc635f067183ac3ec44139533b0715e04b4da8","v":"0x1c"}}]}`
	s.Equal(expected, json)
}
