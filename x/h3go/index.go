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
)

type (
	// Cell is an Index that identifies a single hexagon cell at a resolution.
	Cell int64

	// Index is the constraint satisfied by the H3 index types. They share the same
	// 64-bit encoding and are distinguished by their mode field.
	Index interface {
		Cell | DirectedEdge | Vertex
	}

	// mode is the H3 index mode stored in an index's mode field. It distinguishes
	// the index types that share the 64-bit encoding.
	mode int
)

// Index modes, matching the H3 C library's H3_*_MODE values.
const (
	cellMode         mode = iota + 1 // H3_CELL_MODE
	directedEdgeMode                 // H3_DIRECTEDEDGE_MODE
	edgeMode                         // H3_EDGE_MODE (undirected edge, unused)
	vertexMode                       // H3_VERTEX_MODE
)

// Index-format constants shared by the introspection and hierarchy helpers.
const (
	base16  = 16
	bitSize = 64

	// reservedOffset is the bit offset of the reserved field (H3_RESERVED_OFFSET).
	reservedOffset = 56
	// baseCellMask masks the 7-bit base cell field after shifting.
	baseCellMask = 0x7F
	// modeMask masks the 4-bit mode field after shifting.
	modeMask = 0xF
	// reservedMask masks the 3-bit reserved field after shifting.
	reservedMask = 0x7

	// digitRegionOffset is the number of non-digit bits above the 15×3-bit digit
	// region (high + mode + reserved + resolution + base cell = 19).
	digitRegionOffset = bitSize - MaxResolution*perDigitOffset

	// validCellTopBits is the expected value of the top 8 bits of a valid cell:
	// high bit=0, mode=1 (cell), reserved=000.
	validCellTopBits = 0b00001000
)

// pow7 holds precomputed powers of 7: pow7[i] == 7^i for i in [0, MaxResolution].
var pow7 = [MaxResolution + 1]int64{
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

// modeOf returns the H3 index mode (cell, directed edge, or vertex) of any
// index.
func modeOf[I Index](index I) mode {
	return mode((int64(index) >> modeOffset) & modeMask)
}

// reservedBits returns the 3-bit reserved field of any index. It holds the
// direction of a directed edge or the vertex number of a vertex.
func reservedBits[I Index](index I) int {
	return int(int64(index)>>reservedOffset) & reservedMask
}

// resolution returns the resolution field of any index.
func resolution[I Index](index I) int {
	return int(int64(index)>>resolutionOffset) & resolutionMask
}

// indexDigit returns the indexing digit at res of any index, without bounds
// checking.
func indexDigit[I Index](index I, res int) int {
	return int((int64(index) >> ((MaxResolution - res) * perDigitOffset)) & digitMask)
}

// indexDigitChecked returns the indexing digit at res of any index, for res in
// [1, MaxResolution], and ErrResolutionDomain otherwise.
func indexDigitChecked[I Index](index I, res int) (int, error) {
	if res < 1 || res > MaxResolution {
		return 0, ErrResolutionDomain
	}

	return indexDigit(index, res), nil
}

// ownerCell returns the cell that owns index: the same index bits reinterpreted
// as a cell, with the mode set to cell and the reserved field cleared. For a
// directed edge this is its origin cell; for a vertex, its owner cell.
func ownerCell[I Index](index I) Cell {
	return Cell(int64(index)).setMode(cellMode).setReservedBits(0)
}

// setIndexDigit returns the index with the digit at res set to digit.
func (c Cell) setIndexDigit(res, digit int) Cell {
	shift := (MaxResolution - res) * perDigitOffset
	c &= ^(Cell(digitMask) << shift)
	c |= Cell(digit) << shift

	return c
}

// setBaseCell returns the index with its base cell field set to baseCell.
func (c Cell) setBaseCell(baseCell int) Cell {
	c &= ^(Cell(baseCellMask) << baseCellOffset)
	c |= Cell(baseCell) << baseCellOffset

	return c
}

// setMode returns the index with its 4-bit mode field set to m.
func (c Cell) setMode(m mode) Cell {
	c &= ^(Cell(modeMask) << modeOffset)
	c |= Cell(m) << modeOffset

	return c
}

// setReservedBits returns the index with its 3-bit reserved field set to value.
// The reserved field holds the direction of a directed edge or the vertex number
// of a vertex.
func (c Cell) setReservedBits(value int) Cell {
	c &= ^(Cell(reservedMask) << reservedOffset)
	c |= Cell(value) << reservedOffset

	return c
}

// BaseCellNumber returns the integer ID (0-121) of the base cell the cell
// belongs to.
func BaseCellNumber(c Cell) int {
	return c.BaseCellNumber()
}

// BaseCellNumber returns the integer ID (0-121) of the base cell the cell
// belongs to.
func (c Cell) BaseCellNumber() int {
	return int(c>>baseCellOffset) & baseCellMask
}

// NumCells returns the number of cells at the given resolution. Resolutions
// outside [0, MaxResolution] return 0.
func NumCells(res int) int {
	if res < 0 || res > MaxResolution {
		return 0
	}
	// See h3api.h for the formula derivation.
	return int(2 + 120*pow7[res])
}

// Resolution returns the resolution of the cell.
func (c Cell) Resolution() int {
	return resolution(c)
}

// isResClassIII reports whether a resolution is Class III. Odd resolutions are
// Class III; even resolutions are Class II.
func isResClassIII(res int) bool {
	return res%2 == 1
}

// IsResClassIII reports whether the cell is in a Class III resolution. Odd
// resolutions are Class III; even resolutions are Class II.
func (c Cell) IsResClassIII() bool {
	return isResClassIII(c.Resolution())
}

// IsPentagon reports whether the cell is a pentagon: its base cell is a pentagon
// and it has no leading non-zero digit.
func (c Cell) IsPentagon() bool {
	if !isBaseCellPentagon[c.BaseCellNumber()] {
		return false
	}

	for r := 1; r <= c.Resolution(); r++ {
		if indexDigit(c, r) != centerDigit {
			return false
		}
	}

	return true
}

// IndexDigit returns the indexing digit of the cell at res, which starts at 1
// for resolution 1 up to and including resolution 15.
func (c Cell) IndexDigit(res int) (int, error) {
	return indexDigitChecked(c, res)
}

// IsValid reports whether the cell is a valid H3 cell (hexagon or pentagon). It
// looks for bit patterns that would disqualify an index from being a valid cell,
// exiting early.
func (c Cell) IsValid() bool {
	if c == 0 {
		return false
	}

	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	h := uint64(c)
	// Top 8 bits: high=0, mode=1 (cell), reserved=0 => 0b00001000.
	if h>>reservedOffset != validCellTopBits {
		return false
	}

	bc := c.BaseCellNumber()
	if bc >= NumBaseCells {
		return false
	}

	res := c.Resolution()
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
	shift := perDigitOffset * (MaxResolution - res)
	h >>= shift
	h <<= shift
	h = h & mhi & (^h - mlo)

	return h != 0
}

// hasAll7AfterRes reports whether all unused digits after res are set to 7.
func hasAll7AfterRes(h uint64, res int) bool {
	if res >= MaxResolution {
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
	if !isBaseCellPentagon[baseCell] {
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

// IsValidIndex reports whether index is valid for its mode (cell, directed edge,
// or vertex).
func IsValidIndex[T Index](index T) bool {
	switch modeOf(index) {
	case cellMode:
		return Cell(int64(index)).IsValid()
	case directedEdgeMode:
		return DirectedEdge(int64(index)).IsValid()
	case vertexMode:
		return Vertex(int64(index)).IsValid()
	default:
		return false
	}
}
