package rollup

import (
	"math/big"
)

type EspressoHeader struct {
	Height *big.Int `json:"height"`

	L1Finalized *struct {
		Number *big.Int `json:"number"`
	} `json:"l1_finalized"`
}

type EspressoTransactionNMT struct {
	VM string `json:"vm"`
}

type EspressoPayload struct {
	TransactionNMT []EspressoTransactionNMT `json:"transaction_nmt"`
}

type EspressoBlock struct {
	Header  EspressoHeader  `json:"header"`
	Payload EspressoPayload `json:"payload"`
	Hash    string          `json:"hash"`
}


func (b *EspressoBlock) filterByVM(vmId string) EspressoBlock {
	filteredTransactions := []EspressoTransactionNMT{}
	for _, transaction := range b.Payload.TransactionNMT {
		if transaction.VM == vmId {
			filteredTransactions = append(filteredTransactions, transaction)
		}
	}

	return EspressoBlock{
		Header:  b.Header,
		Payload: EspressoPayload{TransactionNMT: filteredTransactions},
		Hash:    b.Hash,
	}
}

type EspressoAPI interface {
	GetLatestBlockHeight() (*big.Int, error)
	GetHeaderByBlockByHeight(height *big.Int) (*EspressoHeader, error)
	GetBlockByHeight(height *big.Int) (*EspressoBlock, error)
}

type ExpressoService struct{}

func (s *ExpressoService) GetLatestBlockHeight() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (s *ExpressoService) GetHeaderByBlockByHeight(height *big.Int) (*EspressoHeader, error) {
	return &EspressoHeader{}, nil
}

func (s *ExpressoService) GetBlockByHeight(height *big.Int) (*EspressoBlock, error) {
	return &EspressoBlock{}, nil
}
