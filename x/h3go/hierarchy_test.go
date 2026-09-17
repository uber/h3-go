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
	"sort"
	"testing"
)

// TestCellToChildrenSizeKnown covers known-value cases for cellToChildrenSize,
// spanning hexagons, pentagons, and the largest res-0 cases.
func TestCellToChildrenSizeKnown(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveCell     Cell
		giveChildRes int
		wantSize     int64
		wantErr      bool
	}{
		"hex_coarser_res":     {giveCell: 0x87283080dffffff, giveChildRes: 3, wantErr: true},
		"hex_same_res":        {giveCell: 0x87283080dffffff, giveChildRes: 7, wantSize: 1},
		"hex_child_res":       {giveCell: 0x87283080dffffff, giveChildRes: 8, wantSize: 7},
		"hex_grandchild_res":  {giveCell: 0x87283080dffffff, giveChildRes: 9, wantSize: 49},
		"pent_coarser_res":    {giveCell: 0x870800000ffffff, giveChildRes: 3, wantErr: true},
		"pent_same_res":       {giveCell: 0x870800000ffffff, giveChildRes: 7, wantSize: 1},
		"pent_child_res":      {giveCell: 0x870800000ffffff, giveChildRes: 8, wantSize: 6},
		"pent_grandchild_res": {giveCell: 0x870800000ffffff, giveChildRes: 9, wantSize: 5*7 + 6},
		"largest_hexagon":     {giveCell: 0x806dfffffffffff, giveChildRes: 15, wantSize: 4747561509943},
		"largest_pentagon":    {giveCell: 0x8009fffffffffff, giveChildRes: 15, wantSize: 3956301258286},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.giveCell.childrenSize(tt.giveChildRes)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && got != tt.wantSize {
				t.Fatalf("size = %d, want %d", got, tt.wantSize)
			}
		})
	}
}

// TestChildrenErrors covers the error cases for Cell.Children.
func TestChildrenErrors(t *testing.T) {
	t.Parallel()

	c, err := LatLngToCell(LatLng{Lat: 37.7749, Lng: -122.4194}, 7)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	t.Run("coarser_resolution", func(t *testing.T) {
		t.Parallel()

		if _, err := c.Children(6); err == nil {
			t.Fatal("Children at coarser res should fail")
		}
	})

	t.Run("beyond_finest_resolution", func(t *testing.T) {
		t.Parallel()

		if _, err := c.Children(MaxResolution + 1); err == nil {
			t.Fatal("Children beyond finest res should fail")
		}
	})

	t.Run("same_resolution_returns_self", func(t *testing.T) {
		t.Parallel()

		same, err := c.Children(7)
		if err != nil {
			t.Fatalf("Children same res: %v", err)
		}

		if len(same) != 1 || same[0] != c {
			t.Fatalf("Children same res: got %v, want [%015x]", same, uint64(c))
		}
	})
}

// TestCellToChildPosErrors covers the resolution error cases for
// CellToChildPos. The cell is res 8.
func TestCellToChildPosErrors(t *testing.T) {
	t.Parallel()

	child := Cell(0x88283080ddfffff)

	tests := map[string]struct {
		giveRes int
		wantErr error
	}{
		"negative_resolution": {giveRes: -1, wantErr: ErrResolutionDomain},
		"too_high_resolution": {giveRes: 42, wantErr: ErrResolutionDomain},
		"finer_than_cell":     {giveRes: 9, wantErr: ErrResolutionMismatch},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := CellToChildPos(child, tt.giveRes); !errors.Is(err, tt.wantErr) {
				t.Fatalf("CellToChildPos(res=%d): got %v, want %v", tt.giveRes, err, tt.wantErr)
			}
		})
	}
}

