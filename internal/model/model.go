// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

// The nonodo model uses a shared-memory paradigm to synchronize between threads.
package model

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Nonodo model shared among the internal workers.
// The model store inputs as pointers because these pointers are shared with the rollup state.
type NonodoModel struct {
	mutex            sync.Mutex
	advances         []*AdvanceInput
	inspects         []*InspectInput
	vouchersMetadata []*VoucherMetadata
	state            rollupsState
}

// Create a new model.
func NewNonodoModel() *NonodoModel {
	return &NonodoModel{
		state: &rollupsStateIdle{},
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

	index := len(m.advances)
	input := AdvanceInput{
		Index:       index,
		Status:      CompletionStatusUnprocessed,
		MsgSender:   sender,
		Payload:     payload,
		Timestamp:   timestamp,
		BlockNumber: blockNumber,
	}
	m.advances = append(m.advances, &input)
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
			m.state = newRollupsStateInspect(input, m.getProccessedInputCount)
			return *input
		}
	}

	// try to get first unprocessed advance
	for _, input := range m.advances {
		if input.Status == CompletionStatusUnprocessed {
			m.state = newRollupsStateAdvance(input)
			return *input
		}
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

func (m *NonodoModel) AddVoucherMetadata(voucherMetadata *VoucherMetadata) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.vouchersMetadata = append(m.vouchersMetadata, voucherMetadata)
	slog.Info("Added Vouchers metadata", "len", len(m.vouchersMetadata))
	return nil
}

func (m *NonodoModel) GetVouchersMetadata(filters []*MetadataFilter) ([]*VoucherMetadata, error) {
	result := []*VoucherMetadata{}
	slog.Info("Filtering Vouchers metadata", "len", len(m.vouchersMetadata))
	for _, voucherMetadata := range m.vouchersMetadata {
		if passFilters(voucherMetadata, filters) {
			result = append(result, voucherMetadata)
		}
	}
	return result, nil
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

// Get the voucher for the given index and input index.
// Return false if not found.
func (m *NonodoModel) GetVoucher(voucherIndex, inputIndex int) (Voucher, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if inputIndex >= len(m.advances) ||
		voucherIndex >= len(m.advances[inputIndex].Vouchers) {
		var voucher Voucher
		return voucher, false
	}
	return m.advances[inputIndex].Vouchers[voucherIndex], true
}

// Get the notice for the given index and input index.
// Return false if not found.
func (m *NonodoModel) GetNotice(noticeIndex, inputIndex int) (Notice, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if inputIndex >= len(m.advances) ||
		noticeIndex >= len(m.advances[inputIndex].Notices) {
		var notice Notice
		return notice, false
	}
	return m.advances[inputIndex].Notices[noticeIndex], true
}

// Get the report for the given index and input index.
// Return false if not found.
func (m *NonodoModel) GetReport(reportIndex, inputIndex int) (Report, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if inputIndex >= len(m.advances) ||
		reportIndex >= len(m.advances[inputIndex].Reports) {
		var report Report
		return report, false
	}
	return m.advances[inputIndex].Reports[reportIndex], true
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

// Get the number of vouchers given the filter.
func (m *NonodoModel) GetNumVouchers(filter OutputFilter) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n := 0
	for _, input := range m.advances {
		for _, voucher := range input.Vouchers {
			if !filter.Filter(voucher) {
				n++
			}
		}
	}
	return n
}

// Get the number of notices given the filter.
func (m *NonodoModel) GetNumNotices(filter OutputFilter) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n := 0
	for _, input := range m.advances {
		for _, notice := range input.Notices {
			if !filter.Filter(notice) {
				n++
			}
		}
	}
	return n
}

