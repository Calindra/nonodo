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

	cModel "github.com/calindra/nonodo/internal/convenience/model"
	cRepos "github.com/calindra/nonodo/internal/convenience/repository"

	"github.com/calindra/nonodo/internal/commons"
	"github.com/calindra/nonodo/internal/convenience"
	"github.com/calindra/nonodo/internal/convenience/services"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jmoiron/sqlx"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

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
	reportRepository   *cRepos.ReportRepository
	inputRepository    *cRepos.InputRepository
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
	s.reportRepository = container.GetReportRepository()
	s.inputRepository = container.GetInputRepository()
	s.m = NewNonodoModel(
		decoder,
		s.reportRepository,
		s.inputRepository,
	)
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
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}

	// get inputs
	inputs := s.getAllInputs(0, 100)
	s.Len(inputs, s.n)
	for i := 0; i < s.n; i++ {
		input := inputs[i]
		s.Equal(i, input.Index)
		s.Equal(cModel.CompletionStatusUnprocessed, input.Status)
		s.Equal(s.senders[i], input.MsgSender)
		s.Equal(s.payloads[i], input.Payload)
		s.Equal(s.blockNumbers[i], input.BlockNumber)
		s.Equal(s.timestamps[i].UnixMilli(), input.BlockTimestamp.UnixMilli())
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
		input, _ := s.m.GetInspectInput(i)

		s.Equal(i, input.Index)
		s.Equal(cModel.CompletionStatusUnprocessed, input.Status)
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
	input, _ := s.m.FinishAndGetNext(true)
	s.Nil(input)
}

func (s *ModelSuite) TestItGetsFirstAdvanceInput() {
	defer s.teardown()
	// add inputs
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}

	// get first input
	input, _ := s.m.FinishAndGetNext(true)

	convertedInput, ok := input.(cModel.AdvanceInput)
	s.NotNil(convertedInput)
	s.True(ok)
	s.Equal(0, convertedInput.Index)
	s.Equal(s.payloads[0], convertedInput.Payload)
}

func (s *ModelSuite) TestItGetsFirstInspectInput() {
	defer s.teardown()
	// add inputs
	for i := 0; i < s.n; i++ {
		s.m.AddInspectInput(s.payloads[i])
	}

	// get first input
	input, _ := s.m.FinishAndGetNext(true)
	convertedInput, ok := input.(InspectInput)
	s.NotNil(convertedInput)
	s.True(ok)
	s.Equal(0, convertedInput.Index)
	s.Equal(s.payloads[0], convertedInput.Payload)
}

func (s *ModelSuite) TestItGetsInspectBeforeAdvance() {
	defer s.teardown()
	// add inputs
	for i := 0; i < s.n; i++ {
		s.m.AddInspectInput(s.payloads[i])
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)

		s.NoError(err)
	}

	// get inspects
	for i := 0; i < s.n; i++ {
		input, _ := s.m.FinishAndGetNext(true)
		convertedInput, ok := input.(InspectInput)
		s.NotNil(convertedInput)
		s.True(ok)
		s.Equal(i, convertedInput.Index)
	}

	// get advances
	for i := 0; i < s.n; i++ {
		input, _ := s.m.FinishAndGetNext(true)
		convertedInput, ok := input.(cModel.AdvanceInput)
		s.NotNil(convertedInput)
		s.True(ok)
		s.Equal(i, convertedInput.Index)
	}

	// get nil
	input, _ := s.m.FinishAndGetNext(true)
	s.Nil(input)
}

