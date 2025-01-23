package claimer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math/big"
	"sort"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type ClaimerService struct {
	VoucherRepository *repository.VoucherRepository
	NoticeRepository  *repository.NoticeRepository
	claimer           *Claimer
}

func NewClaimService(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	claimer *Claimer,
) *ClaimerService {
	return &ClaimerService{
		VoucherRepository: voucherRepository,
		NoticeRepository:  noticeRepository,
		claimer:           claimer,
	}
}

func (c *ClaimerService) CreateProofsAndSendClaim(
	ctx context.Context,
	consensusAddress common.Address,
	startBlockGte uint64,
	endBlockLt uint64,
) error {
	vouchers, err := c.VoucherRepository.FindAll(ctx)
	if err != nil {
		return err
	}
	notices, err := c.NoticeRepository.FindAllNoticesByBlockNumber(
		ctx,
		0,
		endBlockLt,
	)
	if err != nil {
		return err
	}
	slog.Info("CreateProofs",
		"vouchers", len(vouchers),
		"startBlockGte", startBlockGte,
		"endBlockLt", endBlockLt,
	)
	lenVouchers := len(vouchers)
	lenNotices := len(notices)
	if lenVouchers == 0 && lenNotices == 0 {
		return nil
	}
	outputs := make([]*UnifiedOutput, lenVouchers+lenNotices)
	for i, voucher := range vouchers {
		outputs[i] = NewUnifiedOutput(voucher.Payload, voucher.OutputIndex)
		outputs[i].AppContract = voucher.AppContract
		outputs[i].OutputType = "voucher"
		outputs[i].OutputIndex = voucher.OutputIndex
	}
	for i, notice := range notices {
		outputs[i+lenVouchers] = NewUnifiedOutput(notice.Payload, notice.OutputIndex)
		outputs[i+lenVouchers].AppContract = common.HexToAddress(notice.AppContract)
		outputs[i+lenVouchers].OutputType = "notice"
		outputs[i+lenVouchers].OutputIndex = notice.OutputIndex
	}
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].proof.OutputIndex < outputs[j].proof.OutputIndex
	})
	// for i := range outputs {
	// 	slog.Info("Index",
	// 		"i", i,
	// 		"OutputIndex", outputs[i].proof.OutputIndex,
	// 	)
	// }
	claim, err := c.claimer.FillProofsAndReturnClaim(outputs)
	if err != nil {
		return err
	}
	slog.Debug("CreateProofs", "claim", claim.Hex())
	for i := range outputs {
		if outputs[i].OutputType == "voucher" {
			voucher := model.ConvenienceVoucher{
				AppContract:          outputs[i].AppContract,
				OutputHashesSiblings: ToJsonArray(outputs[i].proof.OutputHashesSiblings),
				OutputIndex:          outputs[i].OutputIndex,
			}
			err := c.VoucherRepository.SetProof(ctx, &voucher)
			if err != nil {
				return err
			}
		} else if outputs[i].OutputType == "notice" {
			notice := model.ConvenienceNotice{
				AppContract:          outputs[i].AppContract.Hex(),
				OutputHashesSiblings: ToJsonArray(outputs[i].proof.OutputHashesSiblings),
				OutputIndex:          outputs[i].OutputIndex,
				ProofOutputIndex:     outputs[i].OutputIndex,
			}
			err := c.NoticeRepository.SetProof(ctx, &notice)
			if err != nil {
				return err
			}
		}
	}
	var appAddress common.Address
	if lenVouchers > 0 {
		appAddress = vouchers[0].AppContract
	} else {
		appAddress = common.HexToAddress(notices[0].AppContract)
	}
	doesNotMatter := new(big.Int).SetInt64(10) // nolint
	err = c.claimer.MakeTheClaim(ctx, &consensusAddress, &appAddress, claim, doesNotMatter, nil)
	if err != nil {
		return err
	}
	return nil
}

func ToJsonArray(OutputHashesSiblings [][32]byte) string {
	var jsonArray []string
	for _, siblings := range OutputHashesSiblings {
		jsonArray = append(jsonArray, "0x"+hex.EncodeToString(siblings[:]))
	}
	jsonData, err := json.Marshal(jsonArray)
	if err != nil {
		panic(err)
	}
	return string(jsonData)
}
