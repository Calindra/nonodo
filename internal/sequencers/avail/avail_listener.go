package avail

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/calindra/nonodo/internal/sequencers/paiodecoder"
	"github.com/calindra/nonodo/internal/supervisor"
	"github.com/cartesi/rollups-graphql/pkg/commons"
	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	cRepos "github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	TIMESTAMP_SECTION_INDEX = 3
	DELAY                   = 500
	ONE_SECOND_IN_MS        = 1000
	FIVE_MINUTES            = 300
)

type AvailListener struct {
	PaioDecoder        paiodecoder.DecoderPaio
	InputRepository    *cRepos.InputRepository
	InputterWorker     *inputter.InputterWorker
	FromBlock          uint64
	AvailFromBlock     uint64
	L1CurrentBlock     uint64
	ApplicationAddress common.Address
	L1ReadDelay        int
}

type PaioDecoder interface {
	DecodePaioBatch(ctx context.Context, bytes []byte) (string, error)
}

func NewAvailListener(availFromBlock uint64, repository *cRepos.InputRepository,
	w *inputter.InputterWorker, fromBlock uint64, binaryDecoderPathLocation string,
	applicationAddress string,
) supervisor.Worker {
	var paioDecoder PaioDecoder = paiodecoder.ZzzzHuiDecoder{}
	if binaryDecoderPathLocation != "" {
		paioDecoder = paiodecoder.NewPaioDecoder(binaryDecoderPathLocation)
	}
	l1ReadDelay := FIVE_MINUTES
	l1ReadDelayStr, ok := os.LookupEnv("L1_READ_DELAY_IN_SECONDS")
	if ok {
		aux, err := strconv.Atoi(l1ReadDelayStr)
		if err != nil {
			slog.Error("Configuration error: The L1_READ_DELAY_IN_SECONDS environment variable should be a numeric value.")
			panic(err)
		}
		l1ReadDelay = aux
	}
	return AvailListener{
		AvailFromBlock:     availFromBlock,
		InputRepository:    repository,
		InputterWorker:     w,
		PaioDecoder:        paioDecoder,
		L1CurrentBlock:     fromBlock,
		ApplicationAddress: common.HexToAddress(applicationAddress),
		L1ReadDelay:        l1ReadDelay,
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
	availInputs, err := a.ReadInputsFromPaioBlock(ctx, block)
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
		(availBlockTimestamp/ONE_SECOND_IN_MS)-uint64(a.L1ReadDelay),
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
			if inputs[i].AppContract != a.ApplicationAddress {
				slog.Warn("Skipping input",
					"appContract", inputs[i].AppContract.Hex(),
					"expected", a.ApplicationAddress.Hex(),
					"msgSender", inputs[i].MsgSender.Hex(),
					"payload", inputs[i].Payload,
				)
				continue
			}
			// The chainId information does not come in Paio's batch.
			_, err = a.InputRepository.Create(ctx, inputs[i])
			if err != nil {
				return nil, err
			}
			slog.Info("Input saved",
				"ID", inputs[i].ID,
				"index", inputs[i].Index,
				"appContract", inputs[i].AppContract.Hex(),
				"msgSender", inputs[i].MsgSender.Hex(),
				"payload", inputs[i].Payload,
			)
		}
	}
	return &currentL1Block, nil
}

func (av *AvailListener) ReadInputsFromPaioBlock(ctx context.Context, block *types.SignedBlock) ([]cModel.AdvanceInput, error) {
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
		args := ext.Method.Args
		jsonStr, err := av.PaioDecoder.DecodePaioBatch(ctx, args)
		if err != nil {
			return inputs, err
		}
		parsedInputs, err := paiodecoder.ParsePaioBatchToInputs(jsonStr, chainId)
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
		paioMessage, err := paiodecoder.ParsePaioFrom712Message(typedData)
		if err != nil {
			return inputs, err
		}
		slog.Debug("MsgSender", "value", msgSender)
		inputs = append(inputs, cModel.AdvanceInput{
			Index:                int(0),
			CartesiTransactionId: common.Bytes2Hex(crypto.Keccak256(signature)),
			MsgSender:            msgSender,
			Payload:              common.Bytes2Hex(paioMessage.Payload),
			AppContract:          common.HexToAddress(paioMessage.App),
			AvailBlockNumber:     int(block.Block.Header.Number),
			AvailBlockTimestamp:  time.Unix(int64(timestamp)/ONE_SECOND_IN_MS, 0),
			InputBoxIndex:        -2,
			Type:                 "Avail",
		})
	}
	return inputs, nil
}