func (s *ModelSuite) TestItFinishesAdvanceWithAccept() {
	defer s.teardown()
	// add input and process it
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	_, err = s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.NoError(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.NoError(err)
	err = s.m.AddReport(s.payloads[0])
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	// check input
	ctx := context.Background()
	input, err := s.inputRepository.FindByIndex(ctx, 0)
	s.NoError(err)
	s.Equal(0, input.Index)
	s.Equal(cModel.CompletionStatusAccepted, input.Status)

	// check vouchers
	vouchers, err := s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(vouchers.Rows, 1)

	// check notices
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(notices.Rows, 1)

	inputIndex := 0
	reportPage, err := s.reportRepository.FindAllByInputIndex(ctx, nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, int(reportPage.Total))
}

func (s *ModelSuite) TestItFinishesAdvanceWithReject() {
	defer s.teardown()
	// add input and process it
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.Nil(err)
	_, err = s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Nil(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	_, err = s.m.FinishAndGetNext(false) // finish
	s.Nil(err)

	// check input
	ctx := context.Background()
	input, err := s.inputRepository.FindByIndex(ctx, 0)
	s.NoError(err)
	s.Equal(0, input.Index)
	s.Equal(cModel.CompletionStatusRejected, input.Status)
	s.Empty(input.Exception)
	s.Empty(input.Notices)  // deprecated
	s.Empty(input.Vouchers) // deprecated

	// check vouchers

	vouchers, err := s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(vouchers.Rows, 0)

	// check notices
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(notices.Rows, 0)

	inputIndex := 0
	page, err := s.reportRepository.FindAllByInputIndex(ctx, nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, int(page.Total))
}

func (s *ModelSuite) TestItFinishesInspectWithAccept() {
	defer s.teardown()
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	_, err := s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	err = s.m.AddReport(s.payloads[0])
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	// check input
	input, _ := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(cModel.CompletionStatusAccepted, input.Status)
	s.Equal(s.payloads[0], input.Payload)
	s.Equal(0, input.ProcessedInputCount)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItFinishesInspectWithReject() {
	defer s.teardown()
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	_, err := s.m.FinishAndGetNext(true) // get
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	_, err = s.m.FinishAndGetNext(false) // finish
	s.Nil(err)

	// check input
	input, _ := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(cModel.CompletionStatusRejected, input.Status)
	s.Equal(s.payloads[0], input.Payload)
	s.Equal(0, input.ProcessedInputCount)
	s.Len(input.Reports, 1)
	s.Empty(input.Exception)
}

func (s *ModelSuite) TestItComputesProcessedInputCount() {
	defer s.teardown()
	// process n advance inputs
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}

	// add inspect and finish it
	s.m.AddInspectInput(s.payloads[0])
	_, err := s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	// check input
	input, _ := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(s.n, input.ProcessedInputCount)
}

//
// AddVoucher
//

func (s *ModelSuite) TestItAddsVoucher() {
	defer s.teardown()
	// add input and get it
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

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
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

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
	_, err := s.m.FinishAndGetNext(true)
	s.NoError(err)
	_, err = s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Error(err)
}

func (s *ModelSuite) TestItFailsToAddVoucherWhenIdle() {
	defer s.teardown()
	_, err := s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Error(err)
	s.Equal(errors.New("cannot add voucher in idle state"), err)
}

//
// AddNotice
//

func (s *ModelSuite) TestItAddsNotice() {
	defer s.teardown()
	// add input and get it
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

	// add notices
	for i := 0; i < s.n; i++ {
		index, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
		s.Equal(i, index)
	}

	// check notices are not there before finish
	ctx := context.Background()
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(notices.Rows)

	// finish input
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

	// check notices
	notices, err = s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(notices.Rows, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, int(notices.Rows[i].InputIndex))
		s.Equal(i, int(notices.Rows[i].OutputIndex))
		s.Equal(s.payloads[i], common.Hex2Bytes(notices.Rows[i].Payload[2:]))
	}
}

func (s *ModelSuite) TestItFailsToAddNoticeWhenInspect() {
	defer s.teardown()
	s.m.AddInspectInput(s.payloads[0])
	_, err := s.m.FinishAndGetNext(true)
	s.NoError(err)
	_, err = s.m.AddNotice(s.payloads[0])
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
	ctx := context.Background()

	// add input and get it
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

	// add reports
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}

	// check reports are not there before finish
	reports, err := s.reportRepository.FindAll(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)

	// finish input
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

	// check reports
	count, err := s.reportRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(s.n, int(count))

	page, err := s.reportRepository.FindAll(ctx, nil, nil, nil, nil, nil)
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
	_, err := s.m.FinishAndGetNext(true)
	s.NoError(err)

	// add reports
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}

	// check reports are not there before finish
	reports, _ := s.m.GetInspectInput(0)
	s.Empty(reports.Reports)

	// finish input
	_, err = s.m.FinishAndGetNext(true)
	s.NoError(err)

	// check reports
	reports, _ = s.m.GetInspectInput(0)
	s.Len(reports.Reports, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(0, reports.Reports[i].InputIndex)
		s.Equal(i, reports.Reports[i].Index)
		s.Equal(s.payloads[i], reports.Reports[i].Payload)
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
	ctx := context.Background()
	// add input and process it
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.Nil(err)
	_, err = s.m.AddVoucher(s.senders[0], s.payloads[0])
	s.Nil(err)
	_, err = s.m.AddNotice(s.payloads[0])
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	err = s.m.RegisterException(s.payloads[0])
	s.Nil(err)

	// check input
	input, err := s.inputRepository.FindByIndex(ctx, 0)
	s.NoError(err)
	s.Equal(0, input.Index)
	s.Equal(cModel.CompletionStatusException, input.Status)
	s.Empty(input.Vouchers)
	s.Empty(input.Notices)
	s.Empty(input.Reports)
	s.Equal(s.payloads[0], input.Exception)

	total, err := s.reportRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(1, int(total))
}

