// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Interface that represents the state of the rollup.
type rollupsState interface {

	// Finish the current state, saving the result to the model.
	finish(status CompletionStatus)

	// Add voucher to current state.
	addVoucher(destination common.Address, payload []byte) (int, error)

	// Add notice to current state.
	addNotice(payload []byte) (int, error)

	// Add report to current state.
	addReport(payload []byte) error

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

func newRollupsStateIdle() *rollupsStateIdle {
	return &rollupsStateIdle{}
}

func (s *rollupsStateIdle) finish(status CompletionStatus) {
	// Do nothing
}

func (s *rollupsStateIdle) addVoucher(destination common.Address, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add voucher in current state")
}

func (s *rollupsStateIdle) addNotice(payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add notice in current state")
}

func (s *rollupsStateIdle) addReport(payload []byte) error {
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
	input    *AdvanceInput
	vouchers []Voucher
	notices  []Notice
	reports  []Report
	decoder  Decoder
}

func newRollupsStateAdvance(input *AdvanceInput, decoder Decoder) *rollupsStateAdvance {
	slog.Info("nonodo: processing advance", "index", input.Index)
	return &rollupsStateAdvance{
		input:   input,
		decoder: decoder,
	}
}

func sendAllInputVouchersToDecoder(decoder Decoder, inputIndex uint64, vouchers []Voucher) {
	if decoder == nil {
		slog.Warn("Missing OutputDecoder to send vouchers")
		return
	}
	ctx := context.Background()
	for _, v := range vouchers {
		err := decoder.HandleOutput(
			ctx,
			v.Destination,
			common.Bytes2Hex(v.Payload),
			inputIndex,
			uint64(v.Index),
		)
		if err != nil {
			panic(err)
		}
	}
}

func (s *rollupsStateAdvance) finish(status CompletionStatus) {
	s.input.Status = status
	if status == CompletionStatusAccepted {
		s.input.Vouchers = s.vouchers
		s.input.Notices = s.notices
		if s.decoder != nil {
			sendAllInputVouchersToDecoder(s.decoder, uint64(s.input.Index), s.vouchers)
		}
	}
	s.input.Reports = s.reports
	slog.Info("nonodo: finished advance")
}

func (s *rollupsStateAdvance) addVoucher(destination common.Address, payload []byte) (int, error) {
	index := len(s.vouchers)
	voucher := Voucher{
		Index:       index,
		InputIndex:  s.input.Index,
		Destination: destination,
		Payload:     payload,
	}
	s.vouchers = append(s.vouchers, voucher)
	slog.Info("nonodo: added voucher", "index", index, "destination", destination,
		"payload", hexutil.Encode(payload))
	return index, nil
}

func (s *rollupsStateAdvance) addNotice(payload []byte) (int, error) {
	index := len(s.notices)
	notice := Notice{
		Index:      index,
		InputIndex: s.input.Index,
		Payload:    payload,
	}
	s.notices = append(s.notices, notice)
	slog.Info("nonodo: added notice", "index", index, "payload", hexutil.Encode(payload))
	return index, nil
}

func (s *rollupsStateAdvance) addReport(payload []byte) error {
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

func (s *rollupsStateAdvance) registerException(payload []byte) error {
	s.input.Status = CompletionStatusException
	s.input.Reports = s.reports
	s.input.Exception = payload
	slog.Info("nonodo: finished advance with exception")
	return nil
}

//
// Inspect
//

// In the inspect state, the model accumulates the reports from an inspect.
type rollupsStateInspect struct {
	input                   *InspectInput
	reports                 []Report
	getProccessedInputCount func() int
}

func newRollupsStateInspect(
	input *InspectInput,
	getProccessedInputCount func() int,
) *rollupsStateInspect {
	slog.Info("nonodo: processing inspect", "index", input.Index)
	return &rollupsStateInspect{
		input:                   input,
		getProccessedInputCount: getProccessedInputCount,
	}
}

func (s *rollupsStateInspect) finish(status CompletionStatus) {
	s.input.Status = status
	s.input.ProccessedInputCount = s.getProccessedInputCount()
	s.input.Reports = s.reports
	slog.Info("nonodo: finished inspect")
}

func (s *rollupsStateInspect) addVoucher(destination common.Address, payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add voucher in current state")
}

func (s *rollupsStateInspect) addNotice(payload []byte) (int, error) {
	return 0, fmt.Errorf("cannot add notice in current state")
}

func (s *rollupsStateInspect) addReport(payload []byte) error {
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
	s.input.Status = CompletionStatusException
	s.input.ProccessedInputCount = s.getProccessedInputCount()
	s.input.Reports = s.reports
	s.input.Exception = payload
	slog.Info("nonodo: finished inspect with exception")
	return nil
}
