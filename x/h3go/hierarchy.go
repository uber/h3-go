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

import "iter"

// setResolution returns the cell with its resolution field set to res.
func (c Cell) setResolution(res int) Cell {
	c &= ^(Cell(resolutionMask) << resolutionOffset)
	c |= Cell(res) << resolutionOffset

	return c
}

// reservedBits returns the 3-bit reserved field of the index.
func (c Cell) reservedBits() int {
	return int(c>>reservedOffset) & 0x7
}

// zeroIndexDigits zeroes out index digits from start to end inclusive. It is a
// no-op if start > end.
func (c Cell) zeroIndexDigits(start, end int) Cell {
	if start > end {
		return c
	}
	// Mask with 0s in the [start, end] digit slots and 1s everywhere else.
	digits := Cell(1)<<(perDigitOffset*(end-start+1)) - 1
	mask := ^(digits << (perDigitOffset * (MaxResolution - end)))

	return c & mask
}

// hasChildAtRes reports whether childRes is a valid child resolution for c.
func (c Cell) hasChildAtRes(childRes int) bool {
	parentRes := c.Resolution()
	return childRes >= parentRes && childRes <= MaxResolution
}

// childrenSize returns the exact number of children of c at childRes, handling
// hexagons and pentagons.
func (c Cell) childrenSize(childRes int) (int64, error) {
	if !c.hasChildAtRes(childRes) {
		return 0, ErrResolutionDomain
	}

	n := childRes - c.Resolution()
	if c.IsPentagon() {
		return 1 + 5*(pow7[n]-1)/6, nil
	}

	return pow7[n], nil
}

// Parent returns the parent (or grandparent, etc.) of the cell at parentRes.
func (c Cell) Parent(parentRes int) (Cell, error) {
	childRes := c.Resolution()
	switch {
	case parentRes < 0 || parentRes > MaxResolution:
		return 0, ErrResolutionDomain
	case parentRes > childRes:
		return 0, ErrResolutionMismatch
	case parentRes == childRes:
		return c, nil
	}

	parentH := c.setResolution(parentRes)
	for i := parentRes + 1; i <= childRes; i++ {
		parentH = parentH.setIndexDigit(i, digitMask)
	}

	return parentH, nil
}

// ImmediateParent returns the immediate parent of the cell.
func (c Cell) ImmediateParent() (Cell, error) {
	return c.Parent(c.Resolution() - 1)
}

// CenterChild returns the center child of the cell at childRes.
func (c Cell) CenterChild(childRes int) (Cell, error) {
	if !c.hasChildAtRes(childRes) {
		return 0, ErrResolutionDomain
	}
	h := c.zeroIndexDigits(c.Resolution()+1, childRes)

	return h.setResolution(childRes), nil
}

// Children returns the children (or grandchildren, etc.) of the cell at
// childRes.
func (c Cell) Children(childRes int) ([]Cell, error) {
	size, err := c.childrenSize(childRes)
	if err != nil {
		return nil, err
	}

	out := make([]Cell, 0, size)
	for child := range c.childCells(childRes) {
		out = append(out, child)
	}

	return out, nil
}

// ImmediateChildren returns the immediate children of the cell.
func (c Cell) ImmediateChildren() ([]Cell, error) {
	return c.Children(c.Resolution() + 1)
}