// Get the number of reports given the filter.
func (m *NonodoModel) GetNumReports(filter OutputFilter) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n := 0
	for _, input := range m.advances {
		for _, report := range input.Reports {
			if !filter.Filter(report) {
				n++
			}
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

// Get the vouchers given the filter and pagination parameters.
func (m *NonodoModel) GetVouchers(filter OutputFilter, offset int, limit int) []Voucher {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var vouchers []Voucher
	for _, input := range m.advances {
		for _, voucher := range input.Vouchers {
			if !filter.Filter(voucher) {
				vouchers = append(vouchers, voucher)
			}
		}
	}
	return paginate(vouchers, offset, limit)
}

// Get the notices given the filter and pagination parameters.
func (m *NonodoModel) GetNotices(filter OutputFilter, offset int, limit int) []Notice {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var notices []Notice
	for _, input := range m.advances {
		for _, notice := range input.Notices {
			if !filter.Filter(notice) {
				notices = append(notices, notice)
			}
		}
	}
	return paginate(notices, offset, limit)
}

// Get the reports given the filter and pagination parameters.
func (m *NonodoModel) GetReports(filter OutputFilter, offset int, limit int) []Report {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var reports []Report
	for _, input := range m.advances {
		for _, report := range input.Reports {
			if !filter.Filter(report) {
				reports = append(reports, report)
			}
		}
	}
	return paginate(reports, offset, limit)
}

//
// Auxiliary Methods
//

func (m *NonodoModel) getProccessedInputCount() int {
	n := 0
	for _, input := range m.advances {
		if input.Status == CompletionStatusUnprocessed {
			break
		}
		n++
	}
	return n
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

func passFilters(voucherMetadata *VoucherMetadata, filterList []*MetadataFilter) bool {
	for _, filter := range filterList {
		if filter.Eq != nil && !fieldValueMatches(voucherMetadata,
			filter.Field, *filter.Eq) {
			return false
		} else if filter.Gt != nil &&
			!fieldValueGt(voucherMetadata, filter.Field, *filter.Gt) {
			return false
		}
	}
	return true
}

func isIntegerType(t reflect.Type) bool {
	return t == reflect.TypeOf(int(0)) ||
		t == reflect.TypeOf(int8(0)) ||
		t == reflect.TypeOf(int16(0)) ||
		t == reflect.TypeOf(int32(0)) ||
		t == reflect.TypeOf(int64(0))
}

func fieldValueMatches(data interface{}, fieldName string, compareValue interface{}) bool {
	value := reflect.ValueOf(data).Elem()
	field := value.FieldByName(capitalizeFirstLetter(fieldName))
	if !field.IsValid() {
		slog.Warn(fmt.Sprintf("Field with the given name does not exist: %s", fieldName))
		return false
	}
	fieldType := field.Type()
	if isIntegerType(fieldType) {
		intValue, err := toInt64(compareValue)
		if err != nil {
			slog.Error(err.Error())
			return false
		}
		return intValue == field.Int()
	} else if address, ok := field.Interface().(common.Address); ok {
		return address.String() == compareValue
	} else if fieldType.Kind() == reflect.Uint64 {
		num, err := toUint64(compareValue)
		if err != nil {
			slog.Error(err.Error())
			return false
		}
		return num == field.Interface().(uint64)
	}
	return reflect.DeepEqual(field.Interface(), compareValue)
}

func fieldValueGt(data interface{}, fieldName string, compareValue interface{}) bool {
	value := reflect.ValueOf(data).Elem()
	field := value.FieldByName(capitalizeFirstLetter(fieldName))
	if !field.IsValid() {
		slog.Warn(fmt.Sprintf("Field with the given name does not exist: %s", fieldName))
		return false
	}
	fieldType := field.Type()
	if isIntegerType(fieldType) {
		intValue, err := toInt64(compareValue)
		if err != nil {
			slog.Error(err.Error())
			return false
		}
		return intValue > field.Int()
	} else if fieldType.Kind() == reflect.Uint64 {
		num, err := toUint64(compareValue)
		if err != nil {
			slog.Error(err.Error())
			return false
		}
		return num < field.Interface().(uint64)
	} else {
		slog.Warn(fmt.Sprintf("Checking %s not a integer type\n", fieldName))
	}
	return false
}

func toInt64(compareValue interface{}) (int64, error) {
	strValue, ok := compareValue.(string)
	if !ok {
		return 0, errors.New("conversion to string failed")
	}
	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		return 0, errors.New("conversion to int64 failed")
	}
	return intValue, nil
}

func toUint64(compareValue interface{}) (uint64, error) {
	strValue, ok := compareValue.(string)
	if !ok {
		return 0, errors.New("conversion to string failed")
	}
	num, err := strconv.ParseUint(strValue, 10, 64)
	if err != nil {
		return 0, errors.New("conversion to uint64 failed")
	}
	return num, nil
}

func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	firstChar := strings.ToUpper(s[:1])
	return firstChar + s[1:]
}
