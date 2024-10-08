package avail

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	cModel "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/calindra/nonodo/internal/supervisor"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	TIMESTAMP_SECTION_INDEX = 3
	DELAY                   = 500
	ONE_SECOND_IN_MS        = 1000
	FIVE_MINUTES            = 300
)

type AvailListener struct {
	AvailFromBlock  uint64
	InputRepository *cRepos.InputRepository
	InputterWorker  *inputter.InputterWorker
	PaioDecoder     PaioDecoder
	L1CurrentBlock  uint64
}

type PaioDecoder interface {
	DecodePaioBatch(bytes string) (string, error)
}

func NewAvailListener(availFromBlock uint64, repository *cRepos.InputRepository, w *inputter.InputterWorker, fromBlock uint64) supervisor.Worker {
	paioDecoder := ZzzHuiDecoder{}
	return AvailListener{
		AvailFromBlock:  availFromBlock,
		InputRepository: repository,
		InputterWorker:  w,
		PaioDecoder:     paioDecoder,
		L1CurrentBlock:  fromBlock,
	}
}

func (a AvailListener) String() string {
	return "avail_listener"
}

func (a AvailListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	client, err := a.connect(ctx)
	if err != nil {
		slog.Error("Avail", "Error connecting to Avail", err)
		return err
	}
	return a.watchNewTransactions(ctx, client)
}

func (a AvailListener) connect(ctx context.Context) (*gsrpc.SubstrateAPI, error) {
	// uses env RPC_URL for connecting
	// cfg := config.Default()

	// cfg := config.Config{}
	// err := cfg.GetConfig("config.json")
	// if err != nil {
	// 	return nil, err
	// }
	rpcURL, haveURL := os.LookupEnv("AVAIL_RPC_URL")
	if !haveURL {
		rpcURL = DEFAULT_AVAIL_RPC_URL
	}

	errCh := make(chan error)
	clientCh := make(chan *gsrpc.SubstrateAPI)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
			default:
				client, err := NewSubstrateAPICtx(ctx, rpcURL)
				if err != nil {
					slog.Error("Avail", "Error connecting to Avail client", err)
					slog.Info("Avail reconnecting client", "retryInterval", retryInterval)
					time.Sleep(retryInterval)
				} else {
					clientCh <- client
					return
				}

			}
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
	case client := <-clientCh:
		return client, nil
	}
}

const retryInterval = 5 * time.Second

func (a AvailListener) watchNewTransactions(ctx context.Context, client *gsrpc.SubstrateAPI) error {
	latestAvailBlock := a.AvailFromBlock
	var index uint = 0
	defer client.Client.Close()

	ethClient, err := a.InputterWorker.GetEthClient()
	if err != nil {
		return fmt.Errorf("avail inputter: dial: %w", err)
	}
	inputBox, err := contracts.NewInputBox(a.InputterWorker.InputBoxAddress, ethClient)
	if err != nil {
		return fmt.Errorf("avail inputter: bind input box: %w", err)
	}

	for {
		if latestAvailBlock == 0 {
			block, err := client.RPC.Chain.GetHeaderLatest()
			if err != nil {
				slog.Error("Avail", "Error getting latest block hash", err)
				slog.Info("Avail reconnecting", "retryInterval", retryInterval)
				time.Sleep(retryInterval)
				continue
			}

			slog.Info("Avail", "Set last block", block.Number)
			latestAvailBlock = uint64(block.Number)
		}

		subscription, err := client.RPC.Chain.SubscribeNewHeads()
		if err != nil {
			slog.Error("Avail", "Error subscribing to new heads", err)
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
			continue
		}
		defer subscription.Unsubscribe()

		errCh := make(chan error)

		go func() {
			for {
				select {
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				case err := <-subscription.Err():
					errCh <- err
					return
				case <-time.After(DELAY * time.Millisecond):

				case i := <-subscription.Chan():
					for latestAvailBlock <= uint64(i.Number) {
						index++

						if latestAvailBlock < uint64(i.Number) {
							slog.Debug("Avail Catching up", "Chain is at block", i.Number, "fetching block", latestAvailBlock)
						} else {
							slog.Debug("Avail", "index", index, "Chain is at block", i.Number, "fetching block", latestAvailBlock)
						}

						blockHash, err := client.RPC.Chain.GetBlockHash(latestAvailBlock)
						if err != nil {
							errCh <- err
							return
						}
						block, err := client.RPC.Chain.GetBlock(blockHash)
						if err != nil {
							errCh <- err
							return
						}
						currentL1Block, err := a.TableTennis(ctx, block,
							ethClient, inputBox,
							a.L1CurrentBlock,
						)
						if err != nil {
							errCh <- err
							return
						}
						if currentL1Block != nil && *currentL1Block > 0 {
							a.L1CurrentBlock = *currentL1Block
						}
						latestAvailBlock += 1
						time.Sleep(500 * time.Millisecond) // nolint
					}
				}
			}
		}()

		err = <-errCh
		subscription.Unsubscribe()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			slog.Error("Avail", "Error", err)
			slog.Info("Avail reconnecting", "retryInterval", retryInterval)
			time.Sleep(retryInterval)
		} else {
			return nil
		}
	}
}