// TestChildPosToCellErrors covers the resolution and child-position error cases
// for ChildPosToCell. The parent is res 8; at res 10 the maximum valid child
// position is 48.
func TestChildPosToCellErrors(t *testing.T) {
	t.Parallel()

	parent := Cell(0x88283080ddfffff)

	tests := map[string]struct {
		givePos int
		giveRes int
		wantErr error // nil means the call must succeed
	}{
		"too_high_resolution": {givePos: 27, giveRes: 42, wantErr: ErrResolutionDomain},
		"negative_resolution": {givePos: 27, giveRes: -1, wantErr: ErrResolutionDomain},
		"coarser_than_parent": {givePos: 27, giveRes: 7, wantErr: ErrResolutionMismatch},
		"negative_child_pos":  {givePos: -1, giveRes: 10, wantErr: ErrDomain},
		"max_valid_child_pos": {givePos: 48, giveRes: 10, wantErr: nil},
		"child_pos_past_max":  {givePos: 49, giveRes: 10, wantErr: ErrDomain},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ChildPosToCell(tt.givePos, parent, tt.giveRes)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ChildPosToCell(pos=%d, res=%d): unexpected error %v", tt.givePos, tt.giveRes, err)
				}

				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ChildPosToCell(pos=%d, res=%d): got %v, want %v", tt.givePos, tt.giveRes, err, tt.wantErr)
			}
		})
	}
}

// TestCompactErrors covers the error and edge cases for CompactCells.
func TestCompactErrors(t *testing.T) {
	t.Parallel()
	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		out, err := CompactCells(nil)
		if err != nil || len(out) != 0 {
			t.Fatalf("CompactCells(nil): out=%v err=%v", out, err)
		}
	})
	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()

		dup := make([]Cell, 10)
		for i := range dup {
			dup[i] = setIndexCell(5, 0, 2)
		}

		if _, err := CompactCells(dup); !errors.Is(err, ErrDuplicateInput) {
			t.Fatalf("CompactCells(dups): got %v, want ErrDuplicateInput", err)
		}
	})
	t.Run("single_duplicate_minimum", func(t *testing.T) {
		t.Parallel()
		parent := setIndexCell(10, 0, 2)

		children, err := parent.Children(11)
		if err != nil {
			t.Fatalf("Children: %v", err)
		}

		children = append(children, children[0]) // one duplicate
		if _, err := CompactCells(children); !errors.Is(err, ErrDuplicateInput) {
			t.Fatalf("CompactCells: got %v, want ErrDuplicateInput", err)
		}
	})
	t.Run("duplicate_ignored_when_set_still_full", func(t *testing.T) {
		t.Parallel()
		parent := setIndexCell(10, 0, 2)

		children, err := parent.Children(11)
		if err != nil {
			t.Fatalf("Children: %v", err)
		}
		// Overwrite the last child with a duplicate of the first; the count
		// stays at 7 so compaction still succeeds (matches C behavior).
		children[len(children)-1] = children[0]
		if _, err := CompactCells(children); err != nil {
			t.Fatalf("CompactCells: unexpected error %v", err)
		}
	})
	t.Run("disparate_no_compaction_possible", func(t *testing.T) {
		t.Parallel()

		disparate := make([]Cell, 7)
		for i := range disparate {
			disparate[i] = setIndexCell(1, i, 0)
		}

		out, err := CompactCells(disparate)
		if err != nil {
			t.Fatalf("CompactCells(disparate): %v", err)
		}

		assertSameCellSet(t, out, disparate, "disparate")
	})
	t.Run("reserved_bits_set", func(t *testing.T) {
		t.Parallel()

		bad := []Cell{
			0x0010000000010000, 0x0180c6c6c6c61616, 0x1616ffffffffffff,
			0x7fff8affffffffff, 0x7fffffffffffc6c6, 0x7fffffffffffffc6,
			0x46c6c6c6c66fffe0,
		}
		if _, err := CompactCells(bad); !errors.Is(err, ErrCellInvalid) {
			t.Fatalf("CompactCells(reserved bits): got %v, want ErrCellInvalid", err)
		}
	})
	t.Run("parent_resolution_mismatch", func(t *testing.T) {
		t.Parallel()

		bad := []Cell{setIndexCell(10, 0, 0), setIndexCell(5, 0, 0)}
		if _, err := CompactCells(bad); !errors.Is(err, ErrResolutionMismatch) {
			t.Fatalf("CompactCells(res mismatch): got %v, want ErrResolutionMismatch", err)
		}
	})
}

// TestUncompactErrors covers the wrong-resolution cases for UncompactCells.
func TestUncompactErrors(t *testing.T) {
	t.Parallel()

	cells := []Cell{setIndexCell(5, 0, 0), setIndexCell(5, 1, 0), setIndexCell(5, 2, 0)}

	tests := map[string]struct {
		giveRes int
	}{
		"coarser_than_input":  {giveRes: 4},
		"negative_resolution": {giveRes: -1},
		"too_high_resolution": {giveRes: MaxResolution + 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := UncompactCells(cells, tt.giveRes); !errors.Is(err, ErrResolutionMismatch) {
				t.Fatalf("UncompactCells(res=%d): got %v, want ErrResolutionMismatch", tt.giveRes, err)
			}
		})
	}
}

