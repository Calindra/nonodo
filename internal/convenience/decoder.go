package convenience

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type OutputDecoder struct {
	convenienceService ConvenienceService
}

type ConvenienceService interface {
	CreateVoucher(ctx context.Context, voucher *ConvenienceVoucher)
	CreateNotice()
}

func (o *OutputDecoder) HandleOutput(ctx context.Context, destination common.Address, payload string, inputIndex uint64, outputIndex uint64) {
	// TODO detect the output type
	o.convenienceService.CreateVoucher(ctx, &ConvenienceVoucher{
		Destination: destination,
		Payload:     payload,
		Executed:    false,
		InputIndex:  inputIndex,
		OutputIndex: outputIndex,
	})
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