func (s *ModelSuite) TestItRegistersExceptionWhenInspecting() {
	defer s.teardown()
	// add input and finish it
	s.m.AddInspectInput(s.payloads[0])
	_, err := s.m.FinishAndGetNext(true) // get
	s.Nil(err)
	err = s.m.AddReport(s.payloads[0])
	s.Nil(err)
	err = s.m.RegisterException(s.payloads[0])
	s.Nil(err)

	// check input
	input, _ := s.m.GetInspectInput(0)
	s.Equal(0, input.Index)
	s.Equal(cModel.CompletionStatusException, input.Status)
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
	ctx := context.Background()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		input, err := s.inputRepository.FindByIndex(ctx, i)
		s.NoError(err)
		s.Equal(i, input.Index)
		s.Equal(cModel.CompletionStatusUnprocessed, input.Status)
		s.Equal(s.senders[i], input.MsgSender)
		s.Equal(s.payloads[i], input.Payload)
		s.Equal(s.blockNumbers[i], input.BlockNumber)
		s.Equal(s.timestamps[i].UnixMilli(), input.BlockTimestamp.UnixMilli())
	}
}

func (s *ModelSuite) TestItFailsToGetAdvanceInput() {
	defer s.teardown()
	ctx := context.Background()
	input, err := s.inputRepository.FindByIndex(ctx, 0)
	s.NoError(err)
	s.Nil(input)
}

//
// GetVoucher
//

func (s *ModelSuite) TestItGetsVoucher() {
	defer s.teardown()
	ctx := context.Background()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddVoucher(s.senders[j], s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
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
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
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
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddNotice(s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			notice := s.getNotice(i, j)
			s.Equal(j, int(notice.OutputIndex))
			s.Equal(i, int(notice.InputIndex))
			s.Equal(s.payloads[j], common.Hex2Bytes(notice.Payload[2:]))
		}
	}
}

func (s *ModelSuite) TestItFailsToGetNoticeFromNonExistingInput() {
	defer s.teardown()
	notice := s.getNotice(0, 0)
	s.Nil(notice)
}

func (s *ModelSuite) TestItFailsToGetNoticeFromExistingInput() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	notice := s.getNotice(0, 0)
	s.Nil(notice)
}

//
// GetReport
//

