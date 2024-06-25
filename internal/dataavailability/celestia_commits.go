package dataavailability

import (
	"context"
	"fmt"

	blobstreamx "github.com/calindra/nonodo/internal/dataavailability/contracts/BlobstreamX.sol"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var contractAddress common.Address = common.HexToAddress("0xc3e209eb245Fd59c8586777b499d6A665DF3ABD2")

const maxFilterRange = uint64(10_000)

// GetDataCommitment returns the data commitment event matching the given height
// within the last `blocks` blocks.
// Example usage:
//
//	dataCommitment, err := GetDataCommitment(eth, 10000, 90_000 )
//
// Please note this method will make at least blocks/maxFilterRange calls to the
// Ethereum node
func GetDataCommitment(eth *ethclient.Client, height int64, blocks uint64) (*blobstreamx.BlobstreamXDataCommitmentStored, error) {
	ctx := context.Background()

	contract, err := blobstreamx.NewBlobstreamX(contractAddress, eth)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract: %w", err)
	}

	head, err := eth.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get block number: %w", err)
	}

	if uint64(head) < blocks {
		blocks = uint64(head)
	}

	fmt.Printf("Scanning from Head block: %d Block Range: %d\n", head, blocks)

	// Scan backwards
	// Scan in chunks of `maxFilterRange` blocks
	// Stop when we reach `head - scanRange`
	for end := uint64(head); end > uint64(head)-blocks; end -= maxFilterRange {
		start := end - maxFilterRange
		if start < uint64(head)-blocks {
			start = uint64(head) - blocks
		}

		dataCommitment, err := findMatchingDataCommitment(contract, start, end, height)
		if err != nil {
			return nil, fmt.Errorf("failed to find matching data commitment: %w", err)
		}
		if dataCommitment != nil {
			return dataCommitment, nil
		}
	}

	return nil, fmt.Errorf("no matching data commitment found")
}

func findMatchingDataCommitment(contract *blobstreamx.BlobstreamX, start uint64, end uint64, height int64) (*blobstreamx.BlobstreamXDataCommitmentStored, error) {
	// fmt.Printf("Scanning blocks %d to %d\n", start, end)

	// Filter events
	events, err := contract.FilterDataCommitmentStored(&bind.FilterOpts{
		Context: context.Background(),
		Start:   start,
		End:     &end,
	}, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to filter events: %w", err)
	}
	defer events.Close()

	fmt.Printf("Events start %d; end %d;\n", start, end)

	count := 0
	// loop through events and return the first matching event
	for events.Next() {
		e := events.Event
		count += 1
		if int64(e.StartBlock) <= height && height < int64(e.EndBlock) {
			fmt.Printf("Found height %d; StartBlock %d; EndBlock %d; %s\n",
				height, e.StartBlock, e.EndBlock,
				common.Bytes2Hex(e.DataCommitment[:]),
			)
			return e, nil
		}
		fmt.Printf("Events height %d; StartBlock %d; EndBlock %d; %s\n",
			height, e.StartBlock, e.EndBlock,
			common.Bytes2Hex(e.DataCommitment[:]),
		)
	}
	fmt.Printf("Events not found; count %d\n", count)
	return nil, nil
}
