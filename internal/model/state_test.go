package model

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type StateSuite struct {
	suite.Suite
}

type FakeDecoder struct {
	vouchers []Voucher
}

func (f *FakeDecoder) HandleOutput(
	ctx context.Context,
	destination common.Address,
	payload string,
	inputIndex uint64,
	outputIndex uint64,
) error {
	f.vouchers = append(f.vouchers, Voucher{
		Destination: destination,
		Payload:     common.Hex2Bytes(payload),
		Index:       int(outputIndex),
		InputIndex:  int(inputIndex),
	})
	return nil
}

func (s *StateSuite) SetupTest() {

}

func TestStateSuite(t *testing.T) {
	suite.Run(t, new(StateSuite))
}

//
// AddAdvanceInput
//

func (s *StateSuite) TestSendAllVouchersToDecoder() {
	decoder := FakeDecoder{}
	vouchers := []Voucher{}
	vouchers = append(vouchers, Voucher{})
	sendAllInputVouchersToDecoder(&decoder, 1, vouchers)
	s.Equal(1, len(decoder.vouchers))
}