func (s *ModelSuite) TestItGetsReport() {
	defer s.teardown()
	ctx := context.Background()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			report, err := s.reportRepository.FindByInputAndOutputIndex(
				ctx,
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
	ctx := context.Background()
	report, err := s.reportRepository.FindByInputAndOutputIndex(ctx, 0, 0)
	s.NoError(err)
	s.Nil(report)
}

func (s *ModelSuite) TestItFailsToGetReportFromExistingInput() {
	defer s.teardown()
	ctx := context.Background()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	report, err := s.reportRepository.FindByInputAndOutputIndex(ctx, 0, 0)
	s.NoError(err)
	s.Nil(report)
}

//
// GetNumInputs
//

func (s *ModelSuite) TestItGetsNumInputs() {
	defer s.teardown()
	ctx := context.Background()
	n, err := s.inputRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(0, int(n))

	for i := 0; i < s.n; i++ {
		err = s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}

	n, err = s.inputRepository.Count(ctx, nil)
	s.NoError(err)
	s.Equal(s.n, int(n))

	indexGreaterThan := "0"
	indexLowerThan := "2"
	filter := []*cModel.ConvenienceFilter{}
	field := "Index"
	filter = append(filter, &cModel.ConvenienceFilter{
		Field: &field,
		Gt:    &indexGreaterThan,
	})
	filter = append(filter, &cModel.ConvenienceFilter{
		Field: &field,
		Lt:    &indexLowerThan,
	})
	n, err = s.inputRepository.Count(ctx, filter)
	s.NoError(err)
	s.Equal(1, int(n))
}

//
// GetNumVouchers
//

func (s *ModelSuite) TestItGetsNumVouchers() {
	defer s.teardown()
	vouchers := s.getAllVouchers(0, 100, nil)
	s.Len(vouchers, 0)

	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		_, err = s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
		_, err = s.m.FinishAndGetNext(true) // finish
		s.Nil(err)
	}

	vouchers = s.getAllVouchers(0, 100, nil)
	s.Len(vouchers, s.n)

	inputIndex := 0
	vouchers = s.getAllVouchers(0, 100, &inputIndex)
	s.Len(vouchers, 1)
}

//
// GetNumNotices
//

func (s *ModelSuite) TestItGetsNumNotices() {
	defer s.teardown()
	n := s.getAllNotices(0, 100, nil)
	s.Equal(0, len(n))

	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		_, err = s.m.AddNotice(s.payloads[i])
		s.Nil(err)
		_, err = s.m.FinishAndGetNext(true) // finish
		s.Nil(err)
	}

	n = s.getAllNotices(0, 100, nil)
	s.Equal(s.n, len(n))

	inputIndex := 0
	n = s.getAllNotices(0, 100, &inputIndex)
	s.Equal(1, len(n))
}

//
// GetNumReports
//

func (s *ModelSuite) TestItGetsNumReports() {
	ctx := context.Background()
	defer s.teardown()
	inputIndex := 0
	page, err := s.reportRepository.FindAllByInputIndex(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(0, int(page.Total))

	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		err = s.m.AddReport(s.payloads[i])
		s.Nil(err)
		_, err = s.m.FinishAndGetNext(true) // finish
		s.Nil(err)
	}

	page, err = s.reportRepository.FindAllByInputIndex(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Equal(s.n, int(page.Total))
	page, err = s.reportRepository.FindAllByInputIndex(ctx, nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Equal(1, int(page.Total))
}

//
// GetInputs
//

func (s *ModelSuite) TestItGetsNoInputs() {
	defer s.teardown()
	inputs := s.getAllInputs(0, 100)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsInputs() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}
	inputs := s.getAllInputs(0, 100)
	s.Len(inputs, s.n)
	for i := 0; i < s.n; i++ {
		input := inputs[i]
		s.Equal(i, input.Index)
	}
}

func (s *ModelSuite) TestItGetsInputsWithFilter() {
	defer s.teardown()
	ctx := context.Background()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}
	indexGreaterThan := "0"
	indexLowerThan := "2"
	filter := []*cModel.ConvenienceFilter{}
	field := "Index"
	filter = append(filter, &cModel.ConvenienceFilter{
		Field: &field,
		Gt:    &indexGreaterThan,
	})
	filter = append(filter, &cModel.ConvenienceFilter{
		Field: &field,
		Lt:    &indexLowerThan,
	})
	page, err := s.inputRepository.FindAll(ctx, nil, nil, nil, nil, filter)
	s.NoError(err)
	inputs := page.Rows
	s.Len(inputs, 1)
	s.Equal(1, inputs[0].Index)
}

func (s *ModelSuite) TestItGetsInputsWithOffset() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}
	inputs := s.getAllInputs(1, 100)
	s.Len(inputs, 2)
	s.Equal(1, inputs[0].Index)
	s.Equal(2, inputs[1].Index)
}

func (s *ModelSuite) TestItGetsInputsWithLimit() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}
	inputs := s.getAllInputs(0, 2)
	s.Len(inputs, 2)
	s.Equal(0, inputs[0].Index)
	s.Equal(1, inputs[1].Index)
}

