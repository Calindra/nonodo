package rollup

import (
	"math/big"
)

type ExpressoHeader struct {
	L1Finalized struct {
		Number *big.Int `json:"number"`
	} `json:"l1_finalized"`
}

type EspressoBlock struct {
	Header ExpressoHeader `json:"header"`
}

type BlockVM struct {
	Payload struct {
		TransactionNMT  []any `json:"transaction_nmt"`
	} `json:"payload"`
	Header struct {
		Height *big.Int `json:"height"`
	} `json:"header"`
}

func (b *EspressoBlock) filterByVM(vmId string) *BlockVM {
	return nil
}

type EspressoAPI interface {
	GetLatestBlockHeight() (*big.Int, error)
	GetHeaderByBlockByHeight(height *big.Int) (*ExpressoHeader, error)
	GetBlockByHeight(height *big.Int) (*EspressoBlock, error)
}

type ExpressoService struct{}

func (s *ExpressoService) GetLatestBlockHeight() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (s *ExpressoService) GetHeaderByBlockByHeight(height *big.Int) (*ExpressoHeader, error) {
	return &ExpressoHeader{}, nil
}

func (s *ExpressoService) GetBlockByHeight(height *big.Int) (*EspressoBlock, error) {
	return &EspressoBlock{}, nil
}
