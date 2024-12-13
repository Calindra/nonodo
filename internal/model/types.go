// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	cModel "github.com/calindra/cartesi-rollups-graphql/pkg/convenience/model"
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

// Rollups inspect input type.
type InspectInput struct {
	Index               int
	Status              cModel.CompletionStatus
	Payload             []byte
	ProcessedInputCount int
	Reports             []Report
	Exception           []byte
}
