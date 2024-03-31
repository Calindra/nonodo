// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"fmt"

	"github.com/calindra/nonodo/internal/model"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//
// Nonodo -> GraphQL conversions
//

func convertCompletionStatus(status model.CompletionStatus) CompletionStatus {
	switch status {
	case model.CompletionStatusUnprocessed:
		return CompletionStatusUnprocessed
	case model.CompletionStatusAccepted:
		return CompletionStatusAccepted
	case model.CompletionStatusRejected:
		return CompletionStatusRejected
	case model.CompletionStatusException:
		return CompletionStatusException
	default:
		panic("invalid completion status")
	}
}

func convertInput(input model.AdvanceInput) *Input {
	return &Input{
		Index:       input.Index,
		Status:      convertCompletionStatus(input.Status),
		MsgSender:   input.MsgSender.String(),
		Timestamp:   fmt.Sprint(input.Timestamp.Unix()),
		BlockNumber: fmt.Sprint(input.BlockNumber),
		Payload:     hexutil.Encode(input.Payload),
	}
}

func convertVoucher(voucher model.Voucher) *Voucher {
	return &Voucher{
		InputIndex:  voucher.InputIndex,
		Index:       voucher.Index,
		Destination: voucher.Destination.String(),
		Payload:     hexutil.Encode(voucher.Payload),
		Proof:       nil, // nonodo doesn't compute proofs
	}
}

func convertVouchersMetadata(oldList []*model.VoucherMetadata) []*VoucherMetadata {
	newList := make([]*VoucherMetadata, len(oldList))
	for i, old := range oldList {
		newList[i] = &VoucherMetadata{
			Label:         old.Label,
			Beneficiary:   old.Beneficiary.String(),
			Contract:      old.Contract.String(),
			Amount:        fmt.Sprintf("%d", old.Amount),
			ExecutedAt:    fmt.Sprintf("%d", old.ExecutedAt),
			ExecutedBlock: fmt.Sprintf("%d", old.ExecutedBlock),
			InputIndex:    old.InputIndex,
			OutputIndex:   old.OutputIndex,
		}
	}
	return newList
}

func convertNotice(notice model.Notice) *Notice {
	return &Notice{
		InputIndex: notice.InputIndex,
		Index:      notice.Index,
		Payload:    hexutil.Encode(notice.Payload),
		Proof:      nil, // nonodo doesn't compute proofs
	}
}

func convertReport(report model.Report) *Report {
	return &Report{
		InputIndex: report.InputIndex,
		Index:      report.Index,
		Payload:    hexutil.Encode(report.Payload),
	}
}

//
// GraphQL -> Nonodo conversions
//

func convertInputFilter(filter *InputFilter) model.InputFilter {
	if filter == nil {
		return model.InputFilter{}
	}
	return model.InputFilter{
		IndexGreaterThan: filter.IndexGreaterThan,
		IndexLowerThan:   filter.IndexGreaterThan,
	}
}

func convertVoucherMetadataFilters(filter []*VoucherMetadataFilter) []*model.MetadataFilter {
	result := []*model.MetadataFilter{}
	for _, f := range filter {
		result = append(result, convertVoucherMetadataFilter(f))
	}
	return result
}

func convertVoucherMetadataFilter(filter *VoucherMetadataFilter) *model.MetadataFilter {
	if filter == nil {
		return nil
	}

	result := &model.MetadataFilter{
		Field: filter.Field,
		Eq:    filter.Eq,
		Ne:    filter.Ne,
		Gt:    filter.Gt,
		Gte:   filter.Gte,
		Lt:    filter.Lt,
		Lte:   filter.Lte,
		In:    filter.In,
		Nin:   filter.Nin,
	}

	// Recursively convert And and Or filters
	for _, f := range filter.And {
		result.And = append(result.And, convertVoucherMetadataFilter(f))
	}
	for _, f := range filter.Or {
		result.Or = append(result.Or, convertVoucherMetadataFilter(f))
	}

	return result
}
