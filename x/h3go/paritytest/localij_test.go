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

// localIJPairs returns origin/target cell pairs drawn from the shared corpus:
// each corpus cell paired with the cells in its grid disk, which is where local
// IJ coordinates are defined.
func localIJPairs(t *testing.T) [][2]h3.Cell {
	t.Helper()

	var pairs [][2]h3.Cell

	for _, origin := range referenceCorpus(t) {
		disk, err := h3.GridDisk(origin, 3)
		if err != nil {
			continue
		}

		for _, target := range disk {
			if target == 0 {
				continue
			}

			pairs = append(pairs, [2]h3.Cell{origin, target})
		}
	}

	return pairs
}

// TestCellToLocalIJMatchesCgo asserts that local IJ coordinates and their
// round trip match the cgo reference for origin/target pairs across the corpus,
// including error parity.
func TestCellToLocalIJMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, pair := range localIJPairs(t) {
		origin, target := pair[0], pair[1]

		want, wantErr := h3.CellToLocalIJ(origin, target)
		got, gotErr := h3go.CellToLocalIJ(h3goCell(origin), h3goCell(target))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("CellToLocalIJ(%015x, %015x) error mismatch: cgo=%v h3go=%v", uint64(origin), uint64(target), wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		if got.I != want.I || got.J != want.J {
			t.Fatalf("CellToLocalIJ(%015x, %015x): got {%d,%d}, want {%d,%d}", uint64(origin), uint64(target), got.I, got.J, want.I, want.J)
		}

		// Round trip back to the cell.
		back, err := h3go.LocalIJToCell(h3goCell(origin), h3go.CoordIJ{I: got.I, J: got.J})
		if err != nil {
			t.Fatalf("LocalIJToCell(%015x, %v): %v", uint64(origin), got, err)
		}

		if back != h3goCell(target) {
			t.Fatalf("round trip (%015x, %015x): got %015x", uint64(origin), uint64(target), uint64(back))
		}
	}
}

// TestLocalIJToCellMatchesCgo asserts LocalIJToCell matches the cgo reference,
// including error parity, by feeding back coordinates obtained from the cgo
// CellToLocalIJ plus some out-of-range offsets.
func TestLocalIJToCellMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, pair := range localIJPairs(t) {
		origin, target := pair[0], pair[1]

		ij, err := h3.CellToLocalIJ(origin, target)
		if err != nil {
			continue
		}

		for _, delta := range []h3.CoordIJ{{I: 0, J: 0}, {I: 2, J: 0}, {I: 0, J: -2}, {I: 5, J: 5}} {
			coord := h3.CoordIJ{I: ij.I + delta.I, J: ij.J + delta.J}

			want, wantErr := h3.LocalIJToCell(origin, coord)
			got, gotErr := h3go.LocalIJToCell(h3goCell(origin), h3go.CoordIJ{I: coord.I, J: coord.J})

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("LocalIJToCell(%015x, %v) error mismatch: cgo=%v h3go=%v", uint64(origin), coord, wantErr, gotErr)
			}

			if wantErr == nil && got != h3goCell(want) {
				t.Fatalf("LocalIJToCell(%015x, %v): got %015x, want %015x", uint64(origin), coord, uint64(got), uint64(want))
			}
		}
	}
}

// TestGridDistanceMatchesCgo asserts grid distance matches the cgo reference for
// origin/target pairs, including error parity.
func TestGridDistanceMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, pair := range localIJPairs(t) {
		origin, target := pair[0], pair[1]

		want, wantErr := h3.GridDistance(origin, target)
		got, gotErr := h3go.GridDistance(h3goCell(origin), h3goCell(target))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("GridDistance(%015x, %015x) error mismatch: cgo=%v h3go=%v", uint64(origin), uint64(target), wantErr, gotErr)
		}

		if wantErr == nil && got != want {
			t.Fatalf("GridDistance(%015x, %015x): got %d, want %d", uint64(origin), uint64(target), got, want)
		}
	}
}

// TestGridPathMatchesCgo asserts the grid path matches the cgo reference cell for
// cell, including error parity.
func TestGridPathMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, pair := range localIJPairs(t) {
		origin, target := pair[0], pair[1]

		want, wantErr := h3.GridPath(origin, target)
		got, gotErr := h3go.GridPath(h3goCell(origin), h3goCell(target))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("GridPath(%015x, %015x) error mismatch: cgo=%v h3go=%v", uint64(origin), uint64(target), wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		if len(got) != len(want) {
			t.Fatalf("GridPath(%015x, %015x): len got=%d want=%d", uint64(origin), uint64(target), len(got), len(want))
		}

		for i := range want {
			if got[i] != h3goCell(want[i]) {
				t.Fatalf("GridPath(%015x, %015x)[%d]: got %015x, want %015x", uint64(origin), uint64(target), i, uint64(got[i]), uint64(want[i]))
			}
		}
	}
}
