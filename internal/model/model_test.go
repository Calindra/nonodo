// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type ModelSuite struct {
	suite.Suite
	m            *NonodoModel
	n            int
	payloads     [][]byte
	senders      []common.Address
	blockNumbers []uint64
	timestamps   []time.Time
}

func (s *ModelSuite) SetupTest() {
	s.m = NewNonodoModel(nil)
	s.n = 3
	s.payloads = make([][]byte, s.n)
	s.senders = make([]common.Address, s.n)
	s.blockNumbers = make([]uint64, s.n)
	s.timestamps = make([]time.Time, s.n)
	now := time.Now()
	for i := 0; i < s.n; i++ {
		for addrI := 0; addrI < common.AddressLength; addrI++ {
			s.senders[i][addrI] = 0xf0 + byte(i)
		}
		s.payloads[i] = []byte{0xf0 + byte(i)}
		s.blockNumbers[i] = uint64(i)
		s.timestamps[i] = now.Add(time.Second * time.Duration(i))
	}
}

func TestModelSuite(t *testing.T) {
	suite.Run(t, new(ModelSuite))
}

//
// AddAdvanceInput
//

func (s *ModelSuite) TestItAddsAndGetsAdvanceInputs() {
	// add inputs
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}

	// get inputs
	inputs := s.m.GetInputs(InputFilter{}, 0, 100)
	s.Len(inputs, s.n)
	for i := 0; i < s.n; i++ {
		input := inputs[i]
		s.Equal(i, input.Index)
		s.Equal(CompletionStatusUnprocessed, input.Status)
		s.Equal(s.senders[i], input.MsgSender)
		s.Equal(s.payloads[i], input.Payload)
		s.Equal(s.blockNumbers[i], input.BlockNumber)
		s.Equal(s.timestamps[i], input.Timestamp)
		s.Empty(input.Vouchers)
		s.Empty(input.Notices)
		s.Empty(input.Reports)
		s.Empty(input.Exception)
	}
}

//
// AddInspectInput and GetInspectInput
//

func (s *ModelSuite) TestItAddsAndGetsInspectInput() {
	// add inputs
	for i := 0; i < s.n; i++ {
		index := s.m.AddInspectInput(s.payloads[i])
		s.Equal(i, index)
	}

	// get inputs
	for i := 0; i < s.n; i++ {
		input := s.m.GetInspectInput(i)
		s.Equal(i, input.Index)
		s.Equal(CompletionStatusUnprocessed, input.Status)
		s.Equal(s.payloads[i], input.Payload)
		s.Equal(0, input.ProccessedInputCount)
		s.Empty(input.Reports)
		s.Empty(input.Exception)
	}
}

//
// FinishAndGetNext
//

func (s *ModelSuite) TestItGetsNilWhenThereIsNoInput() {
	input := s.m.FinishAndGetNext(true)
	s.Nil(input)
}

func (s *ModelSuite) TestItGetsFirstAdvanceInput() {
	// add inputs
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}

	// get first input
	input, ok := s.m.FinishAndGetNext(true).(AdvanceInput)
	s.NotNil(input)
	s.True(ok)
	s.Equal(0, input.Index)
	s.Equal(s.payloads[0], input.Payload)
}

func (s *ModelSuite) TestItGetsFirstInspectInput() {
	// add inputs
	for i := 0; i < s.n; i++ {
		s.m.AddInspectInput(s.payloads[i])
	}

	// get first input
	input, ok := s.m.FinishAndGetNext(true).(InspectInput)
	s.NotNil(input)
	s.True(ok)
	s.Equal(0, input.Index)
	s.Equal(s.payloads[0], input.Payload)
}

func (s *ModelSuite) TestItGetsInspectBeforeAdvance() {
	// add inputs
	for i := 0; i < s.n; i++ {
		s.m.AddInspectInput(s.payloads[i])
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}

	// get inspects
	for i := 0; i < s.n; i++ {
		input, ok := s.m.FinishAndGetNext(true).(InspectInput)
		s.NotNil(input)
		s.True(ok)
		s.Equal(i, input.Index)
	}

	// get advances
	for i := 0; i < s.n; i++ {
		input, ok := s.m.FinishAndGetNext(true).(AdvanceInput)
		s.NotNil(input)
		s.True(ok)
		s.Equal(i, input.Index)
	}

	// get nil
	input := s.m.FinishAndGetNext(true)
	s.Nil(input)
}

