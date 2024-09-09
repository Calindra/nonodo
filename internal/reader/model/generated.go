// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"
)

type AddressFilterInput struct {
	Eq  *string             `json:"eq,omitempty"`
	Ne  *string             `json:"ne,omitempty"`
	In  []*string           `json:"in,omitempty"`
	Nin []*string           `json:"nin,omitempty"`
	And []*ConvenientFilter `json:"and,omitempty"`
	Or  []*ConvenientFilter `json:"or,omitempty"`
}

type BooleanFilterInput struct {
	Eq  *bool               `json:"eq,omitempty"`
	Ne  *bool               `json:"ne,omitempty"`
	And []*ConvenientFilter `json:"and,omitempty"`
	Or  []*ConvenientFilter `json:"or,omitempty"`
}

type ConvenientFilter struct {
	Destination *AddressFilterInput `json:"destination,omitempty"`
	Executed    *BooleanFilterInput `json:"executed,omitempty"`
	And         []*ConvenientFilter `json:"and,omitempty"`
	Or          []*ConvenientFilter `json:"or,omitempty"`
}

// Filter object to restrict results depending on input properties
type InputFilter struct {
	// Filter only inputs with index lower than a given value
	IndexLowerThan *int `json:"indexLowerThan,omitempty"`
	// Filter only inputs with index greater than a given value
	IndexGreaterThan *int `json:"indexGreaterThan,omitempty"`
	// Filter only inputs with the message sender
	MsgSender *string `json:"msgSender,omitempty"`
	Type      *string `json:"type,omitempty"`
}

// Page metadata for the cursor-based Connection pagination pattern
type PageInfo struct {
	// Cursor pointing to the first entry of the page
	StartCursor *string `json:"startCursor,omitempty"`
	// Cursor pointing to the last entry of the page
	EndCursor *string `json:"endCursor,omitempty"`
	// Indicates if there are additional entries after the end curs
	HasNextPage bool `json:"hasNextPage"`
	// Indicates if there are additional entries before the start curs
	HasPreviousPage bool `json:"hasPreviousPage"`
}

// Data that can be used as proof to validate notices and execute vouchers on the base layer blockchain
type Proof struct {
	FirstIndex int `json:"firstIndex"`
	// Reads a single `Input` that is related to this `Proof`.
	InputByInputIndex *Input `json:"inputByInputIndex,omitempty"`
	InputIndex        int    `json:"inputIndex"`
	LastInput         int    `json:"lastInput"`
	// A globally unique identifier. Can be used in various places throughout the system to identify this single value.
	NodeID                                   string    `json:"nodeId"`
	OutputIndex                              int       `json:"outputIndex"`
	ValidityInputIndexWithinEpoch            int       `json:"validityInputIndexWithinEpoch"`
	ValidityMachineStateHash                 string    `json:"validityMachineStateHash"`
	ValidityOutputEpochRootHash              string    `json:"validityOutputEpochRootHash"`
	ValidityOutputHashInOutputHashesSiblings []*string `json:"validityOutputHashInOutputHashesSiblings"`
	ValidityOutputHashesInEpochSiblings      []*string `json:"validityOutputHashesInEpochSiblings"`
	ValidityOutputHashesRootHash             string    `json:"validityOutputHashesRootHash"`
	ValidityOutputIndexWithinInput           int       `json:"validityOutputIndexWithinInput"`
}

type CompletionStatus string

const (
	CompletionStatusUnprocessed                CompletionStatus = "UNPROCESSED"
	CompletionStatusAccepted                   CompletionStatus = "ACCEPTED"
	CompletionStatusRejected                   CompletionStatus = "REJECTED"
	CompletionStatusException                  CompletionStatus = "EXCEPTION"
	CompletionStatusMachineHalted              CompletionStatus = "MACHINE_HALTED"
	CompletionStatusCycleLimitExceeded         CompletionStatus = "CYCLE_LIMIT_EXCEEDED"
	CompletionStatusTimeLimitExceeded          CompletionStatus = "TIME_LIMIT_EXCEEDED"
	CompletionStatusPayloadLengthLimitExceeded CompletionStatus = "PAYLOAD_LENGTH_LIMIT_EXCEEDED"
)

var AllCompletionStatus = []CompletionStatus{
	CompletionStatusUnprocessed,
	CompletionStatusAccepted,
	CompletionStatusRejected,
	CompletionStatusException,
	CompletionStatusMachineHalted,
	CompletionStatusCycleLimitExceeded,
	CompletionStatusTimeLimitExceeded,
	CompletionStatusPayloadLengthLimitExceeded,
}

func (e CompletionStatus) IsValid() bool {
	switch e {
	case CompletionStatusUnprocessed, CompletionStatusAccepted, CompletionStatusRejected, CompletionStatusException, CompletionStatusMachineHalted, CompletionStatusCycleLimitExceeded, CompletionStatusTimeLimitExceeded, CompletionStatusPayloadLengthLimitExceeded:
		return true
	}
	return false
}

func (e CompletionStatus) String() string {
	return string(e)
}

func (e *CompletionStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = CompletionStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid CompletionStatus", str)
	}
	return nil
}

func (e CompletionStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
