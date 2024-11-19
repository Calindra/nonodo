package espresso

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	cModel "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/model"
	cRepos "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/repository"
	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/dataavailability"
	"github.com/calindra/nonodo/internal/sequencers/inputter"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tidwall/gjson"
)

type EspressoListener struct {
	espressoAPI     *dataavailability.EspressoAPI
	espressoUrl     string
	namespace       uint64
	InputRepository *cRepos.InputRepository
	fromBlock       uint64
	InputterWorker  *inputter.InputterWorker
	fromBlockL1     *uint64
}

func (e EspressoListener) String() string {
	return "espresso_listener"
}

func NewEspressoListener(
	espressoUrl string,
	namespace uint64,
	repository *cRepos.InputRepository,
	fromBlock uint64,
	w *inputter.InputterWorker,
	fromBlockL1 *uint64,
) *EspressoListener {
	return &EspressoListener{
		espressoUrl:     espressoUrl,
		namespace:       namespace,
		InputRepository: repository,
		fromBlock:       fromBlock,
		InputterWorker:  w,
		fromBlockL1:     fromBlockL1,
	}
}

func (e EspressoListener) getBaseUrl() string {
	return e.espressoUrl
}

func (e EspressoListener) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
	url := e.getBaseUrl()
	e.espressoAPI = dataavailability.NewEspressoAPI(ctx, &url)
	slog.Info("espresso listener started")
	return e.watchNewTransactions(ctx)
}

