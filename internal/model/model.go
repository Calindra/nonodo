// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// The nonodo model uses a shared-memory paradigm to synchronize between threads.
package model

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	cModel "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/model"
	cRepos "github.com/calindra/cartesi-rollups-hl-graphql/pkg/convenience/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Nonodo model shared among the internal workers.
// The model store inputs as pointers because these pointers are shared with the rollup state.
type NonodoModel struct {
	mutex             sync.Mutex
	inspects          []*InspectInput
	state             rollupsState
	decoder           Decoder
	reportRepository  *cRepos.ReportRepository
	inputRepository   *cRepos.InputRepository
	voucherRepository *cRepos.VoucherRepository
	noticeRepository  *cRepos.NoticeRepository
}

func (m *NonodoModel) GetInputRepository() *cRepos.InputRepository {
	return m.inputRepository
}

// Create a new model.
func NewNonodoModel(
	decoder Decoder,
	reportRepository *cRepos.ReportRepository,
	inputRepository *cRepos.InputRepository,
	voucherRepository *cRepos.VoucherRepository,
	noticeRepository *cRepos.NoticeRepository,
) *NonodoModel {
	return &NonodoModel{
		state:             &rollupsStateIdle{},
		decoder:           decoder,
		reportRepository:  reportRepository,
		inputRepository:   inputRepository,
		voucherRepository: voucherRepository,
		noticeRepository:  noticeRepository,
	}
}

//
// Methods for Inputter
//

// Add an advance input to the model.
func (m *NonodoModel) AddAdvanceInput(
	sender common.Address,
	payload string,
	blockNumber uint64,
	timestamp time.Time,
	inputBoxIndex int,
	prevRandao string,
	appContract common.Address,
	chainId string,
) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ctx := context.Background()
	index, err := m.inputRepository.Count(ctx, nil)
	if err != nil {
		return err
	}
	input := cModel.AdvanceInput{
		ID:                     strconv.Itoa(inputBoxIndex),
		Index:                  int(index),
		Status:                 cModel.CompletionStatusUnprocessed,
		MsgSender:              sender,
		Payload:                payload,
		BlockTimestamp:         timestamp,
		BlockNumber:            blockNumber,
		EspressoBlockNumber:    -1,
		EspressoBlockTimestamp: time.Unix(-1, 0),
		InputBoxIndex:          inputBoxIndex,
		PrevRandao:             prevRandao,
		ChainId:                chainId,
		AppContract:            appContract,
	}

	_, err = m.inputRepository.Create(ctx, input)
	if err != nil {
		return err
	}
	slog.Info("nonodo: added advance input", "index", input.Index, "sender", input.MsgSender,
		"payload", input.Payload)
	return nil
}

//
// Methods for Inspector
//

// Add an inspect input to the model.
// Return the inspect input index that should be used for polling.
func (m *NonodoModel) AddInspectInput(payload []byte) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	index := len(m.inspects)
	input := InspectInput{
		Index:   index,
		Status:  cModel.CompletionStatusUnprocessed,
		Payload: payload,
	}
	m.inspects = append(m.inspects, &input)
	slog.Info("nonodo: added inspect input", "index", input.Index,
		"payload", hexutil.Encode(input.Payload))

	return index
}

// Get the inspect input from the model.
func (m *NonodoModel) GetInspectInput(index int) (InspectInput, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if index >= len(m.inspects) {
		slog.Error(fmt.Sprintf("invalid inspect input index: %v", index))
		return InspectInput{}, fmt.Errorf("invalid inspect input index: %v", index)
	}
	return *m.inspects[index], nil
}

//
// Methods for Rollups
//

// Finish the current input and get the next one.
// If there is no input to be processed return nil.
//
// Note: use in v2 the sequencer instead.
func (m *NonodoModel) FinishAndGetNext(accepted bool) (cModel.Input, error) {
	ctx := context.Background()
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// finish current input
	var status cModel.CompletionStatus
	if accepted {
		status = cModel.CompletionStatusAccepted
	} else {
		status = cModel.CompletionStatusRejected
	}
	err := m.state.finish(status)

	if err != nil {
		return nil, err
	}

	// try to get first unprocessed inspect
	for _, input := range m.inspects {
		if input.Status == cModel.CompletionStatusUnprocessed {
			m.state = newRollupsStateInspect(input, m.getProcessedInputCount)
			return *input, nil
		}
	}

	// try to get first unprocessed advance
	input, err := m.inputRepository.FindByStatus(ctx, cModel.CompletionStatusUnprocessed)

	if err != nil {
		return nil, err
	}
	if input != nil {
		m.state = newRollupsStateAdvance(
			input,
			m.decoder,
			m.reportRepository,
			m.inputRepository,
			m.voucherRepository,
			m.noticeRepository,
		)
		return *input, nil
	}

	// if no input was found, set state to idle
	m.state = newRollupsStateIdle()
	return nil, nil
}

// Add a voucher to the model.
// Return the voucher index within the input.
// Return an error if the state isn't advance.
func (m *NonodoModel) AddVoucher(appAddress common.Address, destination common.Address, value string, payload []byte) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.addVoucher(appAddress, destination, value, payload)
}

// Add a notice to the model.
// Return the notice index within the input.
// Return an error if the state isn't advance.
func (m *NonodoModel) AddNotice(payload []byte, appAddress common.Address) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.addNotice(payload, appAddress)
}

// Add a report to the model.
// Return an error if the state isn't advance or inspect.
func (m *NonodoModel) AddReport(appAddress common.Address, payload []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.addReport(appAddress, payload)
}

// Finish the current input with an exception.
// Return an error if the state isn't advance or inspect.
func (m *NonodoModel) RegisterException(payload []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	err := m.state.registerException(payload)
	if err != nil {
		return err
	}

	// set state to idle
	m.state = newRollupsStateIdle()
	return nil
}

//
// Auxiliary Methods
//

func (m *NonodoModel) getProcessedInputCount() (int, error) {
	ctx := context.Background()
	filter := []*cModel.ConvenienceFilter{}
	field := "Status"
	value := fmt.Sprintf("%d", cModel.CompletionStatusUnprocessed)
	filter = append(filter, &cModel.ConvenienceFilter{
		Field: &field,
		Ne:    &value,
	})
	total, err := m.inputRepository.Count(ctx, filter)

	if err != nil {
		return -1, err
	}

	return int(total), nil
}
