// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Rollups voucher type.
type Voucher struct {
	Index       int
	InputIndex  int
	Destination common.Address
	Payload     []byte
}

func (v Voucher) GetInputIndex() int {
	return v.InputIndex
}

// Rollups notice type.
type Notice struct {
	Index      int
	InputIndex int
	Payload    []byte
}

func (n Notice) GetInputIndex() int {
	return n.InputIndex
}

// Rollups report type.
type Report struct {
	Index      int
	InputIndex int
	Payload    []byte
}

func (r Report) GetInputIndex() int {
	return r.InputIndex
}

// Completion status for inputs.
type CompletionStatus int

const (
	CompletionStatusUnprocessed CompletionStatus = iota
	CompletionStatusAccepted
	CompletionStatusRejected
	CompletionStatusException
)

// Rollups input, which can be advance or inspect.
type Input interface{}

// Rollups advance input type.
type AdvanceInput struct {
	Index       int
	Status      CompletionStatus
	MsgSender   common.Address
	Payload     []byte
	BlockNumber uint64
	Timestamp   time.Time
	Vouchers    []Voucher
	Notices     []Notice
	Reports     []Report
	Exception   []byte
}

// Rollups inspect input type.
type InspectInput struct {
	Index                int
	Status               CompletionStatus
	Payload              []byte
	ProccessedInputCount int
	Reports              []Report
	Exception            []byte
}

// Voucher metadata type
type VoucherMetadata struct {
	Contract      common.Address
	Beneficiary   common.Address
	Label         string
	Amount        uint64
	ExecutedAt    uint64
	ExecutedBlock uint64
	InputIndex    int
	OutputIndex   int
}
