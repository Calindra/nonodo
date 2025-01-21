package claimer

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/calindra/nonodo/internal/contracts"
	"github.com/calindra/nonodo/internal/devnet"
	"github.com/calindra/nonodo/internal/merkle"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	if opts == nil {
		aux, err := devnet.DefaultTxOpts(ctx, c.ethClient)
		if err != nil {
			return err
		}
		opts = aux
	}
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

func (c *Claimer) CreateNewOnChainApp(
	ctx context.Context,
	consensusAddress common.Address,
) (*common.Address, error) {
	addressBook := devnet.GetContractInfo()
	appFactoryAddress := common.HexToAddress(addressBook.Contracts["ApplicationFactory"].Address)
	applicationFactory, err := contracts.NewIApplicationFactory(appFactoryAddress, c.ethClient)
	if err != nil {
		return nil, err
	}
	txOpts, err := devnet.DefaultTxOpts(ctx, c.ethClient)
	if err != nil {
		return nil, err
	}

	sender := common.HexToAddress(devnet.AnvilDefaultAddress)
	templateHash := [32]byte{}
	salt := [32]byte{}
	tx, err := applicationFactory.NewApplication(txOpts, consensusAddress, sender, templateHash, salt)
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

type UnifiedOutput struct {
	payload []byte
	proof   contracts.OutputValidityProof
}

func NewUnifiedOutput(payload string, outputIndex uint64) *UnifiedOutput {
	auxPayload := strings.TrimPrefix(payload, "0x")
	return &UnifiedOutput{
		payload: common.Hex2Bytes(auxPayload),
		proof: contracts.OutputValidityProof{
			OutputIndex: outputIndex,
		},
	}
}

func (c *Claimer) FillProofsAndReturnClaim(
	outputs []*UnifiedOutput,
) (common.Hash, error) {
	leaves := make([]common.Hash, len(outputs))
	for i, output := range outputs {
		leaves[i] = crypto.Keccak256Hash(output.payload)
	}
	claim, proofs, err := merkle.CreateProofs(leaves, uint(MAX_OUTPUT_TREE_HEIGHT))
	if err != nil {
		return common.Hash{}, err
	}
	for idx := range outputs {
		// WARN: simplification to avoid redoing all the proofs in the world
		old := outputs[idx].proof.OutputIndex
		outputs[idx].proof.OutputIndex = uint64(idx)
		start := outputs[idx].proof.OutputIndex * MAX_OUTPUT_TREE_HEIGHT
		end := (outputs[idx].proof.OutputIndex * MAX_OUTPUT_TREE_HEIGHT) + MAX_OUTPUT_TREE_HEIGHT
		outputs[idx].proof.OutputHashesSiblings = ConvertHashesToOutputHashesSiblings(proofs[start:end])
		outputs[idx].proof.OutputIndex = old
	}
	return claim, err
}

func ConvertHashesToOutputHashesSiblings(hashes []common.Hash) [][32]byte {
	var output [][32]byte
	for _, hash := range hashes {
		var hashArray [32]byte
		copy(hashArray[:], hash.Bytes())
		output = append(output, hashArray)
	}
	return output
}
