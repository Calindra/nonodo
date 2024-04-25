package model

import "github.com/ethereum/go-ethereum/common"

const EXECUTED = "Executed"
const FALSE = "false"
const DESTINATION = "Destination"
const VOUCHER_SELECTOR = "ef615e2f"
const NOTICE_SELECTOR = "c258d6e5"
const INPUT_INDEX = "InputIndex"

type ConvenienceNotice struct {
	Payload     string `db:"payload"`
	InputIndex  uint64 `db:"input_index"`
	OutputIndex uint64 `db:"output_index"`
}

// Voucher metadata type
type ConvenienceVoucher struct {
	Destination common.Address `db:"Destination"`
	Payload     string         `db:"Payload"`
	InputIndex  uint64         `db:"InputIndex"`
	OutputIndex uint64         `db:"OutputIndex"`
	Executed    bool           `db:"Executed"`

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
	Id             int64  `db:"Id"`
	TimestampAfter uint64 `db:"TimestampAfter"`
	IniCursorAfter string `db:"IniCursorAfter"`
	LogVouchersIds string `db:"LogVouchersIds"`
	EndCursorAfter string `db:"EndCursorAfter"`
}
