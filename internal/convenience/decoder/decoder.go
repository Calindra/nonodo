package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/calindra/nonodo/internal/convenience/adapter"
	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
)

type OutputDecoder struct {
	convenienceService services.ConvenienceService
}

func NewOutputDecoder(convenienceService services.ConvenienceService) *OutputDecoder {
	return &OutputDecoder{
		convenienceService: convenienceService,
	}
}

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
			Destination: destination,
			Payload:     adapter.RemoveSelector(payload),
			InputIndex:  inputIndex,
			OutputIndex: outputIndex,
		})
		return err
	}
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