func (a AvailListener) TableTennis(ctx context.Context,
	block *types.SignedBlock, ethClient *ethclient.Client,
	inputBox *contracts.InputBox, startBlockNumber uint64) (*uint64, error) {

	var currentL1Block uint64
	availInputs, err := a.ReadInputsFromPaioBlock(block)
	if err != nil {
		return nil, err
	}
	var availBlockTimestamp uint64
	if len(availInputs) == 0 {
		availBlockTimestamp, err = ReadTimestampFromBlock(block)
		if err != nil {
			return nil, err
		}
	} else {
		availBlockTimestamp = uint64(availInputs[0].BlockTimestamp.Unix())
	}
	inputsL1, err := a.InputterWorker.FindAllInputsByBlockAndTimestampLT(ctx,
		ethClient, inputBox, startBlockNumber,
		(availBlockTimestamp/ONE_SECOND_IN_MS)-FIVE_MINUTES,
	)
	if err != nil {
		return nil, err
	}
	if len(inputsL1) > 0 {
		currentL1Block = inputsL1[len(inputsL1)-1].BlockNumber + 1
	}
	inputs := append(inputsL1, availInputs...)
	if len(inputs) > 0 {
		inputCount, err := a.InputRepository.Count(ctx, nil)
		if err != nil {
			return nil, err
		}
		for i := range inputs {
			inputs[i].Index = i + int(inputCount)
			_, err = a.InputRepository.Create(ctx, inputs[i])
			if err != nil {
				return nil, err
			}
		}
	}
	return &currentL1Block, nil
}

func DecodeTimestamp(hexStr string) uint64 {
	decoded, err := hex.DecodeString(padHexStringRight(hexStr))
	if err != nil {
		fmt.Println("Error decoding hex:", err)
		return 0
	}
	return decodeCompactU64(decoded)
}

// nolint
func decodeCompactU64(data []byte) uint64 {
	firstByte := data[0]
	if firstByte&0b11 == 0b00 { // Single byte (6-bit value)
		return uint64(firstByte >> 2)
	} else if firstByte&0b11 == 0b01 { // Two bytes (14-bit value)
		return uint64(firstByte>>2) | uint64(data[1])<<6
	} else if firstByte&0b11 == 0b10 { // Four bytes (30-bit value)
		return uint64(firstByte>>2) | uint64(data[1])<<6 | uint64(data[2])<<14 | uint64(data[3])<<22
	} else { // Eight bytes (64-bit value)
		return uint64(data[1]) | uint64(data[2])<<8 | uint64(data[3])<<16 | uint64(data[4])<<24 |
			uint64(data[5])<<32 | uint64(data[6])<<40 | uint64(data[7])<<48
	}
}

// nolint
func encodeCompactU64(value uint64) []byte {
	var result []byte

	if value < (1 << 6) { // Single byte (6-bit value)
		result = []byte{byte(value<<2) | 0b00}
	} else if value < (1 << 14) { // Two bytes (14-bit value)
		result = []byte{
			byte((value&0x3F)<<2) | 0b01,
			byte(value >> 6),
		}
	} else if value < (1 << 30) { // Four bytes (30-bit value)
		result = []byte{
			byte((value&0x3F)<<2) | 0b10,
			byte(value >> 6),
			byte(value >> 14),
			byte(value >> 22),
		}
	} else { // Eight bytes (64-bit value)
		result = []byte{
			0b11, // First byte indicates 8-byte encoding
			byte(value),
			byte(value >> 8),
			byte(value >> 16),
			byte(value >> 24),
			byte(value >> 32),
			byte(value >> 40),
			byte(value >> 48),
			byte(value >> 56),
		}
	}

	return result
}

