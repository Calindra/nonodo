// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	cModel "github.com/cartesi/rollups-graphql/pkg/convenience/model"
)

//
// Nonodo -> GraphQL conversions
//

func convertCompletionStatus(status cModel.CompletionStatus) (CompletionStatus, error) {
	switch status {
	case cModel.CompletionStatusUnprocessed:
		return CompletionStatusUnprocessed, nil
	case cModel.CompletionStatusAccepted:
		return CompletionStatusAccepted, nil
	case cModel.CompletionStatusRejected:
		return CompletionStatusRejected, nil
	case cModel.CompletionStatusException:
		return CompletionStatusException, nil
	case cModel.CompletionStatusMachineHalted:
		return CompletionStatusMachineHalted, nil
	case cModel.CompletionStatusCycleLimitExceeded:
		return CompletionStatusCycleLimitExceeded, nil
	case cModel.CompletionStatusTimeLimitExceeded:
		return CompletionStatusTimeLimitExceeded, nil
	case cModel.CompletionStatusPayloadLengthLimitExceeded:
		return CompletionStatusPayloadLengthLimitExceeded, nil
	default:
		return "", errors.New("invalid completion status")
	}
}

func ConvertInput(input cModel.AdvanceInput) (*Input, error) {
	convertedStatus, err := convertCompletionStatus(input.Status)

	if err != nil {
		slog.Error("Error converting CompletionStatus", "Error", err)
		return nil, err
	}

	espressoBlockTimestampStr := strconv.FormatInt(input.EspressoBlockTimestamp.Unix(), 10)
	if espressoBlockTimestampStr == "-1" {
		espressoBlockTimestampStr = ""
	}
	espressoBlockNumberStr := strconv.FormatInt(int64(input.EspressoBlockNumber), 10)
	if espressoBlockNumberStr == "-1" {
		espressoBlockNumberStr = ""
	}

	var inputBoxIndexStr string
	if input.InputBoxIndex != -1 {
		inputBoxIndexStr = strconv.FormatInt(int64(input.InputBoxIndex), 10)
	}

	timestamp := fmt.Sprint(input.BlockTimestamp.Unix())
	return &Input{
		ID:                  input.ID,
		Index:               input.Index,
		Status:              convertedStatus,
		MsgSender:           input.MsgSender.String(),
		Timestamp:           timestamp,
		BlockNumber:         fmt.Sprint(input.BlockNumber),
		Payload:             input.Payload,
		EspressoTimestamp:   espressoBlockTimestampStr,
		EspressoBlockNumber: espressoBlockNumberStr,
		InputBoxIndex:       inputBoxIndexStr,
		BlockTimestamp:      timestamp,
		PrevRandao:          input.PrevRandao,
	}, nil
}

func ConvertConvenientVoucherV1(cVoucher cModel.ConvenienceVoucher) *Voucher {
	var outputHashesSiblings []string
	err := json.Unmarshal([]byte(cVoucher.OutputHashesSiblings), &outputHashesSiblings)
	if err != nil {
		outputHashesSiblings = []string{}
	}
	return &Voucher{
		Index:           int(cVoucher.OutputIndex),
		InputIndex:      int(cVoucher.InputIndex),
		Destination:     cVoucher.Destination.String(),
		Payload:         cVoucher.Payload,
		Value:           cVoucher.Value,
		Executed:        cVoucher.Executed,
		TransactionHash: cVoucher.TransactionHash,
		Proof: Proof{
			OutputIndex:          strconv.FormatUint(cVoucher.ProofOutputIndex, 10),
			OutputHashesSiblings: outputHashesSiblings,
		},
	}
}

