package reader

import (
	"encoding/json"
	graphql "github.com/calindra/nonodo/internal/reader/model"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"log/slog"
	"strings"
)

type InputBlobAdapter struct {
	abiParsed abi.ABI
}

func NewInputBlobAdapter() (*InputBlobAdapter, error) {
	dec := json.NewDecoder(strings.NewReader(abiJSON))

	var abiParsed abi.ABI
	if err := dec.Decode(&abiParsed); err != nil {
		return nil, err
	}

	return &InputBlobAdapter{abiParsed}, nil
}

const abiJSON = `
[
    {
        "inputs": [
            {
                "internalType": "uint256",
                "name": "chainId",
                "type": "uint256"
            },
            {
                "internalType": "address",
                "name": "appContract",
                "type": "address"
            },
            {
                "internalType": "address",
                "name": "msgSender",
                "type": "address"
            },
            {
                "internalType": "uint256",
                "name": "blockNumber",
                "type": "uint256"
            },
            {
                "internalType": "uint256",
                "name": "blockTimestamp",
                "type": "uint256"
            },
            {
                "internalType": "uint256",
                "name": "prevRandao",
                "type": "uint256"
            },
            {
                "internalType": "uint256",
                "name": "index",
                "type": "uint256"
            },
            {
                "internalType": "bytes",
                "name": "payload",
                "type": "bytes"
            }
        ],
        "name": "EvmAdvance",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    }
]
`

func (i *InputBlobAdapter) Adapt(node struct {
	Index  int    `json:"index"`
	Blob   string `json:"blob"`
	Status string `json:"status"`
}) (*graphql.Input, error) {
	unpacked, err := i.abiParsed.Unpack("EvmAdvance", common.Hex2Bytes(node.Blob))

	if err != nil {
		slog.Error("Error unpacking blob.", "err", err)
		return nil, err
	}

	return &graphql.Input{
		Index:       node.Index,
		Status:      convertCompletionStatus(node.Status),
		MsgSender:   unpacked[2].(common.Address).Hex(),
		Timestamp:   unpacked[4].(string),
		BlockNumber: unpacked[3].(string),
		Payload:     unpacked[7].(string),
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