func (e EspressoListener) watchNewTransactions(ctx context.Context) error {
	slog.Debug("Espresso: watchNewTransactions", "fromBlock", e.fromBlock)
	currentBlockHeight := e.fromBlock
	if currentBlockHeight == 0 {
		lastEspressoBlockHeight, err := e.espressoAPI.FetchLatestBlockHeight(ctx)
		if err != nil {
			return err
		}
		currentBlockHeight = lastEspressoBlockHeight
		slog.Info("Espresso: starting from latest block height", "lastEspressoBlockHeight", lastEspressoBlockHeight)
	}
	previousBlockHeight := currentBlockHeight
	var l1FinalizedPrevHeight uint64
	if e.fromBlockL1 != nil {
		l1FinalizedPrevHeight = *e.fromBlockL1
	} else {
		l1FinalizedPrevHeight = e.getL1FinalizedHeight(previousBlockHeight)
	}
	slog.Info("Espresso: starting l1 block from", "blockNumber", l1FinalizedPrevHeight)

	var delay time.Duration = 800

	// main polling loop
	for {
		slog.Debug("Espresso: fetchLatestBlockHeight...")
		lastEspressoBlockHeight, err := e.espressoAPI.FetchLatestBlockHeight(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				slog.Warn("Espresso: the context was canceled. stopping operation.")
				return err
			}
			slog.Warn("Espresso: error fetching the latest block height. Retrying...", "error", err)
			time.Sleep(delay * time.Millisecond)
			continue
		}
		slog.Debug("Espresso:", "lastEspressoBlockHeight", lastEspressoBlockHeight)

		if lastEspressoBlockHeight == currentBlockHeight {
			time.Sleep(delay * time.Millisecond)
			continue
		}
		for ; currentBlockHeight < lastEspressoBlockHeight; currentBlockHeight++ {
			slog.Debug("Espresso:", "currentBlockHeight", currentBlockHeight, "namespace", e.namespace)
			transactions, err := e.espressoAPI.FetchTransactionsInBlock(ctx, currentBlockHeight, e.namespace)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					slog.Warn("Espresso: the context was canceled. stopping operation.")
					return err
				}
				slog.Warn("Espresso: error fetching transactions. Retrying...", "blockHeight", currentBlockHeight, "namespace", e.namespace, "error", err)
				time.Sleep(delay * time.Millisecond)
				continue
			}
			tot := len(transactions.Transactions)

			// read inputbox
			l1FinalizedCurrentHeight := e.getL1FinalizedHeight(currentBlockHeight)
			// read L1 if there might be update
			if l1FinalizedCurrentHeight > l1FinalizedPrevHeight || currentBlockHeight == e.fromBlock {
				slog.Debug("L1 finalized", "from", l1FinalizedPrevHeight, "to", l1FinalizedCurrentHeight)
				slog.Debug("Fetching InputBox between Espresso blocks", "from", previousBlockHeight, "to", currentBlockHeight)
				err = readInputBox(ctx, l1FinalizedPrevHeight, l1FinalizedCurrentHeight, e.InputterWorker)
				if err != nil {
					return err
				}
				l1FinalizedPrevHeight = l1FinalizedCurrentHeight
			}

			slog.Debug("Espresso:", "transactionsLen", tot)
			for i := 0; i < tot; i++ {
				transaction := transactions.Transactions[i]
				slog.Debug("Espresso:", "currentBlockHeight", currentBlockHeight)

				ctx := context.Background()
				// transform and add to InputRepository
				index, err := e.InputRepository.Count(ctx, nil)
				if err != nil {
					return err
				}

				// assume the following encoding
				// transaction = JSON.stringify({
				//		 	signature,
				//		 	typedData: btoa(JSON.stringify(typedData)),
				//		 })
				msgSender, typedData, signature, err := commons.ExtractSigAndData(string(transaction))
				if err != nil {
					slog.Warn("Ignoring transaction", "blockHeight", currentBlockHeight, "transactionIndex", i, "error", err)
					continue
				}
				// type EspressoMessage struct {
				// 	nonce   uint32 `json:"nonce"`
				// 	payload string `json:"payload"`
				// }
				nonceField := typedData.Message["nonce"]

				var nonce int64
				// Request tx is coming from nonodo, the nonce at this point returns as string always
				if _, ok := nonceField.(string); ok {
					strNonce := nonceField.(string)
					parsedNonce, err := strconv.ParseInt(strNonce, 10, 64)
					if err != nil {
						return fmt.Errorf("error converting nonce from string to int64: %v", err)
					}
					nonce = parsedNonce
				} else if _, ok := nonceField.(float64); ok {
					// When coming from other sources, the nonce at this point is a float
					nonce = int64(nonceField.(float64))
				} else {
					//
					return fmt.Errorf("error converting nonce: %T", nonceField)
				}

				app, ok := typedData.Message["app"].(string)
				if !ok {
					return fmt.Errorf("message app address type assertion error")
				}
				appContract := common.HexToAddress(app)

				if appContract.Hex() != e.InputterWorker.ApplicationAddress.Hex() {
					slog.Debug("Espresso: ignoring transaction for other app", "txApp", appContract.Hex(), "expectedApp", e.InputterWorker.ApplicationAddress.Hex())
					continue
				}

				payload, ok := typedData.Message["data"].(string)
				if !ok {
					return fmt.Errorf("message data type assertion error")
				}
				slog.Debug("Espresso input", "msgSender", msgSender, "nonce", nonce, "payload", payload)

				// update nonce maps
				// no need to consider node exits abruptly and restarts from where it left
				// app has to start `nonce` from 1 and increment by 1 for each payload
				dbNonce, err := e.InputRepository.GetNonce(ctx, appContract, msgSender)
				if err != nil {
					slog.Error("calculate internal nonce error",
						"appContract", appContract,
						"msgSender", msgSender,
						"error", err,
					)
					return err
				}
				if int64(dbNonce) != nonce {
					slog.Warn("duplicated or incorrect nonce",
						"nonce", nonce,
						"expected", dbNonce,
						"appContract", appContract,
						"msgSender", msgSender,
					)
					continue
				}

				payload = strings.TrimPrefix(payload, "0x")

				chainId := (*big.Int)(typedData.Domain.ChainId).String()
				slog.Debug("TypedData", "typedData.Domain", typedData.Domain,
					"chainId", chainId,
				)
				blockNumber := e.getL1FinalizedHeight(currentBlockHeight)
				prevRandao, err := readPrevRandao(ctx, currentBlockHeight, e.InputterWorker)
				if err != nil {
					return err
				}
				_, err = e.InputRepository.Create(ctx, cModel.AdvanceInput{
					ID:                     common.Bytes2Hex(crypto.Keccak256(signature)),
					Index:                  int(index),
					MsgSender:              msgSender,
					Payload:                payload,
					BlockNumber:            blockNumber,
					BlockTimestamp:         e.getL1FinalizedTimestamp(currentBlockHeight),
					AppContract:            e.InputterWorker.ApplicationAddress,
					EspressoBlockNumber:    int(currentBlockHeight),
					EspressoBlockTimestamp: e.getEspressoTimestamp(currentBlockHeight),
					InputBoxIndex:          -1,
					Type:                   "Espresso",
					ChainId:                chainId,
					PrevRandao:             prevRandao,
				})
				slog.Info("Espresso: input added", "payload", payload)
				if err != nil {
					return err
				}
			}
		}
	}
}