func ConvertToConvenienceFilter(
	filter []*ConvenientFilter,
) ([]*cModel.ConvenienceFilter, error) {
	filters := []*cModel.ConvenienceFilter{}
	if filter == nil {
		return filters, nil
	}
	for _, f := range filter {
		and, err := ConvertToConvenienceFilter(f.And)
		if err != nil {
			return nil, err
		}
		or, err := ConvertToConvenienceFilter(f.Or)
		if err != nil {
			return nil, err
		}

		// Destination
		if f.Destination != nil {
			_and, err := ConvertToConvenienceFilter(f.Destination.And)
			if err != nil {
				return nil, err
			}
			and = append(_and, and...)
			_or, err := ConvertToConvenienceFilter(f.Destination.Or)
			if err != nil {
				return nil, err
			}
			or = append(_or, or...)

			filter := "Destination"
			filters = append(filters, &cModel.ConvenienceFilter{
				Field: &filter,
				Eq:    f.Destination.Eq,
				Ne:    f.Destination.Ne,
				Gt:    nil,
				Gte:   nil,
				Lt:    nil,
				Lte:   nil,
				In:    f.Destination.In,
				Nin:   f.Destination.Nin,
				And:   and,
				Or:    or,
			})
		}

		// Executed
		if f.Executed != nil {
			_and, err := ConvertToConvenienceFilter(f.Executed.And)
			if err != nil {
				return nil, err
			}
			and = append(_and, and...)
			_or, err := ConvertToConvenienceFilter(f.Executed.Or)
			if err != nil {
				return nil, err
			}
			or = append(_or, or...)

			var eq string
			var ne string

			if f.Executed.Eq != nil {
				eq = strconv.FormatBool(*f.Executed.Eq)
			}

			if f.Executed.Ne != nil {
				ne = strconv.FormatBool(*f.Executed.Ne)
			}

			filter := "Executed"
			filters = append(filters, &cModel.ConvenienceFilter{
				Field: &filter,
				Eq:    &eq,
				Ne:    &ne,
				Gt:    nil,
				Gte:   nil,
				Lt:    nil,
				Lte:   nil,
				In:    nil,
				Nin:   nil,
				And:   and,
				Or:    or,
			})
		}
	}
	return filters, nil
}

func ConvertToVoucherConnectionV1(
	vouchers []cModel.ConvenienceVoucher,
	offset int, total int,
) (*VoucherConnection, error) {
	convNodes := make([]*Voucher, len(vouchers))
	for i := range vouchers {
		convNodes[i] = ConvertConvenientVoucherV1(vouchers[i])
	}
	return NewConnection(offset, total, convNodes), nil
}

func ConvertConvenientNoticeV1(cNotice cModel.ConvenienceNotice) *Notice {
	var outputHashesSiblings []string
	err := json.Unmarshal([]byte(cNotice.OutputHashesSiblings), &outputHashesSiblings)
	if err != nil {
		outputHashesSiblings = []string{}
	}
	return &Notice{
		Index:      int(cNotice.OutputIndex),
		InputIndex: int(cNotice.InputIndex),
		Payload:    cNotice.Payload,
		Proof: Proof{
			OutputIndex:          strconv.FormatUint(cNotice.ProofOutputIndex, 10),
			OutputHashesSiblings: outputHashesSiblings,
		},
	}
}

func ConvertToNoticeConnectionV1(
	notices []cModel.ConvenienceNotice,
	offset int, total int,
) (*NoticeConnection, error) {
	convNodes := make([]*Notice, len(notices))
	for i := range notices {
		convNodes[i] = ConvertConvenientNoticeV1(notices[i])
	}
	return NewConnection(offset, total, convNodes), nil
}

func ConvertToInputConnectionV1(
	inputs []cModel.AdvanceInput,
	offset int, total int,
) (*InputConnection, error) {
	convNodes := make([]*Input, len(inputs))
	for i := range inputs {
		convertedInput, err := ConvertInput(inputs[i])

		if err != nil {
			return nil, err
		}

		convNodes[i] = convertedInput
	}
	return NewConnection(offset, total, convNodes), nil
}

//
// GraphQL -> Nonodo conversions
//
