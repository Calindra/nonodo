// Copyright (c) Gabriel de Quadros Ligneul
// SPDX-License-Identifier: Apache-2.0 (see LICENSE)

package repository

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