func (s *ModelSuite) TestItFinishesAdvanceWithAccept() {
	// add input and process it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Nil(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	s.m.FinishAndGetNext(true) // finish

	// check input
	input, ok := s.m.GetAdvanceInput(0)
	s.True(ok)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusAccepted, input.Status)
	s.Len(input.Vouchers, 1)
	s.Len(input.Notices, 1)
	s.Len(input.Reports, 1)
}

func (s *ModelSuite) TestItFinishesAdvanceWithReject() {
	// add input and process it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Nil(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	s.m.FinishAndGetNext(false) // finish

	// check input
	input, ok := s.m.GetAdvanceInput(0)
	s.True(ok)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusRejected, input.Status)
	s.Empty(input.Vouchers)
	s.Empty(input.Notices)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItFinishesInspectWithAccept() {
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true) // get
	err := s.m.AddReport(s.payloads[0])
	s.Nil(err)
	s.m.FinishAndGetNext(true) // finish

	// check input
	input := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusAccepted, input.Status)
	s.Equal(s.payloads[0], input.Payload)
	s.Equal(0, input.ProccessedInputCount)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItFinishesInspectWithReject() {
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true) // get
	err := s.m.AddReport(s.payloads[0])
	s.Nil(err)
	s.m.FinishAndGetNext(false) // finish

	// check input
	input := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusRejected, input.Status)
	s.Equal(s.payloads[0], input.Payload)
	s.Equal(0, input.ProccessedInputCount)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItComputesProcessedInputCount() {
	// process n advance inputs
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		s.m.FinishAndGetNext(true) // finish
	}

	// add inspect and finish it
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true) // get
	s.m.FinishAndGetNext(true) // finish

	// check input
	input := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(s.n, input.ProccessedInputCount)
}

//
// AddVoucher
//

func (s *ModelSuite) TestItAddsVoucher() {
	// add input and get it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true)

	// add vouchers
	for i := 0; i < s.n; i++ {
		index, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
		s.Equal(i, index)
	}

	// check vouchers are not there before finish
	vouchers := s.m.GetVouchers(OutputFilter{}, 0, 100)
	s.Empty(vouchers)

	// finish input
	s.m.FinishAndGetNext(true)

	// check vouchers
	vouchers = s.m.GetVouchers(OutputFilter{}, 0, 100)
	s.Len(vouchers, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, vouchers[i].InputIndex)
		s.Equal(i, vouchers[i].Index)
		s.Equal(s.senders[i], vouchers[i].Destination)
		s.Equal(s.payloads[i], vouchers[i].Payload)
	}
}

func (s *ModelSuite) TestItFailsToAddVoucherWhenInspect() {
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true)
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Error(err)
}

func (s *ModelSuite) TestItFailsToAddVoucherWhenIdle() {
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add voucher in current state"), err)
}

//
// AddNotice
//

func (s *ModelSuite) TestItAddsNotice() {
	// add input and get it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true)

	// add notices
	for i := 0; i < s.n; i++ {
		index, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
		s.Equal(i, index)
	}

	// check notices are not there before finish
	notices := s.m.GetNotices(OutputFilter{}, 0, 100)
	s.Empty(notices)

	// finish input
	s.m.FinishAndGetNext(true)

	// check notices
	notices = s.m.GetNotices(OutputFilter{}, 0, 100)
	s.Len(notices, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, notices[i].InputIndex)
		s.Equal(i, notices[i].Index)
		s.Equal(s.payloads[i], notices[i].Payload)
	}
}

func (s *ModelSuite) TestItFailsToAddNoticeWhenInspect() {
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true)
	_, err := s.m.AddNotice(s.payloads[0])
	s.Error(err)
}

func (s *ModelSuite) TestItFailsToAddNoticeWhenIdle() {
	_, err := s.m.AddNotice(s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add notice in current state"), err)
}

//
// AddReport
//

func (s *ModelSuite) TestItAddsReportWhenAdvancing() {
	// add input and get it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true)

	// add reports
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}

	// check reports are not there before finish
	reports := s.m.GetReports(OutputFilter{}, 0, 100)
	s.Empty(reports)

	// finish input
	s.m.FinishAndGetNext(true)

	// check reports
	reports = s.m.GetReports(OutputFilter{}, 0, 100)
	s.Len(reports, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, reports[i].InputIndex)
		s.Equal(i, reports[i].Index)
		s.Equal(s.payloads[i], reports[i].Payload)
	}
}

