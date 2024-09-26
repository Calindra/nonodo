package avail

import (
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/suite"
)

type AvailListenerSuite struct {
	suite.Suite
}

func TestAvailListenerSuite(t *testing.T) {
	suite.Run(t, &AvailListenerSuite{})
}

func (s *AvailListenerSuite) TestDecodeTimestamp() {
	// https://explorer.avail.so/#/extrinsics/decode/0x280403000b20008c2e9201
	timestamp := DecodeTimestamp("0b20008c2e9201")
	s.Equal(uint64(1727357780000), timestamp)
}

func (s *AvailListenerSuite) TestReadTimestampFromBlock() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := types.Extrinsic{
		Method: types.Call{
			Args: common.Hex2Bytes("0b20008c2e9201"),
			CallIndex: types.CallIndex{
				SectionIndex: 3,
				MethodIndex:  0,
			},
		},
	}
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	timestamp, err := ReadTimestampFromBlock(&block)
	s.NoError(err)
	s.Equal(uint64(1727357780000), timestamp)
}

func (s *AvailListenerSuite) TestReadTimestampFromBlockError() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := types.Extrinsic{}
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	_, err := ReadTimestampFromBlock(&block)
	s.ErrorContains(err, "block 0 without timestamp")
}

func (s *AvailListenerSuite) TestParsePaioFrom712Message() {
	typedData := apitypes.TypedData{
		Message: apitypes.TypedDataMessage{},
	}
	typedData.Message["app"] = "0x1111111111111111111111111111111111111111"
	typedData.Message["nonce"] = "1"
	typedData.Message["max_gas_price"] = "10"
	typedData.Message["data"] = "0xdeadff"
	message, err := ParsePaioFrom712Message(typedData)
	s.NoError(err)
	s.Equal("0x1111111111111111111111111111111111111111", message.App)
	s.Equal(common.Hex2Bytes("0xdeadff"), message.Payload)
}
