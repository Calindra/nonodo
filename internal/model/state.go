// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"context"
	"fmt"
	"log/slog"

	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
	cRepos "github.com/cartesi/rollups-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Interface that represents the state of the rollup.
type rollupsState interface {

	// Finish the current state, saving the result to the model.
	finish(status cModel.CompletionStatus) error

	// Add voucher to current state.
	addVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error)

	// Add voucher to current state.
	addDCVoucher(appAddress common.Address, destination common.Address, payload []byte) (int, error)

	// Add notice to current state.
	addNotice(payload []byte, appAddress common.Address) (int, error)

	// Add report to current state.
	addReport(appAddress common.Address, payload []byte) error

	// Register exception in current state.
	registerException(payload []byte) error
}

// Convenience OutputDecoder
type Decoder interface {
	HandleOutput(
		ctx context.Context,
		destination common.Address,
		payload string,
		inputIndex uint64,
		outputIndex uint64,
	) error
}

//
// Idle
//

// In the idle state, the model waits for an finish request from the rollups API.
type rollupsStateIdle struct{}

func newRollupsStateIdle() rollupsState {
	return &rollupsStateIdle{}
}

func (s *rollupsStateIdle) finish(status cModel.CompletionStatus) error {
	return nil
}

// addDCVoucher implements rollupsState.
func (s *rollupsStateIdle) addDCVoucher(appAddress common.Address, destination common.Address, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add delegate call voucher in idle state")
}

func (s *rollupsStateIdle) addVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add voucher in idle state")
}

func (s *rollupsStateIdle) addNotice(payload []byte, appAddress common.Address) (int, error) {
	return 0, fmt.Errorf("cannot add notice in current state")
}

func (s *rollupsStateIdle) addReport(appAddress common.Address, payload []byte) error {
	return fmt.Errorf("cannot add report in current state")
}

func (s *rollupsStateIdle) registerException(payload []byte) error {
	return fmt.Errorf("cannot register exception in current state")
}

//
// Advance
//

// In the advance state, the model accumulates the outputs from an advance.
type rollupsStateAdvance struct {
	input             *cModel.AdvanceInput
	vouchers          []cModel.ConvenienceVoucher
	notices           []cModel.ConvenienceNotice
	reports           []cModel.Report
	decoder           Decoder
	reportRepository  *cRepos.ReportRepository
	inputRepository   *cRepos.InputRepository
	voucherRepository *cRepos.VoucherRepository
	noticeRepository  *cRepos.NoticeRepository
}

func newRollupsStateAdvance(
	input *cModel.AdvanceInput,
	decoder Decoder,
	reportRepository *cRepos.ReportRepository,
	inputRepository *cRepos.InputRepository,
	voucherRepository *cRepos.VoucherRepository,
	noticeRepository *cRepos.NoticeRepository,
) rollupsState {
	slog.Info("nonodo: processing advance", "index", input.Index)
	return &rollupsStateAdvance{
		input:             input,
		decoder:           decoder,
		reportRepository:  reportRepository,
		inputRepository:   inputRepository,
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
	}
}

