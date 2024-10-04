package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/convenience/adapter"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type OutputDecoder struct {
	convenienceService services.ConvenienceService
}

func NewOutputDecoder(convenienceService services.ConvenienceService) *OutputDecoder {
	return &OutputDecoder{
		convenienceService: convenienceService,
	}
}

// Deprecated: HandleOutput is deprecated.
func (o *OutputDecoder) HandleOutput(
	ctx context.Context,
	destination common.Address,
	payload string,
	inputIndex uint64,
	outputIndex uint64,
) error {
	// https://github.com/cartesi/rollups-contracts/issues/42#issuecomment-1694932058
	// detect the output type Voucher | Notice
	// 0xc258d6e5 for Notice
	// 0xef615e2f for Vouchers
	if payload[2:10] == model.VOUCHER_SELECTOR {
		_, err := o.convenienceService.CreateVoucher(ctx, &model.ConvenienceVoucher{
			Destination: destination,
			Payload:     adapter.RemoveSelector(payload),
			Executed:    false,
			InputIndex:  inputIndex,
			OutputIndex: outputIndex,
		})
		return err
	} else {
		_, err := o.convenienceService.CreateNotice(ctx, &model.ConvenienceNotice{
			Payload:     adapter.RemoveSelector(payload),
			InputIndex:  inputIndex,
			OutputIndex: outputIndex,
		})
		return err
	}
}

func (o *OutputDecoder) HandleOutputV2(
	ctx context.Context,
	processOutputData model.ProcessOutputData,
) error {
	// https://github.com/cartesi/rollups-contracts/issues/42#issuecomment-1694932058
	// detect the output type Voucher | Notice
	// 0xc258d6e5 for Notice
	// 0xef615e2f for Vouchers
	slog.Debug("Add Voucher/Notices",
		"inputIndex", processOutputData.InputIndex,
		"outputIndex", processOutputData.OutputIndex,
	)
	if processOutputData.Payload[2:10] == model.VOUCHER_SELECTOR {
		destination, err := o.RetrieveDestination(processOutputData.Payload)
		if err != nil {
			slog.Error("Failed to retrieve destination for node blob ", "err", err)
			return fmt.Errorf("error retrieving destination for node blob '%s': %w", processOutputData.Payload, err)
		}

		_, err = o.convenienceService.CreateVoucher(ctx, &model.ConvenienceVoucher{
			Destination: destination,
			Payload:     adapter.RemoveSelector(processOutputData.Payload),
			Executed:    false,
			InputIndex:  processOutputData.InputIndex,
			OutputIndex: processOutputData.OutputIndex,
		})
		return err
	} else {
		_, err := o.convenienceService.CreateNotice(ctx, &model.ConvenienceNotice{
			Payload:     adapter.RemoveSelector(processOutputData.Payload),
			InputIndex:  processOutputData.InputIndex,
			OutputIndex: processOutputData.OutputIndex,
		})
		return err
	}
}

func (o *OutputDecoder) HandleInput(
	ctx context.Context,
	input model.InputEdge,
	status model.CompletionStatus,
) error {
	convertedInput, err := o.GetConvertedInput(input)

	if err != nil {
		slog.Error("Failed to get converted:", "err", err)
		return fmt.Errorf("error getting converted input: %w", err)
	}
	_, err = o.convenienceService.CreateInput(ctx, &model.AdvanceInput{
		ID:                     strconv.Itoa(input.Node.Index),
		Index:                  input.Node.Index,
		Status:                 status,
		MsgSender:              convertedInput.MsgSender,
		Payload:                []byte(convertedInput.Payload),
		BlockNumber:            convertedInput.BlockNumber.Uint64(),
		BlockTimestamp:         time.Unix(convertedInput.BlockTimestamp, 0),
		PrevRandao:             convertedInput.PrevRandao,
		AppContract:            convertedInput.AppContract,
		EspressoBlockNumber:    -1,
		EspressoBlockTimestamp: time.Unix(-1, 0),
		InputBoxIndex:          int(convertedInput.InputBoxIndex),
		AvailBlockNumber:       -1,
		AvailBlockTimestamp:    time.Unix(-1, 0),
		CartesiTransactionId:   "0",
	})
	return err
}

func (o *OutputDecoder) HandleReport(
	ctx context.Context,
	index int,
	outputIndex int,
	payload string,
) error {
	_, err := o.convenienceService.CreateReport(ctx, &model.Report{
		Index:      outputIndex,
		InputIndex: index,
		Payload:    []byte(payload),
	})
	return err
}

func (o *OutputDecoder) GetAbi(address common.Address) (*abi.ABI, error) {
	baseURL := "https://api.etherscan.io/api"
	contextPath := "?module=contract&action=getsourcecode&address="
	url := fmt.Sprintf("%s/%s%s", baseURL, contextPath, address.String())

	var apiResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			ABI string `json:"ABI"`
		} `json:"result"`
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("unexpected error")
	}
	defer resp.Body.Close()
	apiResult, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unexpected error io")
	}
	if err := json.Unmarshal(apiResult, &apiResponse); err != nil {
		return nil, fmt.Errorf("unexpected error")
	}
	abiJSON := apiResponse.Result[0].ABI
	var abiData abi.ABI
	err2 := json.Unmarshal([]byte(abiJSON), &abiData)
	if err2 != nil {
		return nil, fmt.Errorf("unexpected error json %s", err2.Error())
	}
	return &abiData, nil
}

func jsonToAbi(abiJSON string) (*abi.ABI, error) {
	var abiData abi.ABI
	err2 := json.Unmarshal([]byte(abiJSON), &abiData)
	if err2 != nil {
		return nil, fmt.Errorf("unexpected error json %s", err2.Error())
	}
	return &abiData, nil
}

func (o *OutputDecoder) GetConvertedInput(input model.InputEdge) (model.ConvertedInput, error) {
	payload := input.Node.Blob
	var emptyConvertedInput model.ConvertedInput
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return emptyConvertedInput, err
	}

	values, err := abiParsed.Methods["EvmAdvance"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return emptyConvertedInput, err
	}
	convertedInput := model.ConvertedInput{
		MsgSender:      values[2].(common.Address),
		Payload:        string(values[7].([]uint8)),
		BlockNumber:    values[3].(*big.Int),
		BlockTimestamp: values[4].(*big.Int).Int64(),
		PrevRandao:     values[5].(*big.Int).String(),
		AppContract:    values[1].(common.Address),
		InputBoxIndex:  values[6].(*big.Int).Int64(),
	}

	return convertedInput, nil
}

func (o *OutputDecoder) RetrieveDestination(payload string) (common.Address, error) {
	abiParsed, err := contracts.OutputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return common.Address{}, err
	}

	slog.Info("payload", "payload", payload)

	values, err := abiParsed.Methods["Voucher"].Inputs.Unpack(common.Hex2Bytes(payload[10:]))

	if err != nil {
		slog.Error("Error unpacking abi", "err", err)
		return common.Address{}, err
	}

	return values[0].(common.Address), nil
}
