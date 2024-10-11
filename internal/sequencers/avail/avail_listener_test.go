package avail

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"os/exec"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/suite"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type AvailListenerSuite struct {
	suite.Suite
	fd            paiodecoder.DecoderPaio
	ctx           context.Context
	workerCtx     context.Context
	timeoutCancel context.CancelFunc
	workerCancel  context.CancelFunc
	workerResult  chan error
	rpcUrl        string
	portCounter   int
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
	timestampExtrinsic := CreateTimestampExtrinsic()
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

func (s *AvailListenerSuite) TestReadInputsFromBlockZzzHui() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := CreateTimestampExtrinsic()
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	// nolint
	jsonStr := `{"signature":"0x0a1bcb9c208b3e797e1561970322dc6ba7039b2303c5317d5cb0e970a684c6eb0c4a881c993ab2bc00cdbe95c22492dd4299567e0166f9062a731fba77d375531b","typedData":"eyJ0eXBlcyI6eyJDYXJ0ZXNpTWVzc2FnZSI6W3sibmFtZSI6ImFwcCIsInR5cGUiOiJhZGRyZXNzIn0seyJuYW1lIjoibm9uY2UiLCJ0eXBlIjoidWludDY0In0seyJuYW1lIjoibWF4X2dhc19wcmljZSIsInR5cGUiOiJ1aW50MTI4In0seyJuYW1lIjoiZGF0YSIsInR5cGUiOiJzdHJpbmcifV0sIkVJUDcxMkRvbWFpbiI6W3sibmFtZSI6Im5hbWUiLCJ0eXBlIjoic3RyaW5nIn0seyJuYW1lIjoidmVyc2lvbiIsInR5cGUiOiJzdHJpbmcifSx7Im5hbWUiOiJjaGFpbklkIiwidHlwZSI6InVpbnQyNTYifSx7Im5hbWUiOiJ2ZXJpZnlpbmdDb250cmFjdCIsInR5cGUiOiJhZGRyZXNzIn1dfSwicHJpbWFyeVR5cGUiOiJDYXJ0ZXNpTWVzc2FnZSIsImRvbWFpbiI6eyJuYW1lIjoiQXZhaWxNIiwidmVyc2lvbiI6IjEiLCJjaGFpbklkIjoiMHgzZTkiLCJ2ZXJpZnlpbmdDb250cmFjdCI6IjB4Q2NDQ2NjY2NDQ0NDY0NDQ0NDQ2NDY0NjY0NjQ0NDY0NjY2NjY2NjQyIsInNhbHQiOiIifSwibWVzc2FnZSI6eyJhcHAiOiIweGFiNzUyOGJiODYyZmI1N2U4YTJiY2Q1NjdhMmU5MjlhMGJlNTZhNWUiLCJkYXRhIjoiR00iLCJtYXhfZ2FzX3ByaWNlIjoiMTAiLCJub25jZSI6IjEifX0="}`
	extrinsicInput := CreatePaioExtrinsic([]byte(jsonStr))
	block.Block.Extrinsics = append(block.Block.Extrinsics, extrinsicInput)
	/*
		inputFromL1 := ReadInputsFromL1(&block)
		inputs, err := ReadInputsFromAvailBlock(&block)
		for ...
			dbIndex = repo
			repo.Create()
	*/
	inputs, err := ReadInputsFromAvailBlockZzzHui(&block)
	s.NoError(err)
	s.Equal(1, len(inputs))
	s.Equal(common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e"), inputs[0].AppContract)
	s.Equal("GM", string(inputs[0].Payload))
	s.Equal(common.HexToAddress("0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"), inputs[0].MsgSender)
}

