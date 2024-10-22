package synchronizernode

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/calindra/nonodo/internal/convenience/repository"
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
			Payload:     rawReport.RawData,
			RawID:       uint64(rawReport.ID),
		})
		if err != nil {
			slog.Error("fail to create report", "err", err)
			return err
		}
	}
	return nil
}
