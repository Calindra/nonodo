// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// This module is a wrapper for the nonodo model that converts the internal types to
// GraphQL-compatible types.
package model

import (
	"fmt"
	"strconv"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common"
)

// Nonodo model wrapper that convert types to GraphQL types.
type ModelWrapper struct {
	model *model.NonodoModel
}

func NewModelWrapper(model *model.NonodoModel) *ModelWrapper {
	return &ModelWrapper{model}
}

func (m *ModelWrapper) GetInput(index int) (*Input, error) {
	input, ok := m.model.GetAdvanceInput(index)
	if !ok {
		return nil, fmt.Errorf("input not found")
	}
	return convertInput(input), nil
}

func (m *ModelWrapper) GetVoucher(voucherIndex int, inputIndex int) (*Voucher, error) {
	voucher, ok := m.model.GetVoucher(voucherIndex, inputIndex)
	if !ok {
		return nil, fmt.Errorf("voucher not found")
	}
	return convertVoucher(voucher), nil
}

func (m *ModelWrapper) GetNotice(noticeIndex int, inputIndex int) (*Notice, error) {
	notice, ok := m.model.GetNotice(noticeIndex, inputIndex)
	if !ok {
		return nil, fmt.Errorf("notice not found")
	}
	return convertNotice(notice), nil
}

func (m *ModelWrapper) GetReport(reportIndex int, inputIndex int) (*Report, error) {
	report, ok := m.model.GetReport(reportIndex, inputIndex)
	if !ok {
		return nil, fmt.Errorf("report not found")
	}
	return convertReport(report), nil
}

func (m *ModelWrapper) GetInputs(
	first *int, last *int, after *string, before *string, where *InputFilter,
) (*InputConnection, error) {
	filter := convertInputFilter(where)
	total := m.model.GetNumInputs(filter)
	offset, limit, err := computePage(first, last, after, before, total)
	if err != nil {
		return nil, err
	}
	nodes := m.model.GetInputs(filter, offset, limit)
	convNodes := make([]*Input, len(nodes))
	for i := range nodes {
		convNodes[i] = convertInput(nodes[i])
	}
	return newConnection(offset, total, convNodes), nil
}

func (m *ModelWrapper) GetVouchers(
	first *int, last *int, after *string, before *string, inputIndex *int,
) (*VoucherConnection, error) {
	filter := model.OutputFilter{InputIndex: inputIndex}
	total := m.model.GetNumVouchers(filter)
	offset, limit, err := computePage(first, last, after, before, total)
	if err != nil {
		return nil, err
	}
	nodes := m.model.GetVouchers(filter, offset, limit)
	convNodes := make([]*Voucher, len(nodes))
	for i := range nodes {
		convNodes[i] = convertVoucher(nodes[i])
	}
	return newConnection(offset, total, convNodes), nil
}

func (m *ModelWrapper) GetNotices(
	first *int, last *int, after *string, before *string, inputIndex *int,
) (*NoticeConnection, error) {
	filter := model.OutputFilter{InputIndex: inputIndex}
	total := m.model.GetNumNotices(filter)
	offset, limit, err := computePage(first, last, after, before, total)
	if err != nil {
		return nil, err
	}
	nodes := m.model.GetNotices(filter, offset, limit)
	convNodes := make([]*Notice, len(nodes))
	for i := range nodes {
		convNodes[i] = convertNotice(nodes[i])
	}
	return newConnection(offset, total, convNodes), nil
}

func (m *ModelWrapper) GetReports(
	first *int, last *int, after *string, before *string, inputIndex *int,
) (*ReportConnection, error) {
	filter := model.OutputFilter{InputIndex: inputIndex}
	total := m.model.GetNumReports(filter)
	offset, limit, err := computePage(first, last, after, before, total)
	if err != nil {
		return nil, err
	}
	nodes := m.model.GetReports(filter, offset, limit)
	convNodes := make([]*Report, len(nodes))
	for i := range nodes {
		convNodes[i] = convertReport(nodes[i])
	}
	return newConnection(offset, total, convNodes), nil
}

func (m *ModelWrapper) GetVouchersMetadata(
	filter []*VoucherMetadataFilter,
) ([]*VoucherMetadata, error) {
	models, err := m.model.GetVouchersMetadata(convertVoucherMetadataFilters(filter))
	if err != nil {
		return nil, err
	}
	return convertVouchersMetadata(models), nil
}

func (m *ModelWrapper) CreateVoucherMetadata(input NewVoucherMetadata) (*VoucherMetadata, error) {
	amount, err := strconv.ParseUint(input.Amount, 10, 64)
	if err != nil {
		return nil, err
	}
	voucherMetadata := model.VoucherMetadata{
		Contract:      common.HexToAddress(input.Contract),
		Beneficiary:   common.HexToAddress(input.Beneficiary),
		Label:         input.Label,
		Amount:        amount,
		ExecutedAt:    0,
		ExecutedBlock: 0,
		InputIndex:    input.InputIndex,
		OutputIndex:   input.OutputIndex,
	}
	err = m.model.AddVoucherMetadata(&voucherMetadata)
	if err != nil {
		return nil, err
	}
	graphVoucherMetadata := VoucherMetadata{
		Label:         input.Label,
		Contract:      input.Contract,
		Beneficiary:   input.Beneficiary,
		Amount:        input.Amount,
		ExecutedAt:    "0",
		ExecutedBlock: "0",
		InputIndex:    input.InputIndex,
		OutputIndex:   input.OutputIndex,
	}
	return &graphVoucherMetadata, nil
}
