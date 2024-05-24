package reader

import (
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

	return &graphql.Input{
		Index:       node.Index,
		Status:      convertCompletionStatus(node.Status),
		MsgSender:   values[2].(common.Address).Hex(),
		Timestamp:   values[4].(*big.Int).String(),
		BlockNumber: values[3].(*big.Int).String(),
		Payload:     string(values[7].([]uint8)),
	}, nil
}

func convertCompletionStatus(status string) graphql.CompletionStatus {
	switch status {
	case graphql.CompletionStatusUnprocessed.String():
		return graphql.CompletionStatusUnprocessed
	case graphql.CompletionStatusAccepted.String():
		return graphql.CompletionStatusAccepted
	case graphql.CompletionStatusRejected.String():
		return graphql.CompletionStatusRejected
	case graphql.CompletionStatusException.String():
		return graphql.CompletionStatusException
	default:
		panic("invalid completion status")
	}
}