func (s *ModelSuite) TestItGetsNoInputsWithZeroLimit() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}
	inputs := s.getAllInputs(0, 0)
	s.Empty(inputs)
}

func (s *ModelSuite) TestItGetsNoInputsWhenOffsetIsGreaterThanInputs() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
	}
	inputs := s.getAllInputs(3, 100)
	s.Empty(inputs)
}

//
// GetVouchers
//

func (s *ModelSuite) TestItGetsNoVouchers() {
	defer s.teardown()
	vouchers := s.getAllVouchers(0, 100, nil)
	s.Empty(vouchers)
}

func (s *ModelSuite) TestItGetsVouchers() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddVoucher(s.senders[j], s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	vouchers := s.getAllVouchers(0, 100, nil)
	s.Len(vouchers, s.n*s.n)
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			idx := s.n*i + j
			s.Equal(j, int(vouchers[idx].OutputIndex))
			s.Equal(i, int(vouchers[idx].InputIndex))
			s.Equal(s.senders[j], vouchers[idx].Destination)
			s.Equal(s.payloads[j], common.Hex2Bytes(vouchers[idx].Payload[2:]))
		}
	}
}

func (s *ModelSuite) TestItGetsVouchersWithFilter() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddVoucher(s.senders[j], s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	inputIndex := 1
	filters := []*cModel.ConvenienceFilter{}
	field := cModel.INPUT_INDEX
	value := fmt.Sprintf("%d", inputIndex)
	filters = append(filters, &cModel.ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	})
	ctx := context.Background()
	vPage, err := s.convenienceService.FindAllVouchers(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	vouchers := vPage.Rows
	s.Len(vouchers, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, int(vouchers[i].OutputIndex))
		s.Equal(inputIndex, int(vouchers[i].InputIndex))
		s.Equal(s.senders[i], vouchers[i].Destination)
		s.Equal(s.payloads[i], common.Hex2Bytes(vouchers[i].Payload[2:]))
	}
}

func (s *ModelSuite) TestItGetsVouchersWithOffset() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	vouchers := s.getAllVouchers(1, 100, nil)
	s.Len(vouchers, 2)
	s.Equal(1, int(vouchers[0].OutputIndex))
	s.Equal(2, int(vouchers[1].OutputIndex))
}

func (s *ModelSuite) TestItGetsVouchersWithLimit() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	vouchers := s.getAllVouchers(0, 2, nil)
	s.Len(vouchers, 2)
	s.Equal(0, int(vouchers[0].OutputIndex))
	s.Equal(1, int(vouchers[1].OutputIndex))
}

func (s *ModelSuite) TestItGetsNoVouchersWithZeroLimit() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	vouchers := s.getAllVouchers(0, 0, nil)
	s.Empty(vouchers)
}

func (s *ModelSuite) TestItGetsNoVouchersWhenOffsetIsGreaterThanInputs() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddVoucher(s.senders[i], s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)

	vouchers := s.getAllVouchers(0, 0, nil)
	s.Empty(vouchers)
}

//
// GetNotices
//

func (s *ModelSuite) TestItGetsNoNotices() {
	defer s.teardown()
	ctx := context.Background()
	notices, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(notices.Rows)
}

func (s *ModelSuite) TestItGetsNotices() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddNotice(s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	ctx := context.Background()
	noticesPage, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	notices := noticesPage.Rows
	s.Len(notices, s.n*s.n)
	for i := 0; i < s.n; i++ {
		for j := 0; j < s.n; j++ {
			idx := s.n*i + j
			s.Equal(j, int(notices[idx].OutputIndex))
			s.Equal(i, int(notices[idx].InputIndex))
			s.Equal(s.payloads[j], common.Hex2Bytes(notices[idx].Payload[2:]))
		}
	}
}

func (s *ModelSuite) TestItGetsNoticesWithFilter() {
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			_, err := s.m.AddNotice(s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	inputIndex := 1
	filters := []*cModel.ConvenienceFilter{}
	field := cModel.INPUT_INDEX
	value := fmt.Sprintf("%d", inputIndex)
	filters = append(filters, &cModel.ConvenienceFilter{
		Field: &field,
		Eq:    &value,
	})
	ctx := context.Background()
	noticesPage, err := s.convenienceService.FindAllNotices(ctx, nil, nil, nil, nil, filters)
	s.NoError(err)
	notices := noticesPage.Rows
	s.Len(notices, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, int(notices[i].OutputIndex))
		s.Equal(inputIndex, int(notices[i].InputIndex))
		s.Equal(s.payloads[i], common.Hex2Bytes(notices[i].Payload[2:]))
	}
}

