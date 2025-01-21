package synchronizernode

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/cartesi/rollups-graphql/pkg/convenience/model"
	"github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
)

type SynchronizerReport struct {
	ReportRepository *repository.ReportRepository
	RawRepository    *RawRepository
}

func NewSynchronizerReport(
	reportRepository *repository.ReportRepository,
	rawRepository *RawRepository,
) *SynchronizerReport {
	return &SynchronizerReport{
		ReportRepository: reportRepository,
		RawRepository:    rawRepository,
	}
}

func (s *SynchronizerReport) SyncReports(ctx context.Context) error {
	txCtx, err := s.startTransaction(ctx)
	if err != nil {
		return err
	}
	err = s.syncReports(txCtx)
	if err != nil {
		s.rollbackTransaction(txCtx)
		return err
	}
	err = s.commitTransaction(txCtx)
	if err != nil {
		slog.Error("report commit transaction failed")
		panic(err)
	}
	return nil
}

func (s *SynchronizerReport) syncReports(ctx context.Context) error {
	lastRawId, err := s.ReportRepository.FindLastRawId(ctx)
	if err != nil {
		slog.Error("fail to find last report imported")
		return err
	}
	rawReports, err := s.RawRepository.FindAllReportsByFilter(ctx, FilterID{IDgt: lastRawId + 1})
	if err != nil {
		slog.Error("fail to find all reports")
		return err
	}
	for _, rawReport := range rawReports {
		appContract := common.BytesToAddress(rawReport.AppContract)
		index, err := strconv.ParseInt(rawReport.Index, 10, 64) // nolint
		if err != nil {
			slog.Error("fail to parse report index to int", "value", rawReport.Index)
			return err
		}
		inputIndex, err := strconv.ParseInt(rawReport.InputIndex, 10, 64) // nolint
		if err != nil {
			slog.Error("fail to parse input index to int", "value", rawReport.InputIndex)
			return err
		}
		_, err = s.ReportRepository.CreateReport(ctx, model.Report{
			AppContract: appContract,
			Index:       int(index),
			InputIndex:  int(inputIndex),
			Payload:     common.Bytes2Hex(rawReport.RawData),
			RawID:       uint64(rawReport.ID),
		})
		if err != nil {
			slog.Error("fail to create report", "err", err)
			return err
		}
	}
	return nil
}

func (s *SynchronizerReport) startTransaction(ctx context.Context) (context.Context, error) {
	db := s.ReportRepository.Db
	ctxWithTx, err := repository.StartTransaction(ctx, db)
	if err != nil {
		return ctx, err
	}
	return ctxWithTx, nil
}

func (s *SynchronizerReport) commitTransaction(ctx context.Context) error {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		return tx.Commit()
	}
	return nil
}

func (s *SynchronizerReport) rollbackTransaction(ctx context.Context) {
	tx, hasTx := repository.GetTransaction(ctx)
	if hasTx && tx != nil {
		err := tx.Rollback()
		if err != nil {
			slog.Error("transaction rollback error", "err", err)
			panic(err)
		}
	}
}