func (s *AvailListenerSuite) TestReadInputsFromBlockPaio() {
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := CreateTimestampExtrinsic()
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	// nolint
	fromPaio := "0x1463f9725f107358c9115bc9d86c72dd5823e9b1e60114ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000a0d48656c6c6f2c20576f726c643f2076a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17207ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da7208000000000000001c"
	extrinsicPaioBlock := CreatePaioExtrinsic(common.Hex2Bytes(fromPaio))
	block.Block.Extrinsics = append(block.Block.Extrinsics, extrinsicPaioBlock)
	availListener := AvailListener{
		PaioDecoder: s.fd,
		InputterWorker: &inputter.InputterWorker{
			Provider: s.rpcUrl,
		},
	}
	inputs, err := availListener.ReadInputsFromPaioBlock(s.ctx, &block)
	s.NoError(err)
	s.Equal(1, len(inputs))
}

func (s *AvailListenerSuite) TestParsePaioBatchToInputs() {
	jsonStr, err := s.fd.DecodePaioBatch(s.ctx, "it doesn't matter")
	s.NoError(err)
	chainId := big.NewInt(11155111)
	inputs, err := ParsePaioBatchToInputs(jsonStr, chainId)
	s.NoError(err)
	s.Equal(1, len(inputs))

	// changed to new msg_sender because domain name changed from CartesiPaio to Cartesi,
	// so hash changed and then public key also changed
	s.Equal("0x631e372a9Ed7808Cbf55117f3263d3e1c9Bc3710", inputs[0].MsgSender.Hex())
	s.Equal("0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e", inputs[0].AppContract.Hex())
	s.Equal("Hello, World?", string(inputs[0].Payload))
}

