package model

import (
	"context"
	"log/slog"
	"testing"

	"github.com/calindra/nonodo/internal/convenience/config"
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
	config.ConfigureLog(slog.LevelDebug)
}

func TestStateSuite(t *testing.T) {
	suite.Run(t, new(StateSuite))
}

func (s *StateSuite) TestSendAllVouchersToDecoder() {
	decoder := FakeDecoder{}
	vouchers := []Voucher{}
	vouchers = append(vouchers, Voucher{
		Payload: common.Hex2Bytes("123456"),
	})
	sendAllInputVouchersToDecoder(&decoder, 1, vouchers)
	s.Equal(1, len(decoder.outputs))
	s.Equal(
		"ef615e2f123456",
		common.Bytes2Hex(decoder.outputs[0].Payload),
	)
}

func (s *StateSuite) TestSendAllNoticesToDecoder() {
	decoder := FakeDecoder{}
	notices := []Notice{}
	notices = append(notices, Notice{
		Payload: common.Hex2Bytes("123456"),
	})
	sendAllInputNoticesToDecoder(&decoder, 1, notices)
	s.Equal(1, len(decoder.outputs))
	s.Equal(
		"c258d6e5123456",
		common.Bytes2Hex(decoder.outputs[0].Payload),
	)
}
