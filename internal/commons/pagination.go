// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package commons

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

const DefaultPaginationLimit = 1000

var ErrMixedPagination = errors.New(
	"cannot mix forward pagination (first, after) with backward pagination (last, before)")
var ErrInvalidCursor = errors.New("invalid pagination cursor")
var ErrInvalidLimit = errors.New("limit cannot be negative")

type PageResult[T any] struct {
	Total  uint64
	Offset uint64
	Rows   []T
}

// Compute the pagination parameters given the GraphQL connection parameters.
func ComputePage(
	first *int, last *int, after *string, before *string, total int,
) (offset int, limit int, err error) {
	forward := first != nil || after != nil
	backward := last != nil || before != nil
	if forward && backward {
		return 0, 0, ErrMixedPagination
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
			return 0, 0, ErrInvalidLimit
		}
		limit = *first
	} else {
		limit = DefaultPaginationLimit
	}
	if after != nil {
		offset, err = DecodeCursor(*after, total)
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
			return 0, 0, ErrInvalidLimit
		}
		limit = *last
	} else {
		limit = DefaultPaginationLimit
	}
	var beforeOffset int
	if before != nil {
		beforeOffset, err = DecodeCursor(*before, total)
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

// Encode the integer offset into a base64 string.
func EncodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(offset)))
}

// Decode the integer offset from a base64 string.
func DecodeCursor(base64Cursor string, total int) (int, error) {
	cursorBytes, err := base64.StdEncoding.DecodeString(base64Cursor)
	if err != nil {
		return 0, err
	}
	offset, err := strconv.Atoi(string(cursorBytes))
	if err != nil {
		return 0, err
	}
	if offset < 0 || offset >= total {
		return 0, ErrInvalidCursor
	}
	return offset, nil
}