// TestImmediateChildren checks that ImmediateChildren returns the cell's
// children at the next finer resolution.
func TestImmediateChildren(t *testing.T) {
	t.Parallel()

	parent := setIndexCell(5, 0, 0)

	got, err := parent.ImmediateChildren()
	if err != nil {
		t.Fatalf("ImmediateChildren: %v", err)
	}

	want, err := parent.Children(6)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}

	assertSameCellSet(t, got, want, "ImmediateChildren")
}

// TestChildCellsInvalidInput checks that the childCells iterator yields nothing
// for inputs that cannot have children at the requested resolution.
func TestChildCellsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveCell Cell
		giveRes  int
	}{
		"zero_cell":           {giveCell: 0, giveRes: 5},
		"coarser_than_parent": {giveCell: setIndexCell(5, 0, 0), giveRes: 4},
		"finer_than_max":      {giveCell: setIndexCell(5, 0, 0), giveRes: MaxResolution + 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			count := 0
			for range tt.giveCell.childCells(tt.giveRes) {
				count++
			}

			if count != 0 {
				t.Fatalf("childCells: yielded %d cells, want 0", count)
			}
		})
	}
}

// TestChildCellsEarlyBreak checks that the childCells iterator stops when the
// consumer breaks out of the range loop.
func TestChildCellsEarlyBreak(t *testing.T) {
	t.Parallel()

	count := 0
	for range setIndexCell(5, 0, 0).childCells(7) {
		count++
		if count == 1 {
			break
		}
	}

	if count != 1 {
		t.Fatalf("childCells early break: visited %d cells, want 1", count)
	}
}

// TestValidateChildPosResError covers the resolution-error path of
// validateChildPos, which the public callers guard against before reaching it.
func TestValidateChildPosResError(t *testing.T) {
	t.Parallel()

	if err := setIndexCell(5, 0, 0).validateChildPos(0, -1); err == nil {
		t.Fatal("validateChildPos with invalid resolution should error")
	}
}

// TestCellToChildPosInvalidDigit covers the invalid-digit guards in
// CellToChildPos for both pentagon and hexagon parents.
func TestCellToChildPosInvalidDigit(t *testing.T) {
	t.Parallel()

	t.Run("pentagon_k_axis_digit", func(t *testing.T) {
		t.Parallel()

		// Pentagon base cell 4, res-2, with a deleted K-axis (1) digit.
		cell := setH3Index(2, 4, centerDigit).setIndexDigit(2, kAxesDigit)
		if _, err := CellToChildPos(cell, 1); !errors.Is(err, ErrCellInvalid) {
			t.Fatalf("CellToChildPos(pentagon k-axis): got %v, want ErrCellInvalid", err)
		}
	})

	t.Run("hexagon_invalid_digit", func(t *testing.T) {
		t.Parallel()

		// Hexagon base cell 0, res-2, with an unused (7) digit in a used slot.
		cell := setH3Index(2, 0, centerDigit).setIndexDigit(2, invalidDigit)
		if _, err := CellToChildPos(cell, 1); !errors.Is(err, ErrCellInvalid) {
			t.Fatalf("CellToChildPos(hexagon invalid digit): got %v, want ErrCellInvalid", err)
		}
	})
}

// TestUncompactSkipsZeroCells checks that UncompactCells ignores zero-valued
// entries in the input.
func TestUncompactSkipsZeroCells(t *testing.T) {
	t.Parallel()

	parent := setIndexCell(5, 0, 0)

	withZero, err := UncompactCells([]Cell{parent, 0}, 6)
	if err != nil {
		t.Fatalf("UncompactCells: %v", err)
	}

	want, err := parent.Children(6)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}

	assertSameCellSet(t, withZero, want, "UncompactCells skip zero")
}

// TestCompactResolutionZero checks that CompactCells returns a copy unchanged
// when the input is at resolution 0 (no compaction is possible).
func TestCompactResolutionZero(t *testing.T) {
	t.Parallel()

	in := []Cell{setH3Index(0, 0, centerDigit), setH3Index(0, 1, centerDigit)}

	out, err := CompactCells(in)
	if err != nil {
		t.Fatalf("CompactCells: %v", err)
	}

	assertSameCellSet(t, out, in, "CompactCells res 0")
}