func (s *ModelSuite) TestItGetsNoticesWithOffset() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	afterOffset := commons.EncodeCursor(0)
	ctx := context.Background()
	noticesPage, err := s.convenienceService.
		FindAllNotices(ctx, nil, nil, &afterOffset, nil, nil)
	s.NoError(err)
	notices := noticesPage.Rows
	s.Len(notices, 2)
	s.Equal(1, int(notices[0].OutputIndex))
	s.Equal(2, int(notices[1].OutputIndex))
}

func (s *ModelSuite) TestItGetsNoticesWithLimit() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	firstLimit := 2
	ctx := context.Background()
	noticesPage, err := s.convenienceService.
		FindAllNotices(ctx, &firstLimit, nil, nil, nil, nil)
	s.NoError(err)
	notices := noticesPage.Rows
	s.Len(notices, 2)
	s.Equal(0, int(notices[0].OutputIndex))
	s.Equal(1, int(notices[1].OutputIndex))
}

func (s *ModelSuite) TestItGetsNoNoticesWithZeroLimit() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	firstLimit := 0
	ctx := context.Background()
	noticesPage, err := s.convenienceService.
		FindAllNotices(ctx, &firstLimit, nil, nil, nil, nil)
	s.NoError(err)
	notices := noticesPage.Rows
	s.Empty(notices)
}