// childCells returns an iterator over the children of c at childRes. It yields
// nothing for invalid input (c == 0, or childRes outside [resolution(c),
// MaxResolution]).
func (c Cell) childCells(childRes int) iter.Seq[Cell] {
	return func(yield func(Cell) bool) {
		parentRes := c.Resolution()
		if c == 0 || childRes < parentRes || childRes > MaxResolution {
			return
		}

		cur := c.zeroIndexDigits(parentRes+1, childRes).setResolution(childRes)

		// For pentagons the deleted K-axis (1) digit is skipped; the skip
		// position moves left as the carry propagates from child toward parent.
		skipDigit := -1
		if cur.IsPentagon() {
			skipDigit = childRes
		}

		for {
			if !yield(cur) {
				return
			}

			cur = cur.incrementResDigit(childRes)
			done := false

			for i := childRes; i >= parentRes; i-- {
				if i == parentRes {
					// Carrying into the parent digit means we're done.
					done = true
					break
				}
				// Skip the deleted 1 (K axis) digit for a pentagon's children:
				// the first non-zero digit can never be 1.
				if i == skipDigit && cur.indexDigit(i) == kAxesDigit {
					cur = cur.incrementResDigit(i)
					skipDigit--

					break
				}

				if cur.indexDigit(i) == invalidDigit {
					cur = cur.incrementResDigit(i) // zeroes digit i and carries into i-1
				} else {
					break
				}
			}

			if done {
				return
			}
		}
	}
}

// incrementResDigit returns h with the digit at res incremented by one. A digit
// that overflows past 6 becomes invalidDigit (7); incrementing again zeroes it
// and carries into the next coarser digit.
func (c Cell) incrementResDigit(res int) Cell {
	var val Cell = 1
	val <<= perDigitOffset * (MaxResolution - res)

	return c + val
}

// validateChildPos returns an error if childPos is out of range for the children
// of parent at childRes.
func (c Cell) validateChildPos(childPos int64, childRes int) error {
	maxChildCount, err := c.childrenSize(childRes)
	if err != nil {
		return err
	}

	if childPos < 0 || childPos >= maxChildCount {
		return ErrDomain
	}

	return nil
}

// CellToChildPos returns the position of cell within an ordered list of all
// children of the cell's parent at parentRes.
func CellToChildPos(cell Cell, parentRes int) (int, error) {
	return cell.ChildPos(parentRes)
}

// ChildPos returns the position of the cell within an ordered list of all
// children of the cell's parent at parentRes.
func (c Cell) ChildPos(parentRes int) (int, error) {
	childRes := c.Resolution()
	// Getting the parent catches any resolution errors.
	parent, err := c.Parent(parentRes)
	if err != nil {
		return 0, err
	}
	parentIsPentagon := parent.IsPentagon()
	var out int64

	if parentIsPentagon {
		// Pentagon parents skip the 1 digit, so the offsets differ from hexagons.
		for res := childRes; res > parentRes; res-- {
			// res-1 is always a valid parent resolution here, so it cannot error.
			parent, _ = c.Parent(res - 1)
			parentIsPentagon = parent.IsPentagon()

			rawDigit := c.indexDigit(res)
			if rawDigit == invalidDigit || (parentIsPentagon && rawDigit == kAxesDigit) {
				return 0, ErrCellInvalid
			}

			digit := rawDigit
			if parentIsPentagon && rawDigit > 0 {
				digit = rawDigit - 1
			}

			if digit != centerDigit {
				hexChildCount := pow7[childRes-res]
				if parentIsPentagon {
					out += 1 + 5*(hexChildCount-1)/6
				} else {
					out += hexChildCount
				}
				out += int64(digit-1) * hexChildCount
			}
		}
	} else {
		// Hexagon offsets are simple powers of 7.
		for res := childRes; res > parentRes; res-- {
			digit := c.indexDigit(res)
			if digit == invalidDigit {
				return 0, ErrCellInvalid
			}
			out += int64(digit) * pow7[childRes-res]
		}
	}

	return int(out), nil
}

// ChildPosToCell returns the child of parent at the given position within an
// ordered list of all children at childRes.
func ChildPosToCell(position int, parent Cell, childRes int) (Cell, error) {
	return parent.ChildPosToCell(position, childRes)
}

