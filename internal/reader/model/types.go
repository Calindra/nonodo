// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

// Request submitted to the application to advance its state
type Input struct {
	// ID of the input
	ID string `json:"id"`
	// Input index starting from genesis
	Index int `json:"index"`
	// Status of the input
	Status CompletionStatus `json:"status"`
	// Address responsible for submitting the input
	MsgSender string `json:"msgSender"`
	// Timestamp associated with the input submission, as defined by the base layer's block in
	// which it was recorded
	Timestamp string `json:"timestamp"`
	// Number of the base layer block in which the input was recorded
	BlockNumber string `json:"blockNumber"`
	// Input payload in Ethereum hex binary format, starting with '0x'
	Payload string `json:"payload"`
	// Timestamp associated with the Espresso input submission
	EspressoTimestamp string `json:"espressoTimestamp"`
	// Number of the Espresso block in which the input was recorded
	EspressoBlockNumber string `json:"espressoBlockNumber"`
	// Input index in the Inpux Box
	InputBoxIndex string `json:"inputBoxIndex"`

	BlockTimestamp string `json:"blockTimestamp"`

	PrevRandao string `json:"prevRandao"`
}

// Representation of a transaction that can be carried out on the base layer blockchain, such as a
// transfer of assets
type Voucher struct {
	// Voucher index within the context of the input that produced it
	Index int `json:"index"`
	// Index of the input
	InputIndex int
	// Transaction destination address in Ethereum hex binary format (20 bytes), starting with
	// '0x'
	Destination string `json:"destination"`
	// Transaction payload in Ethereum hex binary format, starting with '0x'
	Payload string `json:"payload"`

	Value string `json:"value"`

	Executed bool `json:"executed"`

	Proof Proof `json:"proof"`

	TransactionHash string `json:"transactionHash"`
}

type Proof struct {
	OutputIndex          string   `json:"outputIndex"`
	OutputHashesSiblings []string `json:"outputHashesSiblings"`
}

// Application log or diagnostic information
type Report struct {
	// Report index within the context of the input that produced it
	Index int `json:"index"`
	// Index of the input
	InputIndex int
	// Report data as a payload in Ethereum hex binary format, starting with '0x'
	Payload string `json:"payload"`
}

// Informational statement that can be validated in the base layer blockchain
type Notice struct {
	// Notice index within the context of the input that produced it
	Index int `json:"index"`
	// Index of the input
	InputIndex int
	// Notice data as a payload in Ethereum hex binary format, starting with '0x'
	Payload string `json:"payload"`
	// InputId string
	Proof Proof `json:"proof"`
}

//
// Pagination types
//

type InputConnection = Connection[*Input]
type InputEdge = Edge[*Input]

type VoucherConnection = Connection[*Voucher]
type VoucherEdge = Edge[*Voucher]

type NoticeConnection = Connection[*Notice]
type NoticeEdge = Edge[*Notice]

type ReportConnection = Connection[*Report]
type ReportEdge = Edge[*Report]
