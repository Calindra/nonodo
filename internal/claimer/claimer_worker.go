package claimer

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const DEFAULT_EPOCH_BLOCKS = 10

type ClaimerWorker struct {
	RpcUrl            string
	ethClient         *ethclient.Client
	ClaimerService    *ClaimerService
	voucherRepository *repository.VoucherRepository
	noticeRepository  *repository.NoticeRepository
	consensusAddress  *common.Address
	appAddress        *common.Address
	epochBlocks       uint64
}

func NewClaimerWorker(
	rpcURL string,
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	epochBlocks int,
) *ClaimerWorker {
	return &ClaimerWorker{
		RpcUrl:            rpcURL,
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
		epochBlocks:       uint64(epochBlocks),
	}
}

func (c *ClaimerWorker) String() string {
	return "claimer_worker"
}

func (c *ClaimerWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	client, err := ethclient.DialContext(ctx, c.RpcUrl)
	if err != nil {
		return err
	}
	c.ethClient = client
	claimer := NewClaimer(c.ethClient)
	c.ClaimerService = NewClaimService(
		c.voucherRepository,
		c.noticeRepository,
		claimer,
	)
	consensusAddress, err := claimer.CreateConsensusTypeAuthority(ctx)
	if err != nil {
		return err
	}
	if consensusAddress == nil {
		return fmt.Errorf("fail to create consensus")
	}
	c.consensusAddress = consensusAddress

	appAddress, err := claimer.CreateNewOnChainApp(ctx, *c.consensusAddress)
	if err != nil {
		return err
	}
	c.appAddress = appAddress
	slog.Info("AppAddress", "appAddress", appAddress.Hex())
	ready <- struct{}{}
	return c.watchNewBlocks(ctx)
}

func (c *ClaimerWorker) watchNewBlocks(ctx context.Context) error {
	headers := make(chan *types.Header)
	sub, err := c.ethClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatalf("Failed to subscribe to new blocks: %v", err)
		return err
	}
	defer sub.Unsubscribe()

	go func() {
		for header := range headers {
			slog.Debug("New block mined",
				"hash", header.Hash().Hex(),
				"blockNumber", header.Number,
			)
			blockNumber := header.Number.Uint64()
			if blockNumber > 0 && blockNumber%c.epochBlocks == 0 {
				err := c.ClaimerService.CreateProofsAndSendClaim(
					ctx,
					*c.consensusAddress,
					blockNumber-c.epochBlocks,
					blockNumber,
				)
				if err != nil {
					slog.Error("Error creating proofs and claim")
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-sub.Err():
		return err
	}
}