func (s *ModelSuite) TestItAddsReportWhenInspecting() {
	// add input and get it
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true)

	// add reports
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}

	// check reports are not there before finish
	reports := s.m.GetInspectInput(0).Reports
	s.Empty(reports)

	// finish input
	s.m.FinishAndGetNext(true)

	// check reports
	reports = s.m.GetInspectInput(0).Reports
	s.Len(reports, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, reports[i].InputIndex)
		s.Equal(i, reports[i].Index)
		s.Equal(s.payloads[i], reports[i].Payload)
	}
}

func (s *ModelSuite) TestItFailsToAddReportWhenIdle() {
	err := s.m.AddReport(s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add report in current state"), err)
}

//
// RegisterException
//

func (s *ModelSuite) TestItRegistersExceptionWhenAdvancing() {
	// add input and process it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Nil(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	err = s.m.RegisterException(s.payloads[0])
	s.Nil(err)

	// check input
	input, ok := s.m.GetAdvanceInput(0)
	s.True(ok)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusException, input.Status)
	s.Empty(input.Vouchers)
	s.Empty(input.Notices)
	s.Len(input.Reports, 1)
	s.Equal(s.payloads[0], input.Exception)
}

func (s *ModelSuite) TestItRegistersExceptionWhenInspecting() {
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true) // get
	err := s.m.AddReport(s.payloads[0])
	s.Nil(err)
	err = s.m.RegisterException(s.payloads[0])
	s.Nil(err)

	// check input
	input := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusException, input.Status)
	s.Equal(s.payloads[0], input.Payload)
	s.Equal(0, input.ProccessedInputCount)
	s.Len(input.Reports, 1)
	s.Equal(s.payloads[0], input.Exception)
}

func (s *ModelSuite) TestItFailsToRegisterExceptionWhenIdle() {
	err := s.m.RegisterException(s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot register exception in current state"), err)
}

//
// GetAdvanceInput
//

func (s *ModelSuite) TestItGetsAdvanceInputs() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		input, ok := s.m.GetAdvanceInput(i)
		s.True(ok)
		s.Equal(i, input.Index)
		s.Equal(CompletionStatusUnprocessed, input.Status)
		s.Equal(s.senders[i], input.MsgSender)
		s.Equal(s.payloads[i], input.Payload)
		s.Equal(s.blockNumbers[i], input.BlockNumber)
		s.Equal(s.timestamps[i], input.Timestamp)
	}
}

func (s *ModelSuite) TestItFailsToGetAdvanceInput() {
	_, ok := s.m.GetAdvanceInput(0)
	s.False(ok)
}

//
// GetVoucher
//

func (s *ModelSuite) TestItGetsVoucher() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddVoucher(s.senders[j], s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			voucher, ok := s.m.GetVoucher(j, i)
			s.True(ok)
			s.Equal(j, voucher.Index)
			s.Equal(i, voucher.InputIndex)
			s.Equal(s.senders[j], voucher.Destination)
			s.Equal(s.payloads[j], voucher.Payload)
		}
	}
}

func (s *ModelSuite) TestItFailsToGetVoucherFromNonExistingInput() {
	_, ok := s.m.GetVoucher(0, 0)
	s.False(ok)
}

func (s *ModelSuite) TestItFailsToGetVoucherFromExistingInput() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	s.m.FinishAndGetNext(true) // finish
	_, ok := s.m.GetVoucher(0, 0)
	s.False(ok)
}

//
// GetNotice
//

func (s *ModelSuite) TestItGetsNotice() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddNotice(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			notice, ok := s.m.GetNotice(j, i)
			s.True(ok)
			s.Equal(j, notice.Index)
			s.Equal(i, notice.InputIndex)
			s.Equal(s.payloads[j], notice.Payload)
		}
	}
}

func (s *ModelSuite) TestItFailsToGetNoticeFromNonExistingInput() {
	_, ok := s.m.GetNotice(0, 0)
	s.False(ok)
}

func (s *ModelSuite) TestItFailsToGetNoticeFromExistingInput() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	s.m.FinishAndGetNext(true) // finish
	_, ok := s.m.GetNotice(0, 0)
	s.False(ok)
}

//
// GetReport
//

func (s *ModelSuite) TestItGetsReport() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			report, ok := s.m.GetReport(j, i)
			s.True(ok)
			s.Equal(j, report.Index)
			s.Equal(i, report.InputIndex)
			s.Equal(s.payloads[j], report.Payload)
		}
	}
}

func (s *ModelSuite) TestItFailsToGetReportFromNonExistingInput() {
	_, ok := s.m.GetReport(0, 0)
	s.False(ok)
}