func (e EspressoListener) readEspressoHeader(espressoBlockHeight uint64) string {
	requestURL := fmt.Sprintf("%s/availability/header/%d", e.espressoUrl, espressoBlockHeight)
	res, err := http.Get(requestURL)
	if err != nil {
		slog.Error("error making http request", "err", err)
		os.Exit(1)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("could not read response body", "err", err)
		os.Exit(1)
	}

	return string(resBody)
}

func (e EspressoListener) getL1FinalizedTimestamp(espressoBlockHeight uint64) time.Time {
	espressoHeader := e.readEspressoHeader(espressoBlockHeight)
	value := gjson.Get(espressoHeader, "fields.l1_finalized.timestamp")
	timestampStr := value.Str
	timestampInt, err := strconv.ParseInt(timestampStr[2:], 16, 64)
	if err != nil {
		slog.Error("hex to int conversion failed", "err", err)
		os.Exit(1)
	}
	return time.Unix(timestampInt, 0)
}

func (e EspressoListener) getL1FinalizedHeight(espressoBlockHeight uint64) uint64 {
	espressoHeader := e.readEspressoHeader(espressoBlockHeight)
	value := gjson.Get(espressoHeader, "fields.l1_finalized.number")
	return value.Uint()
}

func (e EspressoListener) getEspressoTimestamp(espressoBlockHeight uint64) time.Time {
	espressoHeader := e.readEspressoHeader(espressoBlockHeight)
	value := gjson.Get(espressoHeader, "timestamp")
	return time.Unix(value.Int(), 0)
}

func readInputBox(ctx context.Context, l1FinalizedPrevHeight uint64, l1FinalizedCurrentHeight uint64, w *inputter.InputterWorker) error {
	client, err := ethclient.DialContext(ctx, w.Provider)
	if err != nil {
		return fmt.Errorf("espresso inputter: dial: %w", err)
	}
	inputBox, err := contracts.NewInputBox(w.InputBoxAddress, client)
	if err != nil {
		return fmt.Errorf("espresso inputter: bind input box: %w", err)
	}

	err = w.ReadPastInputs(ctx, client, inputBox, l1FinalizedPrevHeight, &l1FinalizedCurrentHeight)
	if err != nil {
		return fmt.Errorf("espresso inputter: read past inputs: %w", err)
	}

	return nil
}

func readPrevRandao(ctx context.Context, l1FinalizedCurrentHeight uint64, w *inputter.InputterWorker) (string, error) {
	client, err := w.GetEthClient()
	if err != nil {
		return "", fmt.Errorf("espresso eth client error: %w", err)
	}
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(l1FinalizedCurrentHeight)))
	if err != nil {
		return "", fmt.Errorf("espresso read block header error: %w", err)
	}
	prevRandao := header.MixDigest.Hex()
	slog.Debug("readPrevRandao", "prevRandao", prevRandao, "blockNumber", l1FinalizedCurrentHeight)
	return prevRandao, nil
}
