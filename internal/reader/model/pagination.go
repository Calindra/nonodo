// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package model

import (
	"encoding/base64"
	"fmt"
)

const DefaultPaginationLimit = 1000

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