func (s *ModelSuite) TestItFailsToGetReportFromExistingInput() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	s.m.FinishAndGetNext(true) // finish
	_, ok := s.m.GetReport(0, 0)
	s.False(ok)
}

//
// GetNumInputs
//

func (s *ModelSuite) TestItGetsNumInputs() {
	n := s.m.GetNumInputs(InputFilter{})
	s.Equal(0, n)

	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}

	n = s.m.GetNumInputs(InputFilter{})
	s.Equal(s.n, n)

	indexGreaterThan := 0
	indexLowerThan := 2
	filter := InputFilter{
		IndexGreaterThan: &indexGreaterThan,
		IndexLowerThan:   &indexLowerThan,
	}
	n = s.m.GetNumInputs(filter)
	s.Equal(1, n)
}

//
// GetNumVouchers
//

func (s *ModelSuite) TestItGetsNumVouchers() {
	n := s.m.GetNumVouchers(OutputFilter{})
	s.Equal(0, n)

	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
		s.m.FinishAndGetNext(true) // finish
	}

	n = s.m.GetNumVouchers(OutputFilter{})
	s.Equal(s.n, n)

	inputIndex := 0
	filter := OutputFilter{
		InputIndex: &inputIndex,
	}
	n = s.m.GetNumVouchers(filter)
	s.Equal(1, n)
}

//
// GetNumNotices
//

func (s *ModelSuite) TestItGetsNumNotices() {
	n := s.m.GetNumNotices(OutputFilter{})
	s.Equal(0, n)

	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
		s.m.FinishAndGetNext(true) // finish
	}

	n = s.m.GetNumNotices(OutputFilter{})
	s.Equal(s.n, n)

	inputIndex := 0
	filter := OutputFilter{
		InputIndex: &inputIndex,
	}
	n = s.m.GetNumNotices(filter)
	s.Equal(1, n)
}

//
// GetNumReports
//

func (s *ModelSuite) TestItGetsNumReports() {
	n := s.m.GetNumReports(OutputFilter{})
	s.Equal(0, n)

	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
		s.m.FinishAndGetNext(true) // finish
	}

	n = s.m.GetNumReports(OutputFilter{})
	s.Equal(s.n, n)

	inputIndex := 0
	filter := OutputFilter{
		InputIndex: &inputIndex,
	}
	n = s.m.GetNumReports(filter)
	s.Equal(1, n)
}

//
// GetInputs
//

func (s *ModelSuite) TestItGetsNoInputs() {
	inputs := s.m.GetInputs(InputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsInputs() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 0, 100)
	s.Len(inputs, s.n)
	for i := 0; i < s.n; i++ {
		input := inputs[i]
		s.Equal(i, input.Index)
	}
}

func (s *ModelSuite) TestItGetsInputsWithFilter() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	indexGreaterThan := 0
	indexLowerThan := 2
	filter := InputFilter{
		IndexGreaterThan: &indexGreaterThan,
		IndexLowerThan:   &indexLowerThan,
	}
	inputs := s.m.GetInputs(filter, 0, 100)
	s.Len(inputs, 1)
	s.Equal(1, inputs[0].Index)
}

func (s *ModelSuite) TestItGetsInputsWithOffset() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 1, 100)
	s.Len(inputs, 2)
	s.Equal(1, inputs[0].Index)
	s.Equal(2, inputs[1].Index)
}

func (s *ModelSuite) TestItGetsInputsWithLimit() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 0, 2)
	s.Len(inputs, 2)
	s.Equal(0, inputs[0].Index)
	s.Equal(1, inputs[1].Index)
}

func (s *ModelSuite) TestItGetsNoInputsWithZeroLimit() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 0, 0)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsNoInputsWhenOffsetIsGreaterThanInputs() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 3, 0)
	s.Empty(inputs)
}

//
// GetVouchers
//

func (s *ModelSuite) TestItGetsNoVouchers() {
	inputs := s.m.GetVouchers(OutputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsVouchers() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddVoucher(s.senders[j], s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	vouchers := s.m.GetVouchers(OutputFilter{}, 0, 100)
	s.Len(vouchers, s.n*s.n)
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			idx := s.n*i + j
			s.Equal(j, vouchers[idx].Index)
			s.Equal(i, vouchers[idx].InputIndex)
			s.Equal(s.senders[j], vouchers[idx].Destination)
			s.Equal(s.payloads[j], vouchers[idx].Payload)
		}
	}
}

