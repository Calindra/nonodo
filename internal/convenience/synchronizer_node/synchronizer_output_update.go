package synchronizernode

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerOutputUpdate struct {
	VoucherRepository      *repository.VoucherRepository
	NoticeRepository       *repository.NoticeRepository
	RawNodeV2Repository    *RawRepository
	RawOutputRefRepository *repository.RawOutputRefRepository
}

func NewSynchronizerOutputUpdate(
	voucherRepository *repository.VoucherRepository,
	noticeRepository *repository.NoticeRepository,
	rawRepository *RawRepository,
	rawOutputRefRepository *repository.RawOutputRefRepository,
) *SynchronizerOutputUpdate {
	return &SynchronizerOutputUpdate{
		VoucherRepository:      voucherRepository,
		NoticeRepository:       noticeRepository,
		RawNodeV2Repository:    rawRepository,
		RawOutputRefRepository: rawOutputRefRepository,
	}
}

func (s *SynchronizerOutputUpdate) SyncOutputs(ctx context.Context) error {
	txCtx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	err = s.syncOutputs(txCtx)
	if err != nil {
		s.rollbackTransaction(txCtx)
		return err
	}
	err = s.commitTransaction(txCtx)
	if err != nil {
		return err
	}
	return nil
}

func (s *SynchronizerOutputUpdate) syncOutputs(ctx context.Context) error {
	lastOutputIdWithoutProof, err := s.RawOutputRefRepository.GetFirstOutputIdWithoutProof(ctx)
	if err != nil {
		return err
	}
	if lastOutputIdWithoutProof == 0 {
		return nil
	}
	slog.Debug("SyncOutputs", "lastOutputIdWithoutProof", lastOutputIdWithoutProof)
	rawOutputs, err := s.RawNodeV2Repository.FindAllOutputsWithProof(ctx, FilterID{
		IDgt: lastOutputIdWithoutProof,
	})
	if err != nil {
		return err
	}
	for _, rawOutput := range rawOutputs {
		hashes, err := parseAndDecode(string(rawOutput.OutputHashesSiblings))
		if err != nil {
			return err
		}
		err = s.UpdateProof(ctx, rawOutput, hashes)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SynchronizerOutputUpdate) UpdateProof(
	ctx context.Context,
	rawOutput Output,
	hashes []string,
) error {
	ref, err := s.RawOutputRefRepository.FindByID(ctx, rawOutput.ID)
	if err != nil {
		return err
	}
	if ref == nil {
		slog.Warn("We may need to wait for the reference to be created")
		return nil
	}
	// slog.Debug("Ref",
	// 	"appContract", ref.AppContract,
	// 	"OutputIndex", ref.OutputIndex,
	// )
	jsonSiblings, err := json.Marshal(hashes)
	if err != nil {
		return err
	}
	if ref.Type == repository.RAW_VOUCHER_TYPE {
		err = s.VoucherRepository.SetProof(ctx,
			&model.ConvenienceVoucher{
				AppContract:          common.HexToAddress(ref.AppContract),
				OutputIndex:          ref.OutputIndex,
				OutputHashesSiblings: string(jsonSiblings),
				ProofOutputIndex:     ref.OutputIndex,
			})
		if err != nil {
			return err
		}
	} else if ref.Type == repository.RAW_NOTICE_TYPE {
		err = s.NoticeRepository.SetProof(ctx,
			&model.ConvenienceNotice{
				AppContract:          ref.AppContract,
				OutputIndex:          ref.OutputIndex,
				OutputHashesSiblings: string(jsonSiblings),
				ProofOutputIndex:     ref.OutputIndex,
			})
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unexpected output type: %s", ref.Type)
	}
	err = s.RawOutputRefRepository.SetHasProofToTrue(ctx, ref)
	if err != nil {
		return err
	}
	return nil
}

func parseAndDecode(input string) ([]string, error) {
	cleaned := strings.Trim(input, "{}")
	parts := strings.Split(cleaned, ",")
	var decoded []string
	for _, part := range parts {
		trimmed := strings.Trim(part, `" `) // Remove any quotes or spaces.
		hexStr := strings.ReplaceAll(trimmed, `\\x`, `\x`)
		hexStr = strings.ReplaceAll(hexStr, `\x`, "")
		bytes, err := hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex: %v", err)
		}
		decoded = append(decoded, fmt.Sprintf("0x%x", bytes))
	}
	return decoded, nil
}

func (s *SynchronizerOutputUpdate) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.RawOutputRefRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerOutputUpdate) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerOutputUpdate) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}
