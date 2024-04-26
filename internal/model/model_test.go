// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"
)

//
// Test suite
//

type ModelSuite struct {
	suite.Suite
	m                  *NonodoModel
	n                  int
	payloads           [][]byte
	senders            []common.Address
	blockNumbers       []uint64
	timestamps         []time.Time
	reportRepository   *ReportRepository
	inputRepository    *InputRepository
	tempDir            string
	convenienceService *services.ConvenienceService
}

func (s *ModelSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "")
	s.tempDir = tempDir
	s.NoError(err)
	sqliteFileName := fmt.Sprintf("test%d.sqlite3", time.Now().UnixMilli())
	sqliteFileName = path.Join(tempDir, sqliteFileName)
	db := sqlx.MustConnect("sqlite3", sqliteFileName)
	container := convenience.NewContainer(*db)
	decoder := container.GetOutputDecoder()
	s.m = NewNonodoModel(decoder, db)
	s.reportRepository = s.m.reportRepository
	s.inputRepository = s.m.inputRepository
	s.convenienceService = container.GetConvenienceService()
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
	defer s.teardown()
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
	defer s.teardown()
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
		s.Equal(0, input.ProcessedInputCount)
		s.Empty(input.Reports)
		s.Empty(input.Exception)
	}
}

//
// FinishAndGetNext
//

func (s *ModelSuite) TestItGetsNilWhenThereIsNoInput() {
	defer s.teardown()
	input := s.m.FinishAndGetNext(true)
	s.Nil(input)
}

func (s *ModelSuite) TestItGetsFirstAdvanceInput() {
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
	// add input and process it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.NoError(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.NoError(err)
	err = s.m.AddReport(s.payloads[0])
	s.NoError(err)
	s.m.FinishAndGetNext(true) // finish

	// check input
	input, err := s.inputRepository.FindByIndex(0)
	s.NoError(err)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusAccepted, input.Status)

	// check vouchers
	ctx := context.Background()
	vouchers, err := s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(vouchers.Rows, 1)

	// check notices
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(notices.Rows, 1)

	inputIndex := 0
	reportPage, err := s.reportRepository.FindAllByInputIndex(nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, int(reportPage.Total))
}

func (s *ModelSuite) TestItFinishesAdvanceWithReject() {
	defer s.teardown()
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
	input, err := s.inputRepository.FindByIndex(0)
	s.NoError(err)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusRejected, input.Status)
	s.Empty(input.Exception)
	s.Empty(input.Notices)  // deprecated
	s.Empty(input.Vouchers) // deprecated

	// check vouchers
	ctx := context.Background()
	vouchers, err := s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(vouchers.Rows, 0)

	// check notices
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(notices.Rows, 0)

	inputIndex := 0
	page, err := s.reportRepository.FindAllByInputIndex(nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, int(page.Total))
}

func (s *ModelSuite) TestItFinishesInspectWithAccept() {
	defer s.teardown()
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true) // get
	err := s.m.AddReport(s.payloads[0])
	s.NoError(err)
	s.m.FinishAndGetNext(true) // finish

	// check input
	input := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(CompletionStatusAccepted, input.Status)
	s.Equal(s.payloads[0], input.Payload)
	s.Equal(0, input.ProcessedInputCount)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItFinishesInspectWithReject() {
	defer s.teardown()
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
	s.Equal(0, input.ProcessedInputCount)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItComputesProcessedInputCount() {
	defer s.teardown()
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
	s.Equal(s.n, input.ProcessedInputCount)
}

//
// AddVoucher
//

func (s *ModelSuite) TestItAddsVoucher() {
	defer s.teardown()
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
	ctx := context.Background()
	vouchers, err := s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(vouchers.Rows)

	// finish input
	s.m.FinishAndGetNext(true)

	// check vouchers
	vouchers, err = s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(vouchers.Rows, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, int(vouchers.Rows[i].InputIndex))
		s.Equal(i, int(vouchers.Rows[i].OutputIndex))
		s.Equal(s.senders[i], vouchers.Rows[i].Destination)
		s.Equal(s.payloads[i], common.Hex2Bytes(vouchers.Rows[i].Payload[2:]))
	}
}

func (s *ModelSuite) TestItFailsToAddVoucherWhenInspect() {
	defer s.teardown()
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true)
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Error(err)
}

func (s *ModelSuite) TestItFailsToAddVoucherWhenIdle() {
	defer s.teardown()
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add voucher in current state"), err)
}