// ChildPosToCell returns the child cell at the given position within an ordered
// list of all children at childRes.
func (c Cell) ChildPosToCell(position int, childRes int) (Cell, error) {
	if childRes < 0 || childRes > MaxResolution {
		return 0, ErrResolutionDomain
	}

	parentRes := c.Resolution()
	if childRes < parentRes {
		return 0, ErrResolutionMismatch
	}

	if err := c.validateChildPos(int64(position), childRes); err != nil {
		return 0, err
	}
	resOffset := childRes - parentRes
	child := c.setResolution(childRes)
	idx := int64(position)

	if c.IsPentagon() {
		// Pentagon tiles skip the 1 digit, so the offsets differ.
		inPent := true

		for res := 1; res <= resOffset; res++ {
			resWidth := pow7[resOffset-res]
			if inPent {
				pentWidth := 1 + 5*(resWidth-1)/6
				if idx < pentWidth {
					child = child.setIndexDigit(parentRes+res, centerDigit)
				} else {
					idx -= pentWidth
					inPent = false
					child = child.setIndexDigit(parentRes+res, int(idx/resWidth)+2)
					idx %= resWidth
				}
			} else {
				child = child.setIndexDigit(parentRes+res, int(idx/resWidth))
				idx %= resWidth
			}
		}
	} else {
		// Hexagon offsets are simple powers of 7.
		for res := 1; res <= resOffset; res++ {
			resWidth := pow7[resOffset-res]
			child = child.setIndexDigit(parentRes+res, int(idx/resWidth))
			idx %= resWidth
		}
	}

	return child, nil
}

// UncompactCells expands a compacted set back to the full set of cells at res.
func UncompactCells(in []Cell, res int) ([]Cell, error) {
	var total int64

	for _, c := range in {
		if c == 0 {
			continue
		}

		size, err := c.childrenSize(res)
		if err != nil {
			return nil, ErrResolutionMismatch
		}
		total += size
	}
	out := make([]Cell, 0, total)

	for _, c := range in {
		for child := range c.childCells(res) {
			out = append(out, child)
		}
	}

	return out, nil
}

// CompactCells merges full sets of children into their parent recursively,
// until no more merges are possible. The output order is unspecified.
func CompactCells(in []Cell) ([]Cell, error) {
	if len(in) == 0 {
		return []Cell{}, nil
	}

	if in[0].Resolution() == 0 {
		// No compaction possible at resolution 0.
		out := make([]Cell, len(in))
		copy(out, in)

		return out, nil
	}
	remaining := make([]Cell, len(in))
	copy(remaining, in)
	var output []Cell

	for len(remaining) > 0 {
		parentRes := remaining[0].Resolution() - 1
		if parentRes < 0 {
			// Compacted all the way up to base cells.
			output = append(output, remaining...)
			break
		}
		counts := make(map[Cell]int)

		order := make([]Cell, 0, len(remaining))
		for _, c := range remaining {
			if c == 0 {
				continue
			}

			if c.reservedBits() != 0 {
				return nil, ErrCellInvalid
			}

			parent, err := c.Parent(parentRes)
			if err != nil {
				return nil, err
			}

			if _, ok := counts[parent]; !ok {
				order = append(order, parent)
			}

			counts[parent]++
			if counts[parent] > parent.fullChildCount() {
				return nil, ErrDuplicateInput
			}
		}
		compactable := make(map[Cell]bool)
		var next []Cell

		for _, parent := range order {
			if counts[parent] == parent.fullChildCount() {
				compactable[parent] = true
				next = append(next, parent)
			}
		}

		for _, c := range remaining {
			if c == 0 {
				continue
			}

			parent, _ := c.Parent(parentRes)
			if !compactable[parent] {
				output = append(output, c)
			}
		}
		remaining = next
	}

	return output, nil
}

// Child-set sizes: a hexagon has 7 immediate children, a pentagon 6 (its
// K-axis child is deleted).
const (
	numHexChildren  = 7
	numPentChildren = 6
)

// fullChildCount returns the number of immediate children that make a cell's
// child set complete: 6 for pentagons (the K-axis child is deleted), 7 for
// hexagons.
func (c Cell) fullChildCount() int {
	if c.IsPentagon() {
		return numPentChildren
	}

	return numHexChildren
}
