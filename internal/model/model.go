// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// The nonodo model uses a shared-memory paradigm to synchronize between threads.
package model

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/calindra/nonodo/internal/convenience/model"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/jmoiron/sqlx"
)

// Nonodo model shared among the internal workers.
// The model store inputs as pointers because these pointers are shared with the rollup state.
type NonodoModel struct {
	mutex            sync.Mutex
	advances         []*AdvanceInput
	inspects         []*InspectInput
	state            rollupsState
	decoder          Decoder
	reportRepository *ReportRepository
	inputRepository  *InputRepository
}

// Create a new model.
func NewNonodoModel(decoder Decoder, db *sqlx.DB) *NonodoModel {
	reportRepository := ReportRepository{Db: db}
	err := reportRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	inputRepository := InputRepository{Db: db}
	err = inputRepository.CreateTables()
	if err != nil {
		panic(err)
	}
	return &NonodoModel{
		state:            &rollupsStateIdle{},
		decoder:          decoder,
		reportRepository: &reportRepository,
		inputRepository:  &inputRepository,
	}
}

//
// Methods for Inputter
//

// Add an advance input to the model.
func (m *NonodoModel) AddAdvanceInput(
	sender common.Address,
	payload []byte,
	blockNumber uint64,
	timestamp time.Time,
) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	index, err := m.inputRepository.Count(nil)
	if err != nil {
		panic(err)
	}
	input := AdvanceInput{
		Index:       int(index),
		Status:      CompletionStatusUnprocessed,
		MsgSender:   sender,
		Payload:     payload,
		Timestamp:   timestamp,
		BlockNumber: blockNumber,
	}
	m.advances = append(m.advances, &input)
	_, err = m.inputRepository.Create(input)
	if err != nil {
		panic(err)
	}
	slog.Info("nonodo: added advance input", "index", input.Index, "sender", input.MsgSender,
		"payload", hexutil.Encode(input.Payload))
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
		Status:  CompletionStatusUnprocessed,
		Payload: payload,
	}
	m.inspects = append(m.inspects, &input)
	slog.Info("nonodo: added inspect input", "index", input.Index,
		"payload", hexutil.Encode(input.Payload))

	return index
}

// Get the inspect input from the model.
func (m *NonodoModel) GetInspectInput(index int) InspectInput {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if index >= len(m.inspects) {
		panic(fmt.Sprintf("invalid inspect input index: %v", index))
	}
	return *m.inspects[index]
}

//
// Methods for Rollups
//

// Finish the current input and get the next one.
// If there is no input to be processed return nil.
func (m *NonodoModel) FinishAndGetNext(accepted bool) Input {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// finish current input
	var status CompletionStatus
	if accepted {
		status = CompletionStatusAccepted
	} else {
		status = CompletionStatusRejected
	}
	m.state.finish(status)

	// try to get first unprocessed inspect
	for _, input := range m.inspects {
		if input.Status == CompletionStatusUnprocessed {
			m.state = newRollupsStateInspect(input, m.getProcessedInputCount)
			return *input
		}
	}

	// try to get first unprocessed advance
	input, err := m.inputRepository.FindByStatus(CompletionStatusUnprocessed)
	if err != nil {
		panic(err)
	}
	if input != nil {
		m.state = newRollupsStateAdvance(
			input,
			m.decoder,
			m.reportRepository,
			m.inputRepository,
		)
		return *input
	}

	// if no input was found, set state to idle
	m.state = newRollupsStateIdle()
	return nil
}

// Add a voucher to the model.
// Return the voucher index within the input.
// Return an error if the state isn't advance.
func (m *NonodoModel) AddVoucher(destination common.Address, payload []byte) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.addVoucher(destination, payload)
}

// Add a notice to the model.
// Return the notice index within the input.
// Return an error if the state isn't advance.
func (m *NonodoModel) AddNotice(payload []byte) (int, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.addNotice(payload)
}

// Add a report to the model.
// Return an error if the state isn't advance or inspect.
func (m *NonodoModel) AddReport(payload []byte) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.state.addReport(payload)
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
// Methods for Reader
//

// Get the advance input for the given index.
// Return false if not found.
func (m *NonodoModel) GetAdvanceInput(index int) (AdvanceInput, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if index >= len(m.advances) {
		var input AdvanceInput
		return input, false
	}
	return *m.advances[index], true
}

// Get the number of inputs given the filter.
func (m *NonodoModel) GetNumInputs(filter InputFilter) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n := 0
	for _, input := range m.advances {
		if !filter.Filter(input) {
			n++
		}
	}
	return n
}

// Get the inputs given the filter and pagination parameters.
func (m *NonodoModel) GetInputs(filter InputFilter, offset int, limit int) []AdvanceInput {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var inputs []AdvanceInput
	for _, input := range m.advances {
		if !filter.Filter(input) {
			inputs = append(inputs, *input)
		}
	}
	return paginate(inputs, offset, limit)
}

//
// Auxiliary Methods
//

func (m *NonodoModel) getProcessedInputCount() int {
	filter := []*model.ConvenienceFilter{}
	field := "Status"
	value := fmt.Sprintf("%d", CompletionStatusUnprocessed)
	filter = append(filter, &model.ConvenienceFilter{
		Field: &field,
		Ne:    &value,
	})
	total, err := m.inputRepository.Count(filter)
	if err != nil {
		panic(err)
	}
	return int(total)
}

func paginate[T any](slice []T, offset int, limit int) []T {
	if offset >= len(slice) {
		return nil
	}
	upperBound := offset + limit
	if upperBound > len(slice) {
		upperBound = len(slice)
	}
	return slice[offset:upperBound]
}