func (s *ModelSuite) TestItGetsVouchersWithFilter() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddVoucher(s.senders[j], s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	inputIndex := 1
	filter := OutputFilter{
		InputIndex: &inputIndex,
	}
	vouchers := s.m.GetVouchers(filter, 0, 100)
	s.Len(vouchers, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, vouchers[i].Index)
		s.Equal(inputIndex, vouchers[i].InputIndex)
		s.Equal(s.senders[i], vouchers[i].Destination)
		s.Equal(s.payloads[i], vouchers[i].Payload)
	}
}

func (s *ModelSuite) TestItGetsVouchersWithOffset() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	vouchers := s.m.GetVouchers(OutputFilter{}, 1, 100)
	s.Len(vouchers, 2)
	s.Equal(1, vouchers[0].Index)
	s.Equal(2, vouchers[1].Index)
}

func (s *ModelSuite) TestItGetsVouchersWithLimit() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	vouchers := s.m.GetVouchers(OutputFilter{}, 0, 2)
	s.Len(vouchers, 2)
	s.Equal(0, vouchers[0].Index)
	s.Equal(1, vouchers[1].Index)
}

func (s *ModelSuite) TestItGetsNoVouchersWithZeroLimit() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	vouchers := s.m.GetVouchers(OutputFilter{}, 0, 0)
	s.Empty(vouchers)
}

func (s *ModelSuite) TestItGetsNoVouchersWhenOffsetIsGreaterThanInputs() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	vouchers := s.m.GetVouchers(OutputFilter{}, 0, 0)
	s.Empty(vouchers)
}

//
// GetNotices
//

func (s *ModelSuite) TestItGetsNoNotices() {
	inputs := s.m.GetNotices(OutputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsNotices() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddNotice(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	notices := s.m.GetNotices(OutputFilter{}, 0, 100)
	s.Len(notices, s.n*s.n)
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			idx := s.n*i + j
			s.Equal(j, notices[idx].Index)
			s.Equal(i, notices[idx].InputIndex)
			s.Equal(s.payloads[j], notices[idx].Payload)
		}
	}
}

func (s *ModelSuite) TestItGetsNoticesWithFilter() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddNotice(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	inputIndex := 1
	filter := OutputFilter{
		InputIndex: &inputIndex,
	}
	notices := s.m.GetNotices(filter, 0, 100)
	s.Len(notices, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, notices[i].Index)
		s.Equal(inputIndex, notices[i].InputIndex)
		s.Equal(s.payloads[i], notices[i].Payload)
	}
}

func (s *ModelSuite) TestItGetsNoticesWithOffset() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	notices := s.m.GetNotices(OutputFilter{}, 1, 100)
	s.Len(notices, 2)
	s.Equal(1, notices[0].Index)
	s.Equal(2, notices[1].Index)
}

func (s *ModelSuite) TestItGetsNoticesWithLimit() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	notices := s.m.GetNotices(OutputFilter{}, 0, 2)
	s.Len(notices, 2)
	s.Equal(0, notices[0].Index)
	s.Equal(1, notices[1].Index)
}

func (s *ModelSuite) TestItGetsNoNoticesWithZeroLimit() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	notices := s.m.GetNotices(OutputFilter{}, 0, 0)
	s.Empty(notices)
}

func (s *ModelSuite) TestItGetsNoNoticesWhenOffsetIsGreaterThanInputs() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	notices := s.m.GetNotices(OutputFilter{}, 0, 0)
	s.Empty(notices)
}

//
// GetReports
//

func (s *ModelSuite) TestItGetsNoReports() {
	inputs := s.m.GetReports(OutputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsReports() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	reports := s.m.GetReports(OutputFilter{}, 0, 100)
	s.Len(reports, s.n*s.n)
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			idx := s.n*i + j
			s.Equal(j, reports[idx].Index)
			s.Equal(i, reports[idx].InputIndex)
			s.Equal(s.payloads[j], reports[idx].Payload)
		}
	}
}

func (s *ModelSuite) TestItGetsReportsWithFilter() {
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	inputIndex := 1
	filter := OutputFilter{
		InputIndex: &inputIndex,
	}
	reports := s.m.GetReports(filter, 0, 100)
	s.Len(reports, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, reports[i].Index)
		s.Equal(inputIndex, reports[i].InputIndex)
		s.Equal(s.payloads[i], reports[i].Payload)
	}
}

func (s *ModelSuite) TestItGetsReportsWithOffset() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	reports := s.m.GetReports(OutputFilter{}, 1, 100)
	s.Len(reports, 2)
	s.Equal(1, reports[0].Index)
	s.Equal(2, reports[1].Index)
}