func padHexStringRight(hexStr string) string {
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	// Right pad with zeros to ensure it's 16 characters long (8 bytes)
	for len(hexStr) < 16 {
		hexStr += "0"
	}

	return hexStr
}

type ZzzHuiDecoder struct {
}

func (z ZzzHuiDecoder) DecodePaioBatch(bytes string) (string, error) {
	_, typedData, signature, err := commons.ExtractSigAndData(bytes)
	if err != nil {
		return "", err
	}
	signature[64] += 27
	slog.Debug("DecodePaioBatch", "signature", common.Bytes2Hex(signature))
	txs := []PaioTransaction{}
	txs = append(txs, PaioTransaction{
		Signature: PaioSignature{
			R: fmt.Sprintf("0x%s", common.Bytes2Hex(signature[0:32])),
			S: fmt.Sprintf("0x%s", common.Bytes2Hex(signature[32:64])),
			V: fmt.Sprintf("0x%s", common.Bytes2Hex(signature[64:])),
		},
		App:         typedData.Message["app"].(string),
		Nonce:       uint64(typedData.Message["nonce"].(float64)),
		Data:        common.Hex2Bytes(typedData.Message["data"].(string)[2:]),
		MaxGasPrice: uint64(typedData.Message["max_gas_price"].(float64)),
	})
	paioBatch := PaioBatch{
		Txs: txs,
	}
	paioJson, err := json.Marshal(paioBatch)
	if err != nil {
		return "", err
	}
	return string(paioJson), nil
}

func (av *AvailListener) ReadInputsFromPaioBlock(block *types.SignedBlock) ([]cModel.AdvanceInput, error) {
	inputs := []cModel.AdvanceInput{}
	timestamp, err := ReadTimestampFromBlock(block)
	if err != nil {
		return inputs, err
	}
	chainId, err := av.InputterWorker.ChainID()
	if err != nil {
		return inputs, err
	}
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()
		slog.Debug("debug", "appID", appID, "timestamp", timestamp)
		if appID != DEFAULT_APP_ID {
			// slog.Debug("Skipping", "appID", appID)
			continue
		}
		args := string(ext.Method.Args)
		jsonStr, err := av.PaioDecoder.DecodePaioBatch(args)
		if err != nil {
			return inputs, err
		}
		parsedInputs, err := ParsePaioBatchToInputs(jsonStr, chainId)
		if err != nil {
			return inputs, err
		}
		inputs = append(inputs, parsedInputs...)
	}
	for i := range inputs {
		inputs[i].BlockTimestamp = time.Unix(int64(timestamp), 0)
	}
	return inputs, nil
}

type PaioBatch struct {
	SequencerPaymentAddress string            `json:"sequencer_payment_address"`
	Txs                     []PaioTransaction `json:"txs"`
}

type PaioTransaction struct {
	App         string        `json:"app"`
	Nonce       uint64        `json:"nonce"`
	MaxGasPrice uint64        `json:"max_gas_price"`
	Data        []byte        `json:"data"`
	Signature   PaioSignature `json:"signature"`
}

type PaioSignature struct {
	R string `json:"r"`
	S string `json:"s"`
	V string `json:"v"`
}

func (ps *PaioSignature) Hex() string {
	return fmt.Sprintf("%s%s%s", ps.R, ps.S[2:], ps.V[2:])
}

