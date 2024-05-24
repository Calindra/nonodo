package rollup

import (
	"context"
	"math/big"
	"strconv"

	"github.com/EspressoSystems/espresso-sequencer-go/client"
	"github.com/EspressoSystems/espresso-sequencer-go/types"
)

type EspressoHeader = types.Header
type EspressoBlockResponse = client.TransactionsInBlock

type EspressoAPI interface {
	GetLatestBlockHeight() (*big.Int, error)
	GetHeaderByBlockByHeight(height *big.Int) (*EspressoHeader, error)
	GetBlockByHeight(height *big.Int) (*EspressoBlockResponse, error)
}

type ExpressoService struct {
	context context.Context
	client  *client.Client
}

func NewExpressoService(ctx context.Context, url *string) *ExpressoService {
	var myClient *client.Client

	if url != nil {
		myClient = client.NewClient(*url)
	}

	return &ExpressoService{
		context: ctx,
		client:  myClient,
	}
}

/**
 * The last known block height of the chain.
 * GET /status/block_height
 * https://docs.espressosys.com/sequencer/api-reference/sequencer-api/status-api#get-status-block-height
 * returns integer
 */
func (s *ExpressoService) GetLatestBlockHeight() (*big.Int, error) {
	// This is a mock implementation
	if s.client == nil {
		mock := 32644
		return big.NewInt(int64(mock)), nil
	}

	res, err := s.client.FetchLatestBlockHeight(s.context)
	if err != nil {
		return nil, err
	}

	value := big.NewInt(0).SetUint64(res)

	return value, nil

}

/**
 * GET /availability/header/:height
 * https://docs.espressosys.com/sequencer/api-reference/sequencer-api/availability-api#get-availability-header
 */
func (s *ExpressoService) GetHeaderByBlockByHeight(height *big.Int) (*EspressoHeader, error) {
	if s.client == nil {
		mock := 32644

		return &EspressoHeader{
			Height: uint64(mock),
			L1Finalized: &types.L1BlockInfo{
				Number: uint64(mock),
			},
		}, nil
	}

	res, err := s.client.FetchHeaderByHeight(s.context, height.Uint64())

	if err != nil {
		return nil, err
	}

	return &res, nil
}

/**
 * GET /availability/block/:height/namespace/:namespace
 * https://docs.espressosys.com/sequencer/api-reference/sequencer-api/availability-api#get-availability-block-height-namespace-namespace
 */
func (s *ExpressoService) GetTransactionByHeight(height *big.Int) (*EspressoBlockResponse, error) {
	if s.client == nil {
		return &EspressoBlockResponse{
			Transactions: nil,
			Proof:        nil,
		}, nil
	}

	h := height.Uint64()
	// Always fixed in first 16 bits of App Address
	namespace, err := strconv.ParseUint(VM_ID, 10, 64)

	if err != nil {
		return nil, err
	}

	res, err := s.client.FetchTransactionsInBlock(s.context, h, namespace)

	if err != nil {
		return nil, err
	}

	return &res, nil
}