// TestCompactSkipsZeroCells checks that CompactCells ignores zero-valued entries
// in the input.
func TestCompactSkipsZeroCells(t *testing.T) {
	t.Parallel()

	cell := setIndexCell(1, 0, 0)

	out, err := CompactCells([]Cell{cell, 0})
	if err != nil {
		t.Fatalf("CompactCells: %v", err)
	}

	assertSameCellSet(t, out, []Cell{cell}, "CompactCells skip zero")
}

// TestHierarchyCorpus exercises the parent/child/child-position/compaction API
// over the corpus, asserting the round-trip and self-consistency properties that
// the public functions must satisfy.
func TestHierarchyCorpus(t *testing.T) {
	t.Parallel()

	for _, c := range corpus(t) {
		res := c.Resolution()

		assertParentRoundTrip(t, c, res)

		if res >= MaxResolution {
			continue
		}

		assertChildRoundTrip(t, c, res)
	}
}

// assertParentRoundTrip checks Parent at every ancestor resolution plus
// ImmediateParent.
func assertParentRoundTrip(t *testing.T, c Cell, res int) {
	t.Helper()

	for parentRes := 0; parentRes <= res; parentRes++ {
		parent, err := c.Parent(parentRes)
		if err != nil {
			t.Fatalf("Parent(%015x, %d): %v", uint64(c), parentRes, err)
		}

		if parent.Resolution() != parentRes {
			t.Fatalf("Parent(%015x, %d): resolution %d", uint64(c), parentRes, parent.Resolution())
		}
	}

	if res == 0 {
		return
	}

	immediate, err := c.ImmediateParent()
	if err != nil {
		t.Fatalf("ImmediateParent(%015x): %v", uint64(c), err)
	}

	if immediate.Resolution() != res-1 {
		t.Fatalf("ImmediateParent(%015x): resolution %d, want %d", uint64(c), immediate.Resolution(), res-1)
	}
}

// assertChildRoundTrip checks CenterChild, Children, the child-position
// round-trip (function and method forms), and compaction at childRes = res + 1.
func assertChildRoundTrip(t *testing.T, c Cell, res int) {
	t.Helper()

	childRes := res + 1

	center, err := c.CenterChild(childRes)
	if err != nil {
		t.Fatalf("CenterChild(%015x, %d): %v", uint64(c), childRes, err)
	}

	if parent, _ := center.Parent(res); parent != c {
		t.Fatalf("CenterChild(%015x).Parent != cell", uint64(c))
	}

	children, err := c.Children(childRes)
	if err != nil {
		t.Fatalf("Children(%015x, %d): %v", uint64(c), childRes, err)
	}

	for i, child := range children {
		pos, err := child.ChildPos(res)
		if err != nil {
			t.Fatalf("ChildPos(%015x, %d): %v", uint64(child), res, err)
		}

		if pos != i {
			t.Fatalf("ChildPos(%015x): got %d, want %d", uint64(child), pos, i)
		}

		if funcPos, err := CellToChildPos(child, res); err != nil || funcPos != pos {
			t.Fatalf("CellToChildPos(%015x, %d): got %d, %v", uint64(child), res, funcPos, err)
		}

		back, err := c.ChildPosToCell(pos, childRes)
		if err != nil || back != child {
			t.Fatalf("ChildPosToCell(%d, %015x, %d): got %015x, %v", pos, uint64(c), childRes, uint64(back), err)
		}

		if funcBack, err := ChildPosToCell(pos, c, childRes); err != nil || funcBack != child {
			t.Fatalf("ChildPosToCell func(%d, %015x, %d): got %015x, %v", pos, uint64(c), childRes, uint64(funcBack), err)
		}
	}

	compact, err := CompactCells(children)
	if err != nil {
		t.Fatalf("CompactCells(%015x children): %v", uint64(c), err)
	}

	if len(compact) != 1 || compact[0] != c {
		t.Fatalf("CompactCells(%015x children): got %v, want [cell]", uint64(c), compact)
	}

	uncompact, err := UncompactCells([]Cell{c}, childRes)
	if err != nil {
		t.Fatalf("UncompactCells(%015x): %v", uint64(c), err)
	}

	assertSameCellSet(t, uncompact, children, "UncompactCells")
}

