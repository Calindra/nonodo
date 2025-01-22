package model

import (
	"context"
	"log/slog"
	"testing"

	"github.com/cartesi/rollups-graphql/pkg/commons"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type StateSuite struct {
	suite.Suite
}

type GenericOutput struct {
	Index       int
	InputIndex  int
	Destination common.Address
	Payload     []byte
}
type FakeDecoder struct {
	outputs []GenericOutput
}

func (f *FakeDecoder) HandleOutput(
	ctx context.Context,
	destination common.Address,
	payload string,
	inputIndex uint64,
	outputIndex uint64,
) error {
	slog.Debug("HandleOutput", "payload", payload)
	f.outputs = append(f.outputs, GenericOutput{
		Destination: destination,
		Payload:     common.Hex2Bytes(payload[2:]),
		Index:       int(outputIndex),
		InputIndex:  int(inputIndex),
	})
	return nil
}

func (s *StateSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
}

func TestStateSuite(t *testing.T) {
	suite.Run(t, new(StateSuite))
}

// func (s *StateSuite) TestSendAllNoticesToDecoder() {
// 	decoder := FakeDecoder{}
// 	notices := []cModel.ConvenienceNotice{}
// 	notices = append(notices, cModel.ConvenienceNotice{
// 		Payload: "123456",
// 	})
// 	err := sendAllInputNoticesToDecoder(&decoder, 1, notices)
// 	s.NoError(err)
// 	s.Equal(1, len(decoder.outputs))
// 	s.Equal(
// 		"c258d6e5123456",
// 		common.Bytes2Hex(decoder.outputs[0].Payload),
// 	)
// }
