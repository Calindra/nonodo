// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

const DefaultPaginationLimit = 1000

var MixedPaginationErr = errors.New(
	"cannot mix forward pagination (first, after) with backward pagination (last, before)")
var InvalidCursorErr = errors.New("invalid pagination cursor")
var InvalidLimitErr = errors.New("limit cannot be negative")

// Compute the pagination parameters given the GraphQL connection parameters.
func computePage(
	first *int, last *int, after *string, before *string, total int,
) (offset int, limit int, err error) {
	forward := first != nil || after != nil
	backward := last != nil || before != nil
	if forward && backward {
		return 0, 0, MixedPaginationErr
	}
	if !forward && !backward {
		// If nothing was set, use forward pagination by default
		forward = true
	}
	if forward {
		return computeForwardPage(first, after, total)
	} else {
		return computeBackwardPage(last, before, total)
	}
}

// Compute the pagination parameters when paginating forward
func computeForwardPage(first *int, after *string, total int) (offset int, limit int, err error) {
	if first != nil {
		if *first < 0 {
			return 0, 0, InvalidLimitErr
		}
		limit = *first
	} else {
		limit = DefaultPaginationLimit
	}
	if after != nil {
		offset, err = decodeCursor(*after, total)
		if err != nil {
			return 0, 0, err
		}
		offset = offset + 1
	} else {
		offset = 0
	}
	limit = min(limit, total-offset)
	return offset, limit, nil
}

// Compute the pagination parameters when paginating backward
func computeBackwardPage(last *int, before *string, total int) (offset int, limit int, err error) {
	if last != nil {
		if *last < 0 {
			return 0, 0, InvalidLimitErr
		}
		limit = *last
	} else {
		limit = DefaultPaginationLimit
	}
	var beforeOffset int
	if before != nil {
		beforeOffset, err = decodeCursor(*before, total)
		if err != nil {
			return 0, 0, err
		}
	} else {
		beforeOffset = total
	}
	offset = max(0, beforeOffset-limit)
	limit = min(limit, total-offset)
	return offset, limit, nil
}

// Pagination result
type Connection[T any] struct {
	// Total number of entries that match the query
	TotalCount int `json:"totalCount"`
	// Pagination entries returned for the current page
	Edges []*Edge[T] `json:"edges"`
	// Pagination metadata
	PageInfo *PageInfo `json:"pageInfo"`
}

// Create a new connection for the given slice of elements.
func NewConnection[T any](offset int, total int, nodes []T) *Connection[T] {
	edges := make([]*Edge[T], len(nodes))
	for i := range nodes {
		edges[i] = &Edge[T]{
			Node:   nodes[i],
			offset: offset + i,
		}
	}
	var pageInfo PageInfo
	if len(edges) > 0 {
		startCursor := encodeCursor(edges[0].offset)
		pageInfo.StartCursor = &startCursor
		pageInfo.HasPreviousPage = edges[0].offset > 0
		endCursor := encodeCursor(edges[len(edges)-1].offset)
		pageInfo.EndCursor = &endCursor
		pageInfo.HasNextPage = edges[len(edges)-1].offset < total-1
	}
	conn := Connection[T]{
		TotalCount: total,
		Edges:      edges,
		PageInfo:   &pageInfo,
	}
	return &conn
}

// Pagination entry
type Edge[T any] struct {
	// Node instance
	Node T `json:"node"`
	// Pagination offset
	offset int
}

// Encode the cursor from the offset.
func (e *Edge[T]) Cursor() string {
	return encodeCursor(e.offset)
}

// Encode the integer offset into a base64 string.
func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(offset)))
}

// Decode the integer offset from a base64 string.
func decodeCursor(base64Cursor string, total int) (int, error) {
	cursorBytes, err := base64.StdEncoding.DecodeString(base64Cursor)
	if err != nil {
		return 0, err
	}
	offset, err := strconv.Atoi(string(cursorBytes))
	if err != nil {
		return 0, err
	}
	if offset < 0 || offset >= total {
		return 0, InvalidCursorErr
	}
	return offset, nil
}