func (s *AvailListenerSuite) TestTableTennis() {
	ethClient, err := ethclient.DialContext(s.ctx, s.rpcUrl)
	s.NoError(err)
	appAddress := common.HexToAddress("0xab7528bb862fb57e8a2bcd567a2e929a0be56a5e")
	inputBoxAddress := common.HexToAddress("0x58Df21fE097d4bE5dCf61e01d9ea3f6B81c2E1dB")
	inputBox, err := contracts.NewInputBox(inputBoxAddress, ethClient)
	s.NoError(err)
	ctx := context.Background()
	err = devnet.AddInput(ctx, s.rpcUrl, common.Hex2Bytes("deadbeef11"))
	s.NoError(err)

	l1FinalizedPrevHeight := uint64(1)
	timestamp := uint64(time.Now().UnixMilli())
	inputterWorker := inputter.InputterWorker{
		Model:              nil,
		Provider:           s.rpcUrl,
		InputBoxAddress:    inputBoxAddress,
		InputBoxBlock:      1,
		ApplicationAddress: appAddress,
	}
	// Avail's block
	block := types.SignedBlock{}
	block.Block = types.Block{}
	timestampExtrinsic := CreateTimestampExtrinsic()
	delta := 350 * 1000
	timestampExtrinsic.Method.Args = encodeCompactU64(timestamp + uint64(delta))
	block.Block.Extrinsics = append([]types.Extrinsic{}, timestampExtrinsic)
	// nolint
	fromPaio := "0x1463f9725f107358c9115bc9d86c72dd5823e9b1e60114ab7528bb862fb57e8a2bcd567a2e929a0be56a5e000a0d48656c6c6f2c20576f726c643f2076a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17207ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da7208000000000000001c"
	extrinsicPaioBlock := CreatePaioExtrinsic(common.Hex2Bytes(fromPaio))
	block.Block.Extrinsics = append(block.Block.Extrinsics, extrinsicPaioBlock)

	dbFactory := commons.NewDbFactory()
	db := dbFactory.CreateDb("input.sqlite3")
	inputRepository := &repository.InputRepository{
		Db: *db,
	}
	err = inputRepository.CreateTables()
	s.NoError(err)
	fd := FakeDecoder{}
	availListener := AvailListener{
		PaioDecoder:     &fd,
		InputterWorker:  &inputterWorker,
		InputRepository: inputRepository,
	}
	inputs, err := availListener.ReadInputsFromPaioBlock(ctx, &block)
	s.NoError(err)
	s.Equal(1, len(inputs))
	s.Equal(int64(timestamp)+int64(delta), inputs[0].BlockTimestamp.Unix())
	availBlockTimestamp := uint64(inputs[0].BlockTimestamp.Unix())
	inputs, err = inputterWorker.FindAllInputsByBlockAndTimestampLT(ctx, ethClient, inputBox, l1FinalizedPrevHeight, (availBlockTimestamp/1000)-300)
	s.NoError(err)
	s.NotNil(inputs)
	s.Equal(1, len(inputs))

	startBlock := 0
	currentL1Block, err := availListener.TableTennis(s.ctx, &block, ethClient, inputBox, uint64(startBlock))
	s.NoError(err)
	s.NotNil(currentL1Block)

	// check if TableTennis has saved the data.
	savedInputs, err := inputRepository.FindAll(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(2, int(savedInputs.Total))

	// check the input from InputBox
	s.Equal("0", savedInputs.Rows[0].ID)
	s.Equal("deadbeef11", common.Bytes2Hex(savedInputs.Rows[0].Payload))

	// check the input from Avail
	s.Equal("0x4adf75e71bb8831739bfccd25958f03ca057d5df8b93a50e3fb7dae1e540faa7",
		savedInputs.Rows[1].ID)
	s.Equal("Hello, World?", string(savedInputs.Rows[1].Payload))
}

type FakeDecoder struct {
}

func (fd *FakeDecoder) DecodePaioBatch(ctx context.Context, bytes string) (string, error) {
	// nolint
	return `{"sequencer_payment_address":"0x63F9725f107358c9115BC9d86c72dD5823E9B1E6","txs":[{"app":"0xab7528bb862fB57E8A2BCd567a2e929a0Be56a5e","nonce":0,"max_gas_price":10,"data":[72,101,108,108,111,44,32,87,111,114,108,100,63],"signature":{"r":"0x76a270f52ade97cd95ef7be45e08ea956bfdaf14b7fc4f8816207fa9eb3a5c17","s":"0x7ccdd94ac1bd86a749b66526fff6579e2b6bf1698e831955332ad9d5ed44da72","v":"0x1c"}}]}`, nil
}

func CreatePaioExtrinsic(args []byte) types.Extrinsic {
	return types.Extrinsic{
		Method: types.Call{
			Args: args,
			CallIndex: types.CallIndex{
				SectionIndex: 0,
				MethodIndex:  0,
			},
		},
		Signature: types.ExtrinsicSignatureV4{
			AppID: types.UCompact(*big.NewInt(DEFAULT_APP_ID)),
		},
	}
}

func CreateTimestampExtrinsic() types.Extrinsic {
	return types.Extrinsic{
		Method: types.Call{
			Args: common.Hex2Bytes("0b20008c2e9201"),
			CallIndex: types.CallIndex{
				SectionIndex: 3,
				MethodIndex:  0,
			},
		},
	}
}

func (s *AvailListenerSuite) SetupTest() {
	commons.ConfigureLog(slog.LevelDebug)
	var w supervisor.SupervisorWorker
	w.Name = "SupervisorWorker"
	const testTimeout = 5 * time.Second
	s.portCounter += 1
	s.ctx, s.timeoutCancel = context.WithTimeout(context.Background(), testTimeout)
	s.workerResult = make(chan error)
	s.fd = &FakeDecoder{}
	s.workerCtx, s.workerCancel = context.WithCancel(s.ctx)
	w.Workers = append(w.Workers, devnet.AnvilWorker{
		Address:  devnet.AnvilDefaultAddress,
		Port:     devnet.AnvilDefaultPort + s.portCounter,
		Verbose:  true,
		AnvilCmd: "anvil",
	})
	s.rpcUrl = fmt.Sprintf("ws://%s:%v", devnet.AnvilDefaultAddress, devnet.AnvilDefaultPort+s.portCounter)
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

func (s *AvailListenerSuite) TearDownTest() {
	err := exec.Command("pkill", "anvil").Run()
	s.NoError(err)
	s.workerCancel()
	select {
	case <-s.ctx.Done():
		s.Fail("context error", s.ctx.Err())
	case err := <-s.workerResult:
		s.NoError(err)
	}
	s.timeoutCancel()
	s.T().Log("teardown ok.")
}
