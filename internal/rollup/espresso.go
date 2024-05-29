package rollup

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/labstack/echo/v4"
)

var (
	EPOCH_DURATION   = getEpochDuration()
	VM_ID            = devnet.ApplicationAddress[0:18]
	INPUT_FETCH_SIZE = 130
)

type FetchInputBoxContext struct {
	blockNumber             big.Int
	epoch                   big.Int
	currentInput            big.Int
	currentInputBlockNumber big.Int
	currentEpoch            big.Int
}

func computeEpoch(blockNumber *big.Int) (*big.Int, error) {
	// try to mimic current Authority epoch computation
	if EPOCH_DURATION == nil {
		return nil, fmt.Errorf("invalid epoch duration")
	} else {
		result := new(big.Int).Div(blockNumber, EPOCH_DURATION)
		return result, nil
	}
}

func (r *RollupAPI) fetchCurrentInput() (*model.AdvanceInput, error) {
	// retrieve total number of inputs
	input := r.model.GetInputRepository()
	currentInput, err := input.FindByStatusNeDesc(model.CompletionStatusUnprocessed)
	if err != nil {
		return nil, err
	}

	return currentInput, nil
}

func getEpochDuration() *big.Int {
	EPOCH_DURATION := os.Getenv("EPOCH_DURATION")
	var epochDuration *big.Int
	if EPOCH_DURATION != "" {
		i, err := strconv.ParseInt(EPOCH_DURATION, 10, 64)
		if err != nil {
			panic(err)
		}
		epochDuration = big.NewInt(i)
	} else {
		oneDay := 86400
		epochDuration = big.NewInt(int64(oneDay))
	}

	return epochDuration
}

func (r *RollupAPI) fetchContext(blockNumber *big.Int) (*FetchInputBoxContext, error) {
	currentInput, err := r.fetchCurrentInput()
	currentInputIndex := big.NewInt(0).SetInt64(int64(currentInput.Index))

	if err != nil {
		return nil, err
	}

	currentInputBlockNumber := big.NewInt(0).SetInt64(int64(currentInput.BlockNumber))

	currentEpoch, err := computeEpoch(currentInputBlockNumber)
	if err != nil {
		return nil, err
	}
	epoch, err := computeEpoch(blockNumber)
	if err != nil {
		return nil, err
	}

	if epoch.Cmp(currentEpoch) != 1 {
		err := fmt.Sprintf(
			"Requested data beyond current epoch '%s'"+
				" (data estimated to belong to epoch '%s')",
			currentEpoch.String(),
			epoch.String(),
		)
		slog.Error(err)
		return nil, fmt.Errorf(err)
	}

	var context FetchInputBoxContext = FetchInputBoxContext{
		blockNumber:             *blockNumber,
		epoch:                   *epoch,
		currentInput:            *currentInputIndex,
		currentInputBlockNumber: *currentInputBlockNumber,
		currentEpoch:            *currentEpoch,
	}

	return &context, nil
}

func (r *RollupAPI) FetchEspresso(ctx echo.Context, id string) (*string, *model.HttpCustomError) {
	// check if id is valid and parse id as maxBlockNumber and espressoBlockHeight
	if len(id) != INPUT_FETCH_SIZE || id[:2] != "0x" {
		err := fmt.Sprintf("Invalid id %s: : must be a hex string with 32 bytes for maxBlockNumber and 32 bytes for espressoBlockHeight", id)
		slog.Error(err)
		return nil, model.NewHttpCustomError(http.StatusBadRequest, nil)
	}
	maxBlockNumber := big.NewInt(0).SetBytes([]byte(id[2:66]))
	espressoBlockHeight := big.NewInt(0).SetBytes([]byte(id[66:130]))

	context, err := r.fetchContext(maxBlockNumber)

	if err != nil {
		return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)
	}

	// check if out of epoch's scope
	if context.epoch.Cmp(&context.currentEpoch) == 1 {
		error := fmt.Sprintf(
			"Requested data beyond current epoch '%s'"+
				" (data estimated to belong to epoch '%s')",
			context.currentEpoch.String(),
			context.epoch.String(),
		)
		slog.Error(error)
		return nil, model.NewHttpCustomError(http.StatusForbidden, nil)
	}

	ctxHttp := ctx.Request().Context()
	urlBase := "https://query.cappuccino.testnet.espresso.network/"
	espressoService := NewExpressoService(ctxHttp, &urlBase)

	for {
		lastEspressoBlockHeight, err := espressoService.GetLatestBlockHeight()
		if err != nil {
			msg := fmt.Sprintf("Failed to get latest block height: %s", err)
			slog.Error(msg)
			return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

		}
		if espressoBlockHeight.Cmp(lastEspressoBlockHeight) == 1 {
			// requested Espresso block not available yet: just check if we are still within L1 blockNumber scope
			header, err := espressoService.GetHeaderByBlockByHeight(lastEspressoBlockHeight)
			if err != nil {
				msg := fmt.Sprintf("Failed to get header by block height: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			l1FinalizedNumber := header.L1Finalized.Number
			l1Finalized := big.NewInt(0).SetUint64(l1FinalizedNumber)
			if l1Finalized.Cmp(maxBlockNumber) == 1 {
				msg := fmt.Sprintf("Espresso block height %s is not finalized", espressoBlockHeight)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			// call again at some later time to see if we reach the block
			var timeInMs time.Duration = 500
			time.Sleep(timeInMs * time.Millisecond)
		} else {
			// requested Espresso block available: fetch it
			filteredBlock, err := espressoService.GetTransactionByHeight(espressoBlockHeight)
			if err != nil {
				msg := fmt.Sprintf("Failed to get block by height: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			header, err := espressoService.GetHeaderByBlockByHeight(espressoBlockHeight)

			if err != nil {
				msg := fmt.Sprintf("Failed to get header by block height: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}

			// check if within L1 blockNumber scope
			l1FinalizedNumber := header.L1Finalized.Number
			l1Finalized := big.NewInt(0).SetUint64(l1FinalizedNumber)
			if l1Finalized == nil {
				msg := fmt.Sprintf("Espresso block %s with undefined L1 blockNumber", espressoBlockHeight)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusNotFound, nil)
			}

			if l1Finalized.Cmp(maxBlockNumber) == 1 {
				msg := fmt.Sprintf("Espresso block height %s beyond requested L1 blockNumber", espressoBlockHeight)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusNotFound, nil)
			}

			serializedBlock, err := json.Marshal(filteredBlock)
			if err != nil {
				msg := fmt.Sprintf("Failed to marshal block: %s", err)
				slog.Error(msg)
				return nil, model.NewHttpCustomError(http.StatusInternalServerError, nil)

			}
			encodedBlockHex := hexutil.Encode(serializedBlock)
			// nTransactions := len(blockFiltered.Payload.TransactionNMT)
			nTransactions := len(filteredBlock.Transactions)
			slog.Info(fmt.Sprintf("Fetched Espresso block %s with %d transactions", espressoBlockHeight, nTransactions))
			return &encodedBlockHex, nil
		}
	}
}
