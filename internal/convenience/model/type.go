package model

import "github.com/ethereum/go-ethereum/common"

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
	Id             int64  `db:"id"`
	TimestampAfter uint64 `db:"timestamp_after"`
	IniCursorAfter string `db:"ini_cursor_after"`
	LogVouchersIds string `db:"log_vouchers_ids"`
	EndCursorAfter string `db:"end_cursor_after"`
}
