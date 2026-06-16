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
	"math/bits"

	"github.com/uber/h3-go/v4/internal/h3core"
)

// Index-format constants shared by the introspection and hierarchy helpers.
const (
	base16  = 16
	bitSize = 64

	// numBaseCells is the number of H3 base cells (NUM_BASE_CELLS).
	numBaseCells = h3core.NumBaseCells
	// numPentagons is the number of H3 pentagons, the same at every resolution.
	numPentagons = h3core.NumPentagons

	// reservedOffset is the bit offset of the reserved field (H3_RESERVED_OFFSET).
	reservedOffset = 56
	// baseCellMask masks the 7-bit base cell field after shifting.
	baseCellMask = 0x7F
	// modeMask masks the 4-bit mode field after shifting.
	modeMask = 0xF

	// digitRegionOffset is the number of non-digit bits above the 15×3-bit digit
	// region (high + mode + reserved + resolution + base cell = 19).
	digitRegionOffset = bitSize - maxResolution*perDigitOffset

	// validCellTopBits is the expected value of the top 8 bits of a valid cell:
	// high bit=0, mode=1 (cell), reserved=000.
	validCellTopBits = 0b00001000
)

// pow7 holds precomputed powers of 7: pow7[i] == 7^i for i in [0, maxResolution].
var pow7 = [maxResolution + 1]int64{
	1,
	7,
	49,
	343,
	2401,
	16807,
	117649,
	823543,
	5764801,
	40353607,
	282475249,
	1977326743,
	13841287201,
	96889010407,
	678223072849,
	4747561509943,
}

// resolution returns the resolution of the index.
func resolution(c Cell) int {
	return int(c>>resolutionOffset) & resolutionMask
}

// mode returns the H3 index mode (cell, directed edge, or vertex).
func mode(c Cell) int {
	return int(c>>modeOffset) & modeMask
}

// baseCellNumber returns the base cell number (0-121) of the index.
func baseCellNumber(c Cell) int {
	return int(c>>baseCellOffset) & baseCellMask
}

// getIndexDigit returns the indexing digit of the index at res.
func getIndexDigit(c Cell, res int) int {
	return int((c >> ((maxResolution - res) * perDigitOffset)) & digitMask)
}

// setIndexDigit returns the index with the digit at res set to digit.
func setIndexDigit(c Cell, res, digit int) Cell {
	shift := (maxResolution - res) * perDigitOffset
	c &= ^(Cell(digitMask) << shift)
	c |= Cell(digit) << shift

	return c
}

// BaseCellNumber returns the integer ID (0-121) of the base cell the cell
// belongs to.
func BaseCellNumber(c Cell) int {
	return baseCellNumber(c)
}

// BaseCellNumber returns the integer ID (0-121) of the base cell the cell
// belongs to.
func (c Cell) BaseCellNumber() int {
	return baseCellNumber(c)
}

// NumCells returns the number of cells at the given resolution. Resolutions
// outside [0, maxResolution] return 0.
func NumCells(res int) int {
	if res < 0 || res > maxResolution {
		return 0
	}
	// See h3api.h for the formula derivation.
	return int(2 + 120*pow7[res]) //nolint:mnd // math formula
}

// Resolution returns the resolution of the cell.
func (c Cell) Resolution() int {
	return resolution(c)
}

// IsResClassIII reports whether the cell is in a Class III resolution. Odd
// resolutions are Class III; even resolutions are Class II.
func (c Cell) IsResClassIII() bool {
	return resolution(c)%2 == 1
}

// IsPentagon reports whether the cell is a pentagon.
func (c Cell) IsPentagon() bool {
	return isPentagonCell(c)
}

// IsValid reports whether the cell is a valid H3 cell (hexagon or pentagon).
func (c Cell) IsValid() bool {
	return c != 0 && isValidCell(c)
}

// IndexDigit returns the indexing digit of the cell at res, which starts at 1
// for resolution 1 up to and including resolution 15.
func (c Cell) IndexDigit(res int) (int, error) {
	if res < 1 || res > maxResolution {
		return 0, ErrResolutionDomain
	}

	return getIndexDigit(c, res), nil
}

// isPentagonCell reports whether the cell is a pentagon: its base cell is a
// pentagon and it has no leading non-zero digit.
func isPentagonCell(c Cell) bool {
	if !h3core.IsBaseCellPentagon[baseCellNumber(c)] {
		return false
	}

	for r := 1; r <= resolution(c); r++ {
		if getIndexDigit(c, r) != centerDigit {
			return false
		}
	}

	return true
}

// isValidCell looks for bit patterns that would disqualify an index from being
// a valid cell, exiting early.
func isValidCell(c Cell) bool {
	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	h := uint64(c)
	// Top 8 bits: high=0, mode=1 (cell), reserved=0 => 0b00001000.
	if h>>reservedOffset != validCellTopBits {
		return false
	}

	bc := baseCellNumber(c)
	if bc >= numBaseCells {
		return false
	}

	res := resolution(c)
	if hasAny7UptoRes(h, res) {
		return false
	}

	if !hasAll7AfterRes(h, res) {
		return false
	}

	return !hasDeletedSubsequence(h, bc)
}

// hasAny7UptoRes detects whether any digit from 1 to res is 7 (invalid) without
// looping, using a carry-based trick. mhi selects the high bit of each 3-bit
// digit and mlo the low bit; for a digit equal to 7 (0b111), subtracting mlo
// from its complement borrows into the high bit, so any non-zero result means
// at least one digit is 7.
func hasAny7UptoRes(h uint64, res int) bool {
	const mhi uint64 = 0b100100100100100100100100100100100100100100100
	const mlo = mhi >> 2
	shift := perDigitOffset * (maxResolution - res)
	h >>= shift
	h <<= shift
	h = h & mhi & (^h - mlo)

	return h != 0
}

// hasAll7AfterRes reports whether all unused digits after res are set to 7.
func hasAll7AfterRes(h uint64, res int) bool {
	if res >= maxResolution {
		return true
	}
	shift := digitRegionOffset + perDigitOffset*res
	h = ^h
	h <<= shift
	h >>= shift

	return h == 0
}

// hasDeletedSubsequence reports whether a pentagon cell has the invalid
// "deleted subsequence" pattern: its first non-zero digit is 1 (K axis).
func hasDeletedSubsequence(h uint64, baseCell int) bool {
	if !h3core.IsBaseCellPentagon[baseCell] {
		return false
	}
	h <<= digitRegionOffset

	h >>= digitRegionOffset
	if h == 0 {
		return false
	}

	return firstOneIndex(h)%perDigitOffset == 0
}

// firstOneIndex returns the index of the most significant set bit.
func firstOneIndex(h uint64) int {
	return (bitSize - 1) - bits.LeadingZeros64(h)
}
