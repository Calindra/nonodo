package avail

import (
	"log/slog"
	"math/big"
	"testing"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/suite"
)

type AvailListenerSuite struct {
	suite.Suite
}

func TestAvailListenerSuite(t *testing.T) {
	commons.ConfigureLog(slog.LevelDebug)
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
	typedData.Message["app"] = "0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e"
	typedData.Message["nonce"] = "1"
	typedData.Message["max_gas_price"] = "10"
	typedData.Message["data"] = "0xdeadff"
	message, err := ParsePaioFrom712Message(typedData)
	s.NoError(err)
	s.Equal("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e", message.App)
	s.Equal("0xdeadff", string(message.Payload))
}

func (s *AvailListenerSuite) TestReadInputsFromBlock() {
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
	// nolint
	jsonStr := `{"signature":"0x0a1bcb9c208b3e797e1561970322dc6ba7039b2303c5317d5cb0e970a684c6eb0c4a881c993ab2bc00cdbe95c22492dd4299567e0166f9062a731fba77d375531b","typedData":"eyJ0eXBlcyI6eyJDYXJ0ZXNpTWVzc2FnZSI6W3sibmFtZSI6ImFwcCIsInR5cGUiOiJhZGRyZXNzIn0seyJuYW1lIjoibm9uY2UiLCJ0eXBlIjoidWludDY0In0seyJuYW1lIjoibWF4X2dhc19wcmljZSIsInR5cGUiOiJ1aW50MTI4In0seyJuYW1lIjoiZGF0YSIsInR5cGUiOiJzdHJpbmcifV0sIkVJUDcxMkRvbWFpbiI6W3sibmFtZSI6Im5hbWUiLCJ0eXBlIjoic3RyaW5nIn0seyJuYW1lIjoidmVyc2lvbiIsInR5cGUiOiJzdHJpbmcifSx7Im5hbWUiOiJjaGFpbklkIiwidHlwZSI6InVpbnQyNTYifSx7Im5hbWUiOiJ2ZXJpZnlpbmdDb250cmFjdCIsInR5cGUiOiJhZGRyZXNzIn1dfSwicHJpbWFyeVR5cGUiOiJDYXJ0ZXNpTWVzc2FnZSIsImRvbWFpbiI6eyJuYW1lIjoiQXZhaWxNIiwidmVyc2lvbiI6IjEiLCJjaGFpbklkIjoiMHgzZTkiLCJ2ZXJpZnlpbmdDb250cmFjdCI6IjB4Q2NDQ2NjY2NDQ0NDY0NDQ0NDQ2NDY0NjY0NjQ0NDY0NjY2NjY2NjQyIsInNhbHQiOiIifSwibWVzc2FnZSI6eyJhcHAiOiIweGFiNzUyOGJiODYyZmI1N2U4YTJiY2Q1NjdhMmU5MjlhMGJlNTZhNWUiLCJkYXRhIjoiR00iLCJtYXhfZ2FzX3ByaWNlIjoiMTAiLCJub25jZSI6IjEifX0="}`
	timestampExtrinsicInput := types.Extrinsic{
		Method: types.Call{
			Args: ([]byte(jsonStr)),
			CallIndex: types.CallIndex{
				SectionIndex: 0,
				MethodIndex:  0,
			},
		},
		Signature: types.ExtrinsicSignatureV4{
			AppID: types.UCompact(*big.NewInt(DEFAULT_APP_ID)),
		},
	}
	block.Block.Extrinsics = append(block.Block.Extrinsics, timestampExtrinsicInput)
	/*
		inputFromL1 := ReadInputsFromL1(&block)
		inputs, err := ReadInputsFromAvailBlock(&block)
		for ...
			dbIndex = repo
			repo.Create()
	*/
	inputs, err := ReadInputsFromAvailBlock(&block)
	s.NoError(err)
	s.Equal(1, len(inputs))
	s.Equal(common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e"), inputs[0].AppContract)
	s.Equal("GM", string(inputs[0].Payload))
	s.Equal(common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"), inputs[0].MsgSender)
}
