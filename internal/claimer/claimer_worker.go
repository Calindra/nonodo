package claimer

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/calindra/nonodo/internal/convenience/repository"
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
}

func NewClaimerWorker(
	rpcURL string,
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
) *ClaimerWorker {
	return &ClaimerWorker{
		RpcUrl:            rpcURL,
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
	}
}

func (c *ClaimerWorker) String() string {
	return "claimer_worker"
}

func (c *ClaimerWorker) Start(ctx context.Context, ready chan<- struct{}) error {
	ready <- struct{}{}
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

	appAddress, err := claimer.Deploy(ctx, *c.consensusAddress)
	if err != nil {
		return err
	}
	c.appAddress = appAddress
	slog.Info("AppAddress", "appAddress", appAddress.Hex())
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
			if blockNumber > 0 && blockNumber%DEFAULT_EPOCH_BLOCKS == 0 {
				err := c.ClaimerService.CreateProofs(
					ctx,
					*c.consensusAddress,
					blockNumber-DEFAULT_EPOCH_BLOCKS,
					blockNumber,
				)
				if err != nil {
					panic(err)
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
