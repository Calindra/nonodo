package avail

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/suite"
)

type AvailListenerSuite struct {
	suite.Suite
	ctx           context.Context
	workerCtx     context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	rpcUrl        string
}

func TestAvailListenerSuite(t *testing.T) {
	commons.ConfigureLog(slog.LevelDebug)
	suite.Run(t, &AvailListenerSuite{})
}

func (s *AvailListenerSuite) SetupTest() {
	var w supervisor.SupervisorWorker
	w.Name = "TesteInputter"
	const testTimeout = 5 * time.Second
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)

	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	anvilLocation, err := devnet.CheckAnvilAndInstall(s.ctx)
	s.NoError(err)
	w.Workers = append(w.Workers, devnet.AnvilWorker{
		Address:  devnet.AnvilDefaultAddress,
		Port:     devnet.AnvilDefaultPort,
		Verbose:  true,
		AnvilCmd: anvilLocation,
	})
	// var workerCtx context.Context

	s.rpcUrl = fmt.Sprintf("ws://%s:%v", devnet.AnvilDefaultAddress, devnet.AnvilDefaultPort)
	ready := make(chan struct{})
	go func() {
		s.workerResult <- w.Start(s.workerCtx, ready)
	}()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.Fail("worker exited before being ready", err)
	case <-ready:
		s.T().Log("nonodo ready")
	}
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

func (s *AvailListenerSuite) TestReadBlocksFromInputBox() {
	block := types.SignedBlock{}
	block.Block = types.Block{}

	currentTimestamp := time.Now().UnixMilli()
	timestampPlusOneDay := uint64(currentTimestamp + 86400*1000) // 86400 segundos em um dia

	// Codificar o timestampPlusOneDay para o formato compactado
	encodedTimestamp := encodeCompactU64(timestampPlusOneDay)
	hexTimestamp := hex.EncodeToString(encodedTimestamp)

	slog.Debug(hexTimestamp)

	timestampExtrinsic := types.Extrinsic{
		Method: types.Call{
			Args: common.Hex2Bytes(hexTimestamp),
			CallIndex: types.CallIndex{
				SectionIndex: 3,
				MethodIndex:  0,
			},
		},
	}
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)

	appAddress := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	inputBoxAddress := common.HexToAddress("0x58Df21fE097d4bE5dCf61e01d9ea3f6B81c2E1dB")
	err := devnet.AddInput(s.ctx, s.rpcUrl, common.Hex2Bytes("deadbeef"))
	s.NoError(err)
	l1FinalizedPrevHeight := uint64(1)
	w := inputter.InputterWorker{
		Model:              nil,
		Provider:           s.rpcUrl,
		InputBoxAddress:    inputBoxAddress,
		InputBoxBlock:      1,
		ApplicationAddress: appAddress,
	}

	inputs, err := ReadInputsFromInputBox(s.ctx, &w, &block, l1FinalizedPrevHeight)

	s.NoError(err)
	s.Equal(1, len(inputs))

}

// Codificação compacta do timestamp para bytes
func encodeCompactU64(value uint64) []byte {
	if value < 1<<6 {
		return []byte{byte(value << 2)}
	} else if value < 1<<14 {
		return []byte{byte((value << 2) | 0b01), byte(value >> 6)}
	} else if value < 1<<30 {
		return []byte{byte((value << 2) | 0b10), byte(value >> 6), byte(value >> 14), byte(value >> 22)}
	} else {
		bytes := make([]byte, 9)
		bytes[0] = 0b11
		for i := 0; i < 8; i++ {
			bytes[i+1] = byte(value >> (8 * i))
		}
		return bytes
	}
}