func (s *ModelSuite) TestItGetsReportsWithLimit() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	reports := s.m.GetReports(OutputFilter{}, 0, 2)
	s.Len(reports, 2)
	s.Equal(0, reports[0].Index)
	s.Equal(1, reports[1].Index)
}

func (s *ModelSuite) TestItGetsNoReportsWithZeroLimit() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	reports := s.m.GetReports(OutputFilter{}, 0, 0)
	s.Empty(reports)
}

func (s *ModelSuite) TestItGetsNoReportsWhenOffsetIsGreaterThanInputs() {
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	reports := s.m.GetReports(OutputFilter{}, 0, 0)
	s.Empty(reports)
}

var Bob = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
var Bruno = common.HexToAddress("0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC")
var Alice = common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
var Token = common.HexToAddress("0xc6e7DF5E7b4f2A278906862b61205850344D4e7d")

func (s *ModelSuite) TestItAddsVoucherMetadataAndFindByAddress() {
	{
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bruno,
			Contract:    Token,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bob,
			Contract:    Token,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Alice,
			Contract:    Token,
		})
	}
	filters := CreateFilterList(`[{
		"field": "Beneficiary",
		"eq": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	}]`)
	results, err := s.m.GetVouchersMetadata(filters)
	if err != nil {
		s.Fail("Unexpected error")
	}
	s.Equal(1, len(results))
	s.Equal(Bob.String(), results[0].Beneficiary.String())
}

func (s *ModelSuite) TestItAddsVoucherMetadataAndFindByInputIndex() {
	{
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bruno,
			Contract:    Token,
			InputIndex:  1,
			OutputIndex: 0,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bob,
			Contract:    Token,
			InputIndex:  2,
			OutputIndex: 0,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Alice,
			Contract:    Token,
			InputIndex:  3,
			OutputIndex: 0,
		})
	}
	filters := CreateFilterList(`[{"field": "InputIndex", "eq": "2"}]`)
	results, err := s.m.GetVouchersMetadata(filters)
	if err != nil {
		s.Fail("Unexpected error")
	}
	s.Equal(1, len(results))
	s.Equal(Bob.String(), results[0].Beneficiary.String())
	s.Equal(2, results[0].InputIndex)
}

func (s *ModelSuite) TestItAddsVoucherMetadataAndFindByExecutedAt() {
	{
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bruno,
			Contract:    Token,
			InputIndex:  1,
			OutputIndex: 0,
			ExecutedAt:  0,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bob,
			Contract:    Token,
			InputIndex:  2,
			OutputIndex: 0,
			ExecutedAt:  1234,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Alice,
			Contract:    Token,
			InputIndex:  3,
			OutputIndex: 0,
			ExecutedAt:  0,
		})
	}
	filters := CreateFilterList(`[{"field": "ExecutedAt", "gt": "0"}]`)
	results, err := s.m.GetVouchersMetadata(filters)
	if err != nil {
		s.Fail("Unexpected error")
	}
	s.Equal(1, len(results))
	s.Equal(Bob.String(), results[0].Beneficiary.String())
	s.Equal(uint64(1234), results[0].ExecutedAt)
}

func (s *ModelSuite) TestItAddsVoucherMetadataAndFindNeverExecuted() {
	{
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bruno,
			Contract:    Token,
			InputIndex:  1,
			OutputIndex: 0,
			ExecutedAt:  123,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Bob,
			Contract:    Token,
			InputIndex:  2,
			OutputIndex: 0,
			ExecutedAt:  0,
		})
		createVoucherMetadataOrFail(s, VoucherMetadata{
			Beneficiary: Alice,
			Contract:    Token,
			InputIndex:  3,
			OutputIndex: 0,
			ExecutedAt:  123,
		})
	}
	filters := CreateFilterList(`[{"field": "executedAt", "eq": "0"}]`)
	results, err := s.m.GetVouchersMetadata(filters)
	if err != nil {
		s.Fail("Unexpected error")
	}
	s.Equal(1, len(results))
	s.Equal(Bob.String(), results[0].Beneficiary.String())
	s.Equal(uint64(0), results[0].ExecutedAt)
}

func createVoucherMetadataOrFail(s *ModelSuite, voucherMetadata VoucherMetadata) {
	if err := s.m.AddVoucherMetadata(&voucherMetadata); err != nil {
		s.Fail(err.Error())
	}
}