func saveAllInputVouchers(voucherRepository *cRepos.VoucherRepository, inputIndex uint64, vouchers []cModel.ConvenienceVoucher) error {
	if voucherRepository == nil {
		slog.Warn("Missing voucherRepository to send vouchers")
		return nil
	}
	ctx := context.Background()
	for _, v := range vouchers {
		v.InputIndex = inputIndex
		v.Payload = fmt.Sprintf("0x%s", v.Payload)
		slog.Info("nonodo saving voucher", "autocount", voucherRepository.AutoCount)
		_, err := voucherRepository.CreateVoucher(ctx, &v)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveAllInputNotices(noticeRepository *cRepos.NoticeRepository, inputIndex uint64, notices []cModel.ConvenienceNotice) error {
	if noticeRepository == nil {
		slog.Warn("Missing noticeRepository to send notices")
		return nil
	}
	ctx := context.Background()
	for _, v := range notices {
		v.Payload = fmt.Sprintf("0x%s", v.Payload)
		v.InputIndex = inputIndex
		_, err := noticeRepository.Create(ctx, &v)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveAllReports(reportRepository *cRepos.ReportRepository, reports []cModel.Report) error {
	if reportRepository == nil {
		slog.Warn("Missing reportRepository to save reports")
		return nil
	}
	if reportRepository.Db == nil {
		slog.Warn("Missing reportRepository.Db to save reports")
		return nil
	}
	ctx := context.Background()
	for _, r := range reports {
		_, err := reportRepository.CreateReport(ctx, r)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *rollupsStateAdvance) finish(status cModel.CompletionStatus) error {
	s.input.Status = status
	if status == cModel.CompletionStatusAccepted {
		s.input.Vouchers = s.vouchers
		s.input.Notices = s.notices
		if s.decoder != nil {
			err := saveAllInputVouchers(s.voucherRepository, uint64(s.input.Index), s.vouchers)

			if err != nil {
				slog.Error("Error sending all input vouchers to decoder", "Error", err)
				return err
			}

			err = saveAllInputNotices(s.noticeRepository, uint64(s.input.Index), s.notices)

			if err != nil {
				slog.Error("Error sending all input notices to decoder", "Error", err)
				return err
			}
		}
	}
	// s.input.Reports = s.reports
	ctx := context.Background()

	err := saveAllReports(s.reportRepository, s.reports)

	if err != nil {
		slog.Error("Error saving reports", "Error", err)
		return err
	}
	_, erro := s.inputRepository.Update(ctx, *s.input)

	if erro != nil {
		return erro
	}
	slog.Info("nonodo: finished advance")
	return nil
}

func (s *rollupsStateAdvance) addVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error) {
	index := len(s.vouchers)
	voucher := cModel.ConvenienceVoucher{
		AppContract: appAddress,
		OutputIndex: uint64(index),
		InputIndex:  uint64(s.input.Index),
		Destination: destination,
		Payload:     common.Bytes2Hex(payload),
		Value:       value,
	}
	s.vouchers = append(s.vouchers, voucher)
	slog.Info("nonodo: added voucher", "index", index, "destination", destination,
		"value", value, "payload", hexutil.Encode(payload))
	return index, nil
}

// addDCVoucher implements rollupsState.
func (s *rollupsStateAdvance) addDCVoucher(appAddress common.Address, destination common.Address, payload []byte) (int, error) {
	index := len(s.vouchers)
	dcvoucher := cModel.ConvenienceVoucher{
		AppContract:     appAddress,
		OutputIndex:     uint64(index),
		InputIndex:      uint64(s.input.Index),
		Destination:     destination,
		Payload:         common.Bytes2Hex(payload),
		IsDelegatedCall: true,
	}
	s.vouchers = append(s.vouchers, dcvoucher)
	slog.Info("nonodo: added delegate call voucher", "index", index, "destination", destination,
		"payload", hexutil.Encode(payload))
	return index, nil
}

func (s *rollupsStateAdvance) addNotice(payload []byte, appAddress common.Address) (int, error) {
	index := len(s.notices)
	notice := cModel.ConvenienceNotice{
		AppContract: appAddress.Hex(),
		OutputIndex: uint64(index),
		InputIndex:  uint64(s.input.Index),
		Payload:     common.Bytes2Hex(payload),
	}
	s.notices = append(s.notices, notice)
	slog.Info("nonodo: added notice", "index", index, "payload", hexutil.Encode(payload))
	return index, nil
}

func (s *rollupsStateAdvance) addReport(appAddress common.Address, payload []byte) error {
	index := len(s.reports)
	report := cModel.Report{
		AppContract: appAddress,
		Index:       index,
		InputIndex:  s.input.Index,
		Payload:     common.Bytes2Hex(payload),
	}
	s.reports = append(s.reports, report)
	slog.Info("nonodo: added report", "index", index, "payload", hexutil.Encode(payload))
	return nil
}

func (s *rollupsStateAdvance) registerException(payload []byte) error {
	s.input.Status = cModel.CompletionStatusException
	s.input.Reports = s.reports
	s.input.Exception = payload
	ctx := context.Background()
	_, err := s.inputRepository.Update(ctx, *s.input)
	if err != nil {
		return err
	}
	err = saveAllReports(s.reportRepository, s.reports)

	if err != nil {
		return err
	}

	slog.Info("nonodo: finished advance with exception")
	return nil
}

//
// Inspect
//

// In the inspect state, the model accumulates the reports from an inspect.
type rollupsStateInspect struct {
	input                  *InspectInput
	reports                []Report
	getProcessedInputCount func() (int, error)
}

func newRollupsStateInspect(
	input *InspectInput,
	getProcessedInputCount func() (int, error),
) rollupsState {
	slog.Info("nonodo: processing inspect", "index", input.Index)
	return &rollupsStateInspect{
		input:                  input,
		getProcessedInputCount: getProcessedInputCount,
	}
}

func (s *rollupsStateInspect) finish(status cModel.CompletionStatus) error {
	s.input.Status = status
	inputCount, err := s.getProcessedInputCount()

	if err != nil {
		slog.Error("Error getting processed input count", "Error", err)
		return err
	}

	s.input.ProcessedInputCount = inputCount
	s.input.Reports = s.reports
	slog.Info("nonodo: finished inspect")
	return nil
}

func (s *rollupsStateInspect) addVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add voucher in inspect state")
}

// addDCVoucher implements rollupsState.
func (s *rollupsStateInspect) addDCVoucher(appAddress common.Address, destination common.Address, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add delegate call voucher in inspect state")
}

func (s *rollupsStateInspect) addNotice(payload []byte, appAddress common.Address) (int, error) {
	return 0, fmt.Errorf("cannot add notice in current state")
}

func (s *rollupsStateInspect) addReport(appAddress common.Address, payload []byte) error {
	index := len(s.reports)
	report := Report{
		Index:      index,
		InputIndex: s.input.Index,
		Payload:    payload,
	}
	s.reports = append(s.reports, report)
	slog.Info("nonodo: added report", "index", index, "payload", hexutil.Encode(payload))
	return nil
}

func (s *rollupsStateInspect) registerException(payload []byte) error {
	s.input.Status = cModel.CompletionStatusException
	inputCount, _ := s.getProcessedInputCount()
	s.input.ProcessedInputCount = inputCount
	s.input.Reports = s.reports
	s.input.Exception = payload
	slog.Info("nonodo: finished inspect with exception")
	return nil
}
