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

package paritytest

import (
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// TestParentMatchesCgo sweeps every parent resolution of every corpus cell.
func TestParentMatchesCgo(t *testing.T) {
	t.Parallel()
	corpus := referenceCorpus(t)

	t.Run("Parent", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			c := h3goCell(ref)
			for pr := -1; pr <= h3.MaxResolution+1; pr++ {
				got, gErr := c.Parent(pr)

				want, wErr := ref.Parent(pr)
				if !bothErr(gErr, wErr) {
					t.Fatalf("Parent(%015x, %d): err got=%v want=%v", uint64(ref), pr, gErr, wErr)
				}

				if wErr == nil && uint64(got) != uint64(want) {
					t.Fatalf("Parent(%015x, %d): got %015x want %015x", uint64(ref), pr, uint64(got), uint64(want))
				}
			}
		}
	})

	t.Run("ImmediateParent", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			got, gErr := h3goCell(ref).ImmediateParent()

			want, wErr := ref.ImmediateParent()
			if !bothErr(gErr, wErr) {
				t.Fatalf("ImmediateParent(%015x): err got=%v want=%v", uint64(ref), gErr, wErr)
			}

			if wErr == nil && uint64(got) != uint64(want) {
				t.Fatalf("ImmediateParent(%015x): got %015x want %015x", uint64(ref), uint64(got), uint64(want))
			}
		}
	})
}

// TestCenterChildAndChildrenMatchCgo checks CenterChild and Children (as a set)
// for small resolution steps against the cgo reference.
func TestCenterChildAndChildrenMatchCgo(t *testing.T) {
	t.Parallel()
	corpus := referenceCorpus(t)

	t.Run("CenterChild", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			c := h3goCell(ref)

			res := ref.Resolution()
			for step := 0; step <= 2; step++ {
				childRes := res + step
				got, gErr := c.CenterChild(childRes)

				want, wErr := ref.CenterChild(childRes)
				if !bothErr(gErr, wErr) {
					t.Fatalf("CenterChild(%015x, %d): err got=%v want=%v", uint64(ref), childRes, gErr, wErr)
				}

				if wErr == nil && uint64(got) != uint64(want) {
					t.Fatalf("CenterChild(%015x, %d): got %015x want %015x", uint64(ref), childRes, uint64(got), uint64(want))
				}
			}
		}
	})

	t.Run("Children", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			c := h3goCell(ref)

			res := ref.Resolution()
			for step := 0; step <= 2; step++ {
				childRes := res + step
				got, gErr := c.Children(childRes)

				want, wErr := ref.Children(childRes)
				if !bothErr(gErr, wErr) {
					t.Fatalf("Children(%015x, %d): err got=%v want=%v", uint64(ref), childRes, gErr, wErr)
				}

				if wErr == nil {
					assertSameCellSet(t, got, want, "Children")
				}
			}
		}
	})
}

// TestChildPosRoundTrip verifies the child-position round-trip: for each child,
// CellToChildPos must equal its iteration index and ChildPosToCell must recover
// it.
func TestChildPosRoundTrip(t *testing.T) {
	t.Parallel()

	for _, ref := range referenceCorpus(t) {
		c := h3goCell(ref)

		parentRes := ref.Resolution()
		for step := 0; step <= 2; step++ {
			childRes := parentRes + step
			if childRes > h3.MaxResolution {
				continue
			}

			children, err := c.Children(childRes)
			if err != nil {
				t.Fatalf("Children(%015x, %d): %v", uint64(ref), childRes, err)
			}

			for i, child := range children {
				pos, err := child.ChildPos(parentRes)
				if err != nil {
					t.Fatalf("ChildPos(%015x, %d): %v", uint64(child), parentRes, err)
				}

				if pos != i {
					t.Fatalf("ChildPos(%015x): got %d, want %d", uint64(child), pos, i)
				}

				back, err := c.ChildPosToCell(pos, childRes)
				if err != nil {
					t.Fatalf("ChildPosToCell(%d, %015x, %d): %v", pos, uint64(ref), childRes, err)
				}

				if back != child {
					t.Fatalf("ChildPosToCell: got %015x, want %015x", uint64(back), uint64(child))
				}
			}
		}
	}
}

// TestCompactUncompactMatchesCgo compares compaction/uncompaction against the
// cgo reference (as sets) and exercises a parent/children round-trip.
func TestCompactUncompactMatchesCgo(t *testing.T) {
	t.Parallel()
	corpus := referenceCorpus(t)

	t.Run("CompactCells", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			res := ref.Resolution()
			if res == h3.MaxResolution {
				continue
			}

			refChildren, err := ref.Children(res + 1)
			if err != nil {
				t.Fatalf("Children(%015x): %v", uint64(ref), err)
			}
			// Compact a full child set: both implementations should yield the parent.
			goCompact, err := h3go.CompactCells(toGoCells(refChildren))
			if err != nil {
				t.Fatalf("CompactCells(%015x children): %v", uint64(ref), err)
			}

			refCompact, err := h3.CompactCells(refChildren)
			if err != nil {
				t.Fatalf("h3.CompactCells: %v", err)
			}

			assertSameCellSet(t, goCompact, refCompact, "CompactCells")

			if len(goCompact) != 1 || uint64(goCompact[0]) != uint64(ref) {
				t.Fatalf("CompactCells(%015x children): got %v, want [parent]", uint64(ref), goCompact)
			}
		}
	})

	t.Run("UncompactCells", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			res := ref.Resolution()
			if res == h3.MaxResolution {
				continue
			}
			childRes := res + 1

			refChildren, err := ref.Children(childRes)
			if err != nil {
				t.Fatalf("Children(%015x): %v", uint64(ref), err)
			}
			// Uncompact the parent back to the child set.
			goUncompact, err := h3go.UncompactCells([]h3go.Cell{h3goCell(ref)}, childRes)
			if err != nil {
				t.Fatalf("UncompactCells(%015x): %v", uint64(ref), err)
			}

			assertSameCellSet(t, goUncompact, refChildren, "UncompactCells")
		}
	})
}
