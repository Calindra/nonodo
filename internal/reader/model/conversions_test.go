package model

import (
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/stretchr/testify/suite"
)

type ConversionsSuite struct {
	suite.Suite
}

func (s *ConversionsSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func TestConversionsSuite(t *testing.T) {
	suite.Run(t, new(ConversionsSuite))
}

func (s *ConversionsSuite) TestConvertConvenientVoucherV1() {
	cVoucher := cModel.ConvenienceVoucher{
		OutputHashesSiblings: `["0x01","0x02","0x03"]`,
	}
	graphVoucher := ConvertConvenientVoucherV1(cVoucher)
	s.Equal(3, len(graphVoucher.Proof.OutputHashesSiblings))
	s.Equal("0x01", graphVoucher.Proof.OutputHashesSiblings[0])
	s.Equal("0x02", graphVoucher.Proof.OutputHashesSiblings[1])
	s.Equal("0x03", graphVoucher.Proof.OutputHashesSiblings[2])
}