// TestCenterChildErrors covers the resolution error path of CenterChild.
func TestCenterChildErrors(t *testing.T) {
	t.Parallel()

	c := setIndexCell(5, 0, 0)

	tests := map[string]struct {
		giveRes int
	}{
		"coarser_than_cell":   {giveRes: 4},
		"too_high_resolution": {giveRes: MaxResolution + 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := c.CenterChild(tt.giveRes); !errors.Is(err, ErrResolutionDomain) {
				t.Fatalf("CenterChild(res=%d): got %v, want ErrResolutionDomain", tt.giveRes, err)
			}
		})
	}
}

// TestChildPosPentagonMultiLevel exercises the child-position round trip two
// resolutions below a pentagon, which is the only way to reach the
// non-pentagon-sublevel branches in CellToChildPos and ChildPosToCell.
func TestChildPosPentagonMultiLevel(t *testing.T) {
	t.Parallel()

	parent := setIndexCell(1, 4, 0) // res-1 pentagon (base cell 4)
	if !parent.IsPentagon() {
		t.Fatal("setup: parent should be a pentagon")
	}

	const childRes = 3 // two levels below the pentagon parent

	children, err := parent.Children(childRes)
	if err != nil {
		t.Fatalf("Children: %v", err)
	}

	for i, child := range children {
		pos, err := child.ChildPos(1)
		if err != nil {
			t.Fatalf("ChildPos(%015x): %v", uint64(child), err)
		}

		if pos != i {
			t.Fatalf("ChildPos(%015x): got %d, want %d", uint64(child), pos, i)
		}

		back, err := parent.ChildPosToCell(pos, childRes)
		if err != nil || back != child {
			t.Fatalf("ChildPosToCell(%d): got %015x, %v", pos, uint64(back), err)
		}
	}
}

// setIndexCell builds a cell from res/baseCell/initDigit by setting the index
// fields directly, so the hierarchy edge-case tests do not depend on the
// projection pipeline.
func setIndexCell(res, baseCell, initDigit int) Cell {
	const (
		modeOffset     = 59
		resOffset      = 52
		baseCellOffset = 45
		perDigit       = 3
		maxRes         = 15
		allDigitsSet   = 0x1fffffffffff // all 15 digits set to 7
	)
	h := uint64(allDigitsSet)
	h |= uint64(cellMode) << modeOffset
	h |= uint64(res) << resOffset
	h |= uint64(baseCell) << baseCellOffset

	for r := 1; r <= res; r++ {
		shift := uint((maxRes - r) * perDigit)
		h &^= uint64(0x7) << shift
		h |= uint64(initDigit) << shift
	}

	return Cell(h)
}

// sortedU64 returns the cells as a sorted slice of uint64 for set comparison.
func sortedU64[T ~int64](cells []T) []uint64 {
	out := make([]uint64, len(cells))
	for i, c := range cells {
		out[i] = uint64(c)
	}

	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })

	return out
}

// assertSameCellSet fails if the two cell slices differ as sets (order ignored).
func assertSameCellSet[A ~int64, B ~int64](t *testing.T, got []A, want []B, msg string) {
	t.Helper()
	gs := sortedU64(got)

	ws := sortedU64(want)
	if len(gs) != len(ws) {
		t.Fatalf("%s: len got=%d want=%d", msg, len(gs), len(ws))
	}

	for i := range gs {
		if gs[i] != ws[i] {
			t.Fatalf("%s: element %d got=%015x want=%015x", msg, i, gs[i], ws[i])
		}
	}
}

// TestCellToChildrenKnown ports the testCellToChildren.c oneResStep regression:
// the exact seven res-9 children of a specific res-8 hexagon.
func TestCellToChildrenKnown(t *testing.T) {
	t.Parallel()

	parent := Cell(0x88283080ddfffff)
	want := []Cell{
		0x89283080dc3ffff, 0x89283080dc7ffff, 0x89283080dcbffff,
		0x89283080dcfffff, 0x89283080dd3ffff, 0x89283080dd7ffff,
		0x89283080ddbffff,
	}

	got, err := parent.Children(9)
	if err != nil {
		t.Fatalf("Children(9): %v", err)
	}

	assertSameSet(t, got, want, "children")
}