//
// AddNotice
//

func (s *ModelSuite) TestItAddsNotice() {
	defer s.teardown()
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
	defer s.teardown()
	s.m.AddInspectInput(s.payloads[0])
	s.m.FinishAndGetNext(true)
	_, err := s.m.AddNotice(s.payloads[0])
	s.Error(err)
}

func (s *ModelSuite) TestItFailsToAddNoticeWhenIdle() {
	defer s.teardown()
	_, err := s.m.AddNotice(s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add notice in current state"), err)
}

//
// AddReport
//

func (s *ModelSuite) TestItAddsReportWhenAdvancing() {
	defer s.teardown()
	// add input and get it
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true)

	// add reports
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}

	// check reports are not there before finish
	reports, err := s.reportRepository.FindAll(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)

	// finish input
	s.m.FinishAndGetNext(true)

	// check reports
	count, err := s.reportRepository.Count(nil)
	s.NoError(err)
	s.Equal(s.n, int(count))

	page, err := s.reportRepository.FindAll(nil, nil, nil, nil, nil)
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		s.Equal(0, page.Rows[i].InputIndex)
		s.Equal(i, page.Rows[i].Index)
		s.Equal(s.payloads[i], page.Rows[i].Payload)
	}
}

func (s *ModelSuite) TestItAddsReportWhenInspecting() {
	defer s.teardown()
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
	defer s.teardown()
	err := s.m.AddReport(s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add report in current state"), err)
}

//
// RegisterException
//

func (s *ModelSuite) TestItRegistersExceptionWhenAdvancing() {
	defer s.teardown()
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
	defer s.teardown()
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
	s.Equal(0, input.ProcessedInputCount)
	s.Len(input.Reports, 1)
	s.Equal(s.payloads[0], input.Exception)
}

func (s *ModelSuite) TestItFailsToRegisterExceptionWhenIdle() {
	defer s.teardown()
	err := s.m.RegisterException(s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot register exception in current state"), err)
}

//
// GetAdvanceInput
//

func (s *ModelSuite) TestItGetsAdvanceInputs() {
	defer s.teardown()
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
	defer s.teardown()
	_, ok := s.m.GetAdvanceInput(0)
	s.False(ok)
}

//
// GetVoucher
//

func (s *ModelSuite) TestItGetsVoucher() {
	defer s.teardown()
	ctx := context.Background()
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
			voucher, err := s.convenienceService.
				FindVoucherByInputAndOutputIndex(ctx, uint64(i), uint64(j))
			s.NoError(err)
			s.Equal(j, int(voucher.OutputIndex))
			s.Equal(i, int(voucher.InputIndex))
			s.Equal(s.senders[j].Hex(), voucher.Destination.Hex())
			s.Equal(s.payloads[j], common.Hex2Bytes(voucher.Payload[2:]))
		}
	}
}

func (s *ModelSuite) TestItFailsToGetVoucherFromNonExistingInput() {
	defer s.teardown()
	ctx := context.Background()
	voucher, err := s.convenienceService.
		FindVoucherByInputAndOutputIndex(ctx, uint64(0), uint64(0))
	s.NoError(err)
	s.Nil(voucher)
}

func (s *ModelSuite) TestItFailsToGetVoucherFromExistingInput() {
	defer s.teardown()
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	s.m.FinishAndGetNext(true) // finish
	ctx := context.Background()
	voucher, err := s.convenienceService.
		FindVoucherByInputAndOutputIndex(ctx, uint64(0), uint64(0))
	s.NoError(err)
	s.Nil(voucher)
}

//
// GetNotice
//

func (s *ModelSuite) TestItGetsNotice() {
	defer s.teardown()
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
	defer s.teardown()
	_, ok := s.m.GetNotice(0, 0)
	s.False(ok)
}

func (s *ModelSuite) TestItFailsToGetNoticeFromExistingInput() {
	defer s.teardown()
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
	defer s.teardown()
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
			report, err := s.reportRepository.FindByInputAndOutputIndex(
				uint64(i),
				uint64(j),
			)
			s.NoError(err)
			s.Equal(j, report.Index)
			s.Equal(i, report.InputIndex)
			s.Equal(s.payloads[j], report.Payload)
		}
	}
}

func (s *ModelSuite) TestItFailsToGetReportFromNonExistingInput() {
	defer s.teardown()
	report, err := s.reportRepository.FindByInputAndOutputIndex(0, 0)
	s.NoError(err)
	s.Nil(report)
}

