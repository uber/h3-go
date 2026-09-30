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
	"math"
	"strconv"
)

// maxIndexHexLen is the length of the longest hexadecimal rendering of a 64-bit
// index.
const maxIndexHexLen = 16

// IndexFromString parses an H3 index from its hexadecimal string
// representation, with an optional "0x" prefix. Callers should validate the
// result (e.g. with Cell.IsValid) before use.
func IndexFromString(s string) uint64 {
	return parseIndexHex(s)
}

// parseIndexHex parses a hexadecimal index with an optional "0x" or "0X"
// prefix. It matches strconv.ParseUint(s, 16, 64) with the error discarded:
// malformed or empty input yields 0, and a value beyond 64 bits yields
// math.MaxUint64. It is generic over string and []byte so the text unmarshalers
// can parse without copying.
func parseIndexHex[S string | []byte](s S) uint64 {
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}

	if len(s) == 0 {
		return 0
	}

	var index uint64

	for i := 0; i < len(s); i++ {
		var digit uint64

		switch ch := s[i]; {
		case '0' <= ch && ch <= '9':
			digit = uint64(ch - '0')
		case 'a' <= ch && ch <= 'f':
			digit = uint64(ch-'a') + 10
		case 'A' <= ch && ch <= 'F':
			digit = uint64(ch-'A') + 10
		default:
			return 0
		}

		if index>>(64-4) != 0 {
			return math.MaxUint64
		}

		index = index<<4 | digit
	}

	return index
}

// appendIndexHex returns the hexadecimal rendering of index in a fresh byte
// slice, sized so the append never grows it.
func appendIndexHex(index uint64) []byte {
	return strconv.AppendUint(make([]byte, 0, maxIndexHexLen), index, base16)
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
	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	return appendIndexHex(uint64(c)), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (c *Cell) UnmarshalText(text []byte) error {
	//nolint:gosec // an H3 index is a 64-bit value; uint64->int64 is a lossless reinterpretation.
	*c = Cell(parseIndexHex(text))
	if !c.IsValid() {
		return errors.New("invalid cell index")
	}

	return nil
}