// TestCellToChildrenMultipleResSteps ports the testCellToChildren.c
// multipleResSteps regression: the exact 49 res-10 children of a res-8 hexagon.
func TestCellToChildrenMultipleResSteps(t *testing.T) {
	t.Parallel()

	want := []Cell{
		0x8a283080dd27fff, 0x8a283080dd37fff, 0x8a283080dc47fff, 0x8a283080dcdffff,
		0x8a283080dc5ffff, 0x8a283080dc27fff, 0x8a283080ddb7fff, 0x8a283080dc07fff,
		0x8a283080dd8ffff, 0x8a283080dd5ffff, 0x8a283080dc4ffff, 0x8a283080dd47fff,
		0x8a283080dce7fff, 0x8a283080dd1ffff, 0x8a283080dceffff, 0x8a283080dc6ffff,
		0x8a283080dc87fff, 0x8a283080dcaffff, 0x8a283080dd2ffff, 0x8a283080dcd7fff,
		0x8a283080dd9ffff, 0x8a283080dd6ffff, 0x8a283080dcc7fff, 0x8a283080dca7fff,
		0x8a283080dccffff, 0x8a283080dd77fff, 0x8a283080dc97fff, 0x8a283080dd4ffff,
		0x8a283080dd97fff, 0x8a283080dc37fff, 0x8a283080dc8ffff, 0x8a283080dcb7fff,
		0x8a283080dcf7fff, 0x8a283080dd87fff, 0x8a283080dda7fff, 0x8a283080dc9ffff,
		0x8a283080dc77fff, 0x8a283080dc67fff, 0x8a283080dc57fff, 0x8a283080ddaffff,
		0x8a283080dd17fff, 0x8a283080dc17fff, 0x8a283080dd57fff, 0x8a283080dc0ffff,
		0x8a283080dd07fff, 0x8a283080dc1ffff, 0x8a283080dd0ffff, 0x8a283080dc2ffff,
		0x8a283080dd67fff,
	}

	got, err := Cell(0x88283080ddfffff).Children(10)
	if err != nil {
		t.Fatalf("Children(10): %v", err)
	}

	assertSameSet(t, got, want, "multipleResSteps")
}

// TestCellToChildrenPentagon ports the testCellToChildren.c pentagonChildren
// regression: the exact 41 res-3 children of a res-1 pentagon.
func TestCellToChildrenPentagon(t *testing.T) {
	t.Parallel()

	want := []Cell{
		0x830800fffffffff, 0x830802fffffffff, 0x830803fffffffff, 0x830804fffffffff,
		0x830805fffffffff, 0x830806fffffffff, 0x830810fffffffff, 0x830811fffffffff,
		0x830812fffffffff, 0x830813fffffffff, 0x830814fffffffff, 0x830815fffffffff,
		0x830816fffffffff, 0x830818fffffffff, 0x830819fffffffff, 0x83081afffffffff,
		0x83081bfffffffff, 0x83081cfffffffff, 0x83081dfffffffff, 0x83081efffffffff,
		0x830820fffffffff, 0x830821fffffffff, 0x830822fffffffff, 0x830823fffffffff,
		0x830824fffffffff, 0x830825fffffffff, 0x830826fffffffff, 0x830828fffffffff,
		0x830829fffffffff, 0x83082afffffffff, 0x83082bfffffffff, 0x83082cfffffffff,
		0x83082dfffffffff, 0x83082efffffffff, 0x830830fffffffff, 0x830831fffffffff,
		0x830832fffffffff, 0x830833fffffffff, 0x830834fffffffff, 0x830835fffffffff,
		0x830836fffffffff,
	}

	got, err := Cell(0x81083ffffffffff).Children(3)
	if err != nil {
		t.Fatalf("Children(3): %v", err)
	}

	assertSameSet(t, got, want, "pentagonChildren")
}

// TestCellToChildrenResTooFine ports the testCellToChildren.c childResTooFine
// regression: requesting children beyond the maximum resolution fails.
func TestCellToChildrenResTooFine(t *testing.T) {
	t.Parallel()

	if _, err := Cell(0x8f283080dcb0ae2).Children(MaxResolution + 1); !errors.Is(err, ErrResolutionDomain) {
		t.Fatalf("Children(MaxResolution+1): got %v, want ErrResolutionDomain", err)
	}
}
