package claimer

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const MAX_OUTPUT_TREE_HEIGHT = 63

type Claimer struct {
	ethClient *ethclient.Client
}

func NewClaimer(
	ethClient *ethclient.Client,
) *Claimer {
	return &Claimer{ethClient}
}

func (c *Claimer) MakeTheClaim(ctx context.Context,
	consensusAddress *common.Address,
	appContract *common.Address,
	outputRootHashAsClaim common.Hash,
	lastProcessedBlockNumber *big.Int,
	opts *bind.TransactOpts,
) error {
	if consensusAddress == nil {
		return fmt.Errorf("missing consensus address")
	}
	if lastProcessedBlockNumber == nil {
		return fmt.Errorf("missing last processed block number")
	}
	consensus, err := contracts.NewIConsensus(*consensusAddress, c.ethClient)
	if err != nil {
		return err
	}
	tx, err := consensus.SubmitClaim(opts, *appContract, lastProcessedBlockNumber, outputRootHashAsClaim)
	if err != nil {
		return err
	}
	slog.Debug("SubmitClaim", "tx", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), c.ethClient, tx)
	if err != nil {
		return err
	}
	slog.Debug("SubmitClaim", "receipt.status", receipt.Status)
	abi, err := contracts.IConsensusMetaData.GetAbi()
	if err != nil {
		return err
	}
	for _, vLog := range receipt.Logs {
		event := struct {
			Submitter                common.Address
			AppContract              common.Address
			LastProcessedBlockNumber *big.Int
			Claim                    [32]byte
		}{}
		err := abi.UnpackIntoInterface(&event, "ClaimSubmission", vLog.Data)
		if err != nil {
			slog.Debug("failed to decode",
				"vLog.Data", common.Bytes2Hex(vLog.Data),
				"topics", vLog.Topics,
				"err", err,
			)
			continue
		}
		slog.Debug("SubmitClaim event decoded", "data", event)
	}
	return err
}

func (c *Claimer) CreateConsensusTypeAuthority(ctx context.Context) (*common.Address, error) {
	contractInfo := devnet.GetContractInfo()
	authorityFactoryAddress := common.HexToAddress(contractInfo.Contracts["AuthorityFactory"].Address)
	authorityFactory, err := contracts.NewIAuthorityFactory(authorityFactoryAddress, c.ethClient)
	if err != nil {
		return nil, err
	}
	txOpts, err := devnet.DefaultTxOpts(ctx, c.ethClient)
	if err != nil {
		return nil, err
	}
	epochLen := new(big.Int).SetUint64(10) // nolint
	sender := common.HexToAddress(devnet.SenderAddress)
	salt := [32]byte{}
	slog.Debug("CreateConsensusAuthorityType", "salt", common.Bytes2Hex(salt[:]))
	tx, err := authorityFactory.NewAuthority0(txOpts, sender, epochLen, salt)
	if err != nil {
		return nil, err
	}
	receipt, err := bind.WaitMined(context.Background(), c.ethClient, tx)
	if err != nil {
		return nil, err
	}
	authFactoryAbi, err := contracts.IAuthorityFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	for _, vLog := range receipt.Logs {
		event := struct {
			Authority common.Address
		}{}
		err := authFactoryAbi.UnpackIntoInterface(&event, "AuthorityCreated", vLog.Data)
		if err != nil {
			continue // Skip logs that don't match
		}
		return &event.Authority, nil
	}
	return nil, nil
}

func (c *Claimer) Deploy(
	ctx context.Context,
	consensusAddress common.Address,
) (*common.Address, error) {
	addressBook := devnet.GetContractInfo()
	appFactoryAddress := common.HexToAddress(addressBook.Contracts["ApplicationFactory"].Address)
	applicationFactory, err := contracts.NewIApplicationFactory(appFactoryAddress, c.ethClient)
	if err != nil {
		return nil, err
	}
	txOpts1, err := devnet.DefaultTxOpts(ctx, c.ethClient)
	if err != nil {
		return nil, err
	}

	sender := common.HexToAddress(devnet.AnvilDefaultAddress)
	templateHash := [32]byte{}
	salt := [32]byte{}
	tx, err := applicationFactory.NewApplication(txOpts1, consensusAddress, sender, templateHash, salt)
	if err != nil {
		return nil, err
	}
	slog.Debug("Deploy", "tx.hash", tx.Hash())

	receipt, err := bind.WaitMined(context.Background(), c.ethClient, tx)
	if err != nil {
		return nil, err
	}

	authFactoryAbi, err := contracts.IApplicationFactoryMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	for _, vLog := range receipt.Logs {
		event := struct {
			Consensus    common.Address
			AppOwner     common.Address
			TemplateHash [32]byte
			AppContract  common.Address
		}{}
		err := authFactoryAbi.UnpackIntoInterface(&event, "ApplicationCreated", vLog.Data)
		if err != nil {
			continue // Skip logs that don't match
		}
		return &event.AppContract, nil
	}
	return nil, nil
}
