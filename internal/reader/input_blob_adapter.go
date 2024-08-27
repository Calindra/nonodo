package reader

import (
	"fmt"
	"log/slog"
	"math/big"

	"github.com/calindra/nonodo/internal/contracts"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/common"
)

type InputBlobAdapter struct{}

func (i *InputBlobAdapter) Adapt(node struct {
	Index  int    `json:"index"`
	Blob   string `json:"blob"`
	Status string `json:"status"`
}) (*graphql.Input, error) {
	abiParsed, err := contracts.InputsMetaData.GetAbi()

	if err != nil {
		slog.Error("Error parsing abi", "err", err)
		return nil, err
	}

	values, err := abiParsed.Methods["EvmAdvance"].Inputs.UnpackValues(common.Hex2Bytes(node.Blob[10:]))

	if err != nil {
		slog.Error("Error unpacking blob.", "err", err)
		return nil, err
	}

	convertedStatus, err := convertCompletionStatus(node.Status)

	if err != nil {
		slog.Error("Error converting CompletionStatus.", "err", err)
		return nil, err
	}

	return &graphql.Input{
		Index:         node.Index,
		Status:        convertedStatus,
		MsgSender:     values[2].(common.Address).Hex(),
		Timestamp:     values[4].(*big.Int).String(),
		BlockNumber:   values[3].(*big.Int).String(),
		Payload:       string(values[7].([]uint8)),
		InputBoxIndex: values[6].(*big.Int).Int64(),
	}, nil
}

func convertCompletionStatus(status string) (graphql.CompletionStatus, error) {
	switch status {
	case graphql.CompletionStatusUnprocessed.String():
		return graphql.CompletionStatusUnprocessed, nil
	case graphql.CompletionStatusAccepted.String():
		return graphql.CompletionStatusAccepted, nil
	case graphql.CompletionStatusRejected.String():
		return graphql.CompletionStatusRejected, nil
	case graphql.CompletionStatusException.String():
		return graphql.CompletionStatusException, nil
	default:
		return "", fmt.Errorf(`Error converting CompletionStatus to valid value. Status to be converted %s`, status)
	}
}
