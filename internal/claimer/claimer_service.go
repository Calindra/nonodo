package claimer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math/big"

	"github.com/calindra/nonodo/internal/convenience/repository"
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
	vouchers, err := c.VoucherRepository.FindAllVouchersByBlockNumber(
		ctx,
		startBlockGte,
		endBlockLt,
	)
	if err != nil {
		return err
	}
	notices, err := c.NoticeRepository.FindAllNoticesByBlockNumber(
		ctx,
		startBlockGte,
		endBlockLt,
	)
	if err != nil {
		return err
	}
	slog.Debug("CreateProofs",
		"vouchers", len(vouchers),
		"startBlockGte", startBlockGte,
		"endBlockLt", endBlockLt,
	)
	lenVouchers := len(vouchers)
	lenNotices := len(notices)
	outputs := make([]*UnifiedOutput, lenVouchers+lenNotices)
	for i, voucher := range vouchers {
		outputs[i] = NewUnifiedOutput(voucher.Payload)
	}
	for i, notice := range notices {
		outputs[i+lenVouchers] = NewUnifiedOutput(notice.Payload)
	}
	claim, err := c.claimer.FillProofsAndReturnClaim(outputs)
	if err != nil {
		return err
	}
	slog.Debug("CreateProofs", "claim", claim.Hex())
	for i := range vouchers {
		vouchers[i].OutputHashesSiblings = ToJsonArray(outputs[i].proof.OutputHashesSiblings)
		vouchers[i].ProofOutputIndex = outputs[i].proof.OutputIndex
		err := c.VoucherRepository.SetProof(ctx, vouchers[i])
		if err != nil {
			return err
		}
	}
	for i := range notices {
		notices[i].OutputHashesSiblings = ToJsonArray(outputs[i+lenVouchers].proof.OutputHashesSiblings)
		notices[i].ProofOutputIndex = outputs[i+lenVouchers].proof.OutputIndex
		err := c.NoticeRepository.SetProof(ctx, notices[i])
		if err != nil {
			return err
		}
	}
	if lenVouchers == 0 && lenNotices == 0 {
		return nil
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
