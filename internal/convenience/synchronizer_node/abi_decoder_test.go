package synchronizernode

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type AbiDecoderSuite struct {
	suite.Suite
	ctx        context.Context
	AbiDecoder *AbiDecoder
}

func (s *AbiDecoderSuite) SetupTest() {
	s.ctx = context.Background()
	commons.ConfigureLog(slog.LevelDebug)
	abi, err := contracts.OutputsMetaData.GetAbi()
	if err != nil {
		s.Require().NoError(err)
	}
	s.AbiDecoder = NewAbiDecoder(abi)
}

func TestAbiDecoderSuiteSuite(t *testing.T) {
	suite.Run(t, new(AbiDecoderSuite))
}

// nolint
func (s *AbiDecoderSuite) TestVoucherMapFromGetMapRaw() {
	rawDataStr := "237a816f000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb9226600000000000000000000000000000000000000000000000000000000deadbeef00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000005deadbeef01000000000000000000000000000000000000000000000000000000"
	rawData := common.Hex2Bytes(rawDataStr)
	dataMap, err := s.AbiDecoder.GetMapRaw(rawData)
	s.Require().NoError(err)
	s.Equal("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", dataMap["destination"].(common.Address).Hex())
	s.NotNil(dataMap["value"])
}

// nolint
func (s *AbiDecoderSuite) TestNoticeMapFromGetMapRaw() {
	rawDataStr := "c258d6e500000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000005deadbeef00000000000000000000000000000000000000000000000000000000"
	rawData := common.Hex2Bytes(rawDataStr)
	dataMap, err := s.AbiDecoder.GetMapRaw(rawData)
	s.Require().NoError(err)
	s.NotNil(dataMap["payload"])
	s.Nil(dataMap["destination"])
	s.Nil(dataMap["value"])
}
