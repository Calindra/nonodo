package model

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const EXECUTED = "Executed"
const FALSE = "false"
const DESTINATION = "Destination"
const VOUCHER_SELECTOR = "ef615e2f"
const NOTICE_SELECTOR = "c258d6e5"
const INPUT_INDEX = "InputIndex"

// Completion status for inputs.
type CompletionStatus int

const (
	CompletionStatusUnprocessed CompletionStatus = iota
	CompletionStatusAccepted
	CompletionStatusRejected
	CompletionStatusException
)

type ConvenienceNotice struct {
	Payload     string `db:"payload"`
	InputIndex  uint64 `db:"input_index"`
	OutputIndex uint64 `db:"output_index"`
}

// Voucher metadata type
type ConvenienceVoucher struct {
	Destination common.Address `db:"destination"`
	Payload     string         `db:"payload"`
	InputIndex  uint64         `db:"input_index"`
	OutputIndex uint64         `db:"output_index"`
	Executed    bool           `db:"executed"`

	// Proof we can fetch from the original GraphQL

	// future improvements
	// Contract        common.Address
	// Beneficiary     common.Address
	// Label           string
	// Amount          uint64
	// ExecutedAt      uint64
	// ExecutedBlock   uint64
	// InputIndex      int
	// OutputIndex     int
	// MethodSignature string
	// ERCX            string
}

type ConvenienceFilter struct {
	Field *string              `json:"field,omitempty"`
	Eq    *string              `json:"eq,omitempty"`
	Ne    *string              `json:"ne,omitempty"`
	Gt    *string              `json:"gt,omitempty"`
	Gte   *string              `json:"gte,omitempty"`
	Lt    *string              `json:"lt,omitempty"`
	Lte   *string              `json:"lte,omitempty"`
	In    []*string            `json:"in,omitempty"`
	Nin   []*string            `json:"nin,omitempty"`
	And   []*ConvenienceFilter `json:"and,omitempty"`
	Or    []*ConvenienceFilter `json:"or,omitempty"`
}

type SynchronizerFetch struct {
	Id             int64  `db:"id"`
	TimestampAfter uint64 `db:"timestamp_after"`
	IniCursorAfter string `db:"ini_cursor_after"`
	LogVouchersIds string `db:"log_vouchers_ids"`
	EndCursorAfter string `db:"end_cursor_after"`
}

// Rollups input, which can be advance or inspect.
type Input interface{}

// Rollups report type.
type Report struct {
	Index      int
	InputIndex int
	Payload    []byte
}

// Rollups advance input type.
type AdvanceInput struct {
	Index          int
	Status         CompletionStatus
	MsgSender      common.Address
	Payload        []byte
	BlockNumber    uint64
	BlockTimestamp time.Time
	PrevRandao     string
	Vouchers       []ConvenienceVoucher
	Notices        []ConvenienceNotice
	Reports        []Report
	Exception      []byte
}