func (s *ModelSuite) TestItGetsNoNoticesWhenOffsetIsGreaterThanInputs() {
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		_, err := s.m.AddNotice(s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	firstLimit := 0
	afterOffset := commons.EncodeCursor(0)
	ctx := context.Background()
	noticesPage, err := s.convenienceService.
		FindAllNotices(ctx, &firstLimit, nil, &afterOffset, nil, nil)
	s.NoError(err)
	notices := noticesPage.Rows
	s.Empty(notices)

	afterOffset = commons.EncodeCursor(999)
	_, err = s.convenienceService.
		FindAllNotices(ctx, nil, nil, &afterOffset, nil, nil)
	s.Errorf(err, "invalid pagination cursor")
}

//
// GetReports
//

func (s *ModelSuite) TestItGetsNoReports() {
	ctx := context.Background()
	defer s.teardown()
	reports, err := s.reportRepository.FindAll(ctx, nil, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) TestItGetsReports() {
	ctx := context.Background()
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	page, err := s.reportRepository.FindAll(ctx, nil, nil, nil, nil, nil)
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
	ctx := context.Background()
	defer s.teardown()
	for i := 0; i < s.n; i++ {
		err := s.m.AddAdvanceInput(s.senders[i], s.payloads[i], s.blockNumbers[i], s.timestamps[i], i)
		s.NoError(err)
		_, err = s.m.FinishAndGetNext(true) // get
		s.NoError(err)
		for j := 0; j < s.n; j++ {
			err := s.m.AddReport(s.payloads[j])
			s.Nil(err)
		}
		_, err = s.m.FinishAndGetNext(true) // finish
		s.NoError(err)
	}
	inputIndex := 1
	page, err := s.reportRepository.FindAllByInputIndex(ctx, nil, nil, nil, nil, &inputIndex)
	s.NoError(err)
	s.Len(page.Rows, s.n)
	for i := 0; i < s.n; i++ {
		s.Equal(i, page.Rows[i].Index)
		s.Equal(inputIndex, page.Rows[i].InputIndex)
		s.Equal(s.payloads[i], page.Rows[i].Payload)
	}
}

func (s *ModelSuite) TestItGetsReportsWithOffset() {
	ctx := context.Background()
	defer s.teardown()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n*2; i++ {
		err := s.m.AddReport(s.payloads[i%s.n])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	after := commons.EncodeCursor(3)
	page, err := s.reportRepository.FindAllByInputIndex(ctx, nil, nil, &after, nil, nil)
	s.NoError(err)
	s.Require().Len(page.Rows, 2)
	s.Equal(4, page.Rows[0].Index)
	s.Equal(5, page.Rows[1].Index)
}

func (s *ModelSuite) TestItGetsReportsWithLimit() {
	defer s.teardown()
	ctx := context.Background()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	first := 2
	page, err := s.reportRepository.FindAllByInputIndex(ctx, &first, nil, nil, nil, nil)
	s.NoError(err)
	s.Len(page.Rows, 2)
	s.Equal(0, page.Rows[0].Index)
	s.Equal(1, page.Rows[1].Index)
}

func (s *ModelSuite) TestItGetsNoReportsWithZeroLimit() {
	defer s.teardown()
	ctx := context.Background()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.NoError(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	firstLimit := 0
	reports, err := s.reportRepository.FindAll(ctx, &firstLimit, nil, nil, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) TestItGetsNoReportsWhenOffsetIsGreaterThanInputs() {
	defer s.teardown()
	ctx := context.Background()
	err := s.m.AddAdvanceInput(s.senders[0], s.payloads[0], s.blockNumbers[0], s.timestamps[0], 0)
	s.NoError(err)
	_, err = s.m.FinishAndGetNext(true) // get
	s.NoError(err)
	for i := 0; i < s.n; i++ {
		err := s.m.AddReport(s.payloads[i])
		s.Nil(err)
	}
	_, err = s.m.FinishAndGetNext(true) // finish
	s.NoError(err)
	afterOffset := commons.EncodeCursor(2)
	firstLimit := 10
	reports, err := s.reportRepository.FindAll(ctx, &firstLimit, nil, &afterOffset, nil, nil)
	s.NoError(err)
	s.Empty(reports.Rows)
}

func (s *ModelSuite) teardown() {
	defer os.RemoveAll(s.tempDir)
}

func (s *ModelSuite) getAllInputs(offset int, limit int) []cModel.AdvanceInput {
	ctx := context.Background()
	if offset != 0 {
		afterOffset := commons.EncodeCursor(offset - 1)
		vouchers, err := s.inputRepository.
			FindAll(ctx, &limit, nil, &afterOffset, nil, nil)
		s.NoError(err)
		return vouchers.Rows
	} else {
		page, err := s.inputRepository.FindAll(ctx, &limit, nil, nil, nil, nil)
		s.NoError(err)
		return page.Rows
	}
}

func (s *ModelSuite) getAllVouchers(
	offset int, limit int, inputIndex *int,
) []cModel.ConvenienceVoucher {
	ctx := context.Background()
	filters := []*cModel.ConvenienceFilter{}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	if offset != 0 {
		afterOffset := commons.EncodeCursor(offset - 1)
		vouchers, err := s.convenienceService.
			FindAllVouchers(ctx, &limit, nil, &afterOffset, nil, filters)
		s.NoError(err)
		return vouchers.Rows
	} else {
		vouchers, err := s.convenienceService.
			FindAllVouchers(ctx, &limit, nil, nil, nil, filters)
		s.NoError(err)
		return vouchers.Rows
	}
}

func (s *ModelSuite) getAllNotices(
	offset int, limit int, inputIndex *int,
) []cModel.ConvenienceNotice {
	ctx := context.Background()
	filters := []*cModel.ConvenienceFilter{}
	if inputIndex != nil {
		field := cModel.INPUT_INDEX
		value := fmt.Sprintf("%d", *inputIndex)
		filters = append(filters, &cModel.ConvenienceFilter{
			Field: &field,
			Eq:    &value,
		})
	}
	if offset != 0 {
		afterOffset := commons.EncodeCursor(offset - 1)
		notices, err := s.convenienceService.
			FindAllNotices(ctx, &limit, nil, &afterOffset, nil, filters)
		s.NoError(err)
		return notices.Rows
	} else {
		notices, err := s.convenienceService.
			FindAllNotices(ctx, &limit, nil, nil, nil, filters)
		s.NoError(err)
		return notices.Rows
	}
}

func (s *ModelSuite) getNotice(i, j int) *cModel.ConvenienceNotice {
	ctx := context.Background()
	notice, err := s.convenienceService.FindNoticeByInputAndOutputIndex(
		ctx,
		uint64(i),
		uint64(j),
	)
	s.NoError(err)
	return notice
}
