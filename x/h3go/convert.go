/*
 * Copyright 2026 Uber Technologies, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package h3go

import (
	"errors"
	"strconv"
	"strings"
)

// IndexFromString parses an H3 index from its hexadecimal string
// representation, with an optional "0x" prefix. Callers should validate the
// result (e.g. with Cell.IsValid) before use.
func IndexFromString(s string) uint64 {
	if len(s) > 2 && strings.ToLower(s[:2]) == "0x" {
		s = s[2:]
	}
	index, _ := strconv.ParseUint(s, base16, bitSize)

	return index
}

// IndexToString returns the hexadecimal string representation of an H3 index.
func IndexToString(index uint64) string {
	return strconv.FormatUint(index, base16)
}

// CellFromString returns a Cell parsed from its hexadecimal string
// representation. Callers should validate it with Cell.IsValid before use.
func CellFromString(s string) Cell {
	//nolint:gosec // an H3 index is a 64-bit value; uint64->int64 is a lossless reinterpretation.
	return Cell(IndexFromString(s))
}

// CellToString returns the hexadecimal string representation of a Cell.
func CellToString(c Cell) string {
	return c.String()
}

// String returns the hexadecimal string representation of the cell.
func (c Cell) String() string {
	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	return IndexToString(uint64(c))
}

// MarshalText implements the encoding.TextMarshaler interface.
func (c Cell) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (c *Cell) UnmarshalText(text []byte) error {
	*c = CellFromString(string(text))
	if !c.IsValid() {
		return errors.New("invalid cell index")
	}

	return nil
}