func ParsePaioBatchToInputs(jsonStr string, chainId *big.Int) ([]cModel.AdvanceInput, error) {
	inputs := []cModel.AdvanceInput{}
	var paioBatch PaioBatch
	if err := json.Unmarshal([]byte(jsonStr), &paioBatch); err != nil {
		return inputs, fmt.Errorf("unmarshal paio batch: %w", err)
	}
	slog.Debug("PaioBatch", "tx len", len(paioBatch.Txs), "json", jsonStr)
	for _, tx := range paioBatch.Txs {
		slog.Debug("Tx",
			"app", tx.App,
			"signature", tx.Signature.Hex(),
		)
		typedData := paiodecoder.CreateTypedData(
			common.HexToAddress(tx.App),
			tx.Nonce,
			big.NewInt(int64(tx.MaxGasPrice)),
			tx.Data,
			chainId,
		)
		typeJSON, err := json.Marshal(typedData)
		if err != nil {
			return inputs, fmt.Errorf("error marshalling typed data: %w", err)
		}
		// set the typedData as string json below
		sigAndData := commons.SigAndData{
			Signature: tx.Signature.Hex(),
			TypedData: base64.StdEncoding.EncodeToString(typeJSON),
		}
		jsonPayload, err := json.Marshal(sigAndData)
		if err != nil {
			slog.Error("Error json.Marshal message:", "err", err)
			return inputs, err
		}
		slog.Debug("SaveTransaction", "jsonPayload", string(jsonPayload))
		msgSender, _, signature, err := commons.ExtractSigAndData(string(jsonPayload))
		if err != nil {
			slog.Error("Error ExtractSigAndData message:", "err", err)
			return inputs, err
		}
		txId := fmt.Sprintf("0x%s", common.Bytes2Hex(crypto.Keccak256(signature)))
		inputs = append(inputs, cModel.AdvanceInput{
			Index:               int(0),
			ID:                  txId,
			MsgSender:           msgSender,
			Payload:             tx.Data,
			AppContract:         common.HexToAddress(tx.App),
			AvailBlockNumber:    0,
			AvailBlockTimestamp: time.Unix(0, 0),
			InputBoxIndex:       -2,
			Type:                "Avail",
		})
	}
	return inputs, nil
}

func ReadInputsFromAvailBlockZzzHui(block *types.SignedBlock) ([]cModel.AdvanceInput, error) {
	inputs := []cModel.AdvanceInput{}
	timestamp, err := ReadTimestampFromBlock(block)
	if err != nil {
		return inputs, err
	}
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()
		slog.Debug("debug", "appID", appID, "timestamp", timestamp)
		if appID != DEFAULT_APP_ID {
			slog.Debug("Skipping", "appID", appID)
			continue
		}
		args := string(ext.Method.Args)

		msgSender, typedData, signature, err := commons.ExtractSigAndData(args)
		if err != nil {
			return inputs, err
		}
		paioMessage, err := ParsePaioFrom712Message(typedData)
		if err != nil {
			return inputs, err
		}
		slog.Debug("MsgSender", "value", msgSender)
		inputs = append(inputs, cModel.AdvanceInput{
			Index:                int(0),
			CartesiTransactionId: common.Bytes2Hex(crypto.Keccak256(signature)),
			MsgSender:            msgSender,
			Payload:              paioMessage.Payload,
			AppContract:          common.HexToAddress(paioMessage.App),
			AvailBlockNumber:     int(block.Block.Header.Number),
			AvailBlockTimestamp:  time.Unix(int64(timestamp)/ONE_SECOND_IN_MS, 0),
			InputBoxIndex:        -2,
			Type:                 "Avail",
		})
	}
	return inputs, nil
}

func ReadTimestampFromBlock(block *types.SignedBlock) (uint64, error) {
	timestampSectionIndex := uint8(TIMESTAMP_SECTION_INDEX)
	timestampMethodIndex := uint8(0)
	coreAppID := int64(0)
	for _, ext := range block.Block.Extrinsics {
		appID := ext.Signature.AppID.Int64()

		mi := ext.Method.CallIndex.MethodIndex
		si := ext.Method.CallIndex.SectionIndex

		if appID == coreAppID && si == uint8(timestampSectionIndex) && mi == uint8(timestampMethodIndex) {
			timestamp := DecodeTimestamp(common.Bytes2Hex(ext.Method.Args))
			return timestamp, nil
		}
	}
	return 0, fmt.Errorf("block %d without timestamp", block.Block.Header.Number)
}

func ParsePaioFrom712Message(typedData apitypes.TypedData) (PaioMessage, error) {
	message := PaioMessage{
		App:         typedData.Message["app"].(string),
		Nonce:       typedData.Message["nonce"].(string),
		MaxGasPrice: typedData.Message["max_gas_price"].(string),
		Payload:     []byte(typedData.Message["data"].(string)),
	}
	return message, nil
}

// alterar para usar o nome do Paio
type PaioMessage struct {
	App         string
	Nonce       string
	MaxGasPrice string
	Payload     []byte
}