func (s *ModelSuite) TestItFailsToGetReportFromExistingInput() {
	defer s.teardown()
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	s.m.FinishAndGetNext(true) // finish
	report, err := s.reportRepository.FindByInputAndOutputIndex(0, 0)
	s.NoError(err)
	s.Nil(report)
}

//
// GetNumInputs
//

func (s *ModelSuite) TestItGetsNumInputs() {
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
	inputIndex := 0
	page, err := s.reportRepository.FindAllByInputIndex(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(0, int(page.Total))

	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
		s.m.FinishAndGetNext(true) // finish
	}

	page, err = s.reportRepository.FindAllByInputIndex(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(s.n, int(page.Total))
	page, err = s.reportRepository.FindAllByInputIndex(nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, int(page.Total))
}

//
// GetInputs
//

func (s *ModelSuite) TestItGetsNoInputs() {
	defer s.teardown()
	inputs := s.m.GetInputs(InputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsInputs() {
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 1, 100)
	s.Len(inputs, 2)
	s.Equal(1, inputs[0].Index)
	s.Equal(2, inputs[1].Index)
}

func (s *ModelSuite) TestItGetsInputsWithLimit() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 0, 2)
	s.Len(inputs, 2)
	s.Equal(0, inputs[0].Index)
	s.Equal(1, inputs[1].Index)
}

func (s *ModelSuite) TestItGetsNoInputsWithZeroLimit() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
	}
	inputs := s.m.GetInputs(InputFilter{}, 0, 0)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsNoInputsWhenOffsetIsGreaterThanInputs() {
	defer s.teardown()
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
	defer s.teardown()
	inputs := s.m.GetVouchers(OutputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsVouchers() {
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
	inputs := s.m.GetNotices(OutputFilter{}, 0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsNotices() {
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
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
	defer s.teardown()
	reports, err := s.reportRepository.FindAll(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) TestItGetsReports() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i])
		s.m.FinishAndGetNext(true) // get
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		s.m.FinishAndGetNext(true) // finish
	}
	page, err := s.reportRepository.FindAll(nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(page.Rows, s.n*s.n)
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			idx := s.n*i + j
			s.Equal(j, page.Rows[idx].Index)
			s.Equal(i, page.Rows[idx].InputIndex)
			s.Equal(s.payloads[j], page.Rows[idx].Payload)
		}
	}
}

func (s *ModelSuite) TestItGetsReportsWithFilter() {
	defer s.teardown()
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
	page, err := s.reportRepository.FindAllByInputIndex(nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Len(page.Rows, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, page.Rows[i].Index)
		s.Equal(inputIndex, page.Rows[i].InputIndex)
		s.Equal(s.payloads[i], page.Rows[i].Payload)
	}
}

func (s *ModelSuite) TestItGetsReportsWithOffset() {
	defer s.teardown()
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n*2; i++ {
		err := s.m.AddReport(s.payloads[i%s.n])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	after := commons.EncodeCursor(3)
	page, err := s.reportRepository.FindAllByInputIndex(nil, nil, &after, nil, nil)
	s.NoError(err)
	s.Require().Len(page.Rows, 2)
	s.Equal(4, page.Rows[0].Index)
	s.Equal(5, page.Rows[1].Index)
}

func (s *ModelSuite) TestItGetsReportsWithLimit() {
	defer s.teardown()
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	first := 2
	page, err := s.reportRepository.FindAllByInputIndex(&first, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(page.Rows, 2)
	s.Equal(0, page.Rows[0].Index)
	s.Equal(1, page.Rows[1].Index)
}

func (s *ModelSuite) TestItGetsNoReportsWithZeroLimit() {
	defer s.teardown()
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.NoError(err)
	}
	s.m.FinishAndGetNext(true) // finish
	firstLimit := 0
	reports, err := s.reportRepository.FindAll(&firstLimit, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) TestItGetsNoReportsWhenOffsetIsGreaterThanInputs() {
	defer s.teardown()
	s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0])
	s.m.FinishAndGetNext(true) // get
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	s.m.FinishAndGetNext(true) // finish

	afterOffset := commons.EncodeCursor(2)
	firstLimit := 10
	reports, err := s.reportRepository.FindAll(&firstLimit, nil, &afterOffset, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) teardown() {
	defer os.RemoveAll(s.tempDir)
}
