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
	"testing"
)

// localIJPairs returns origin/target pairs from the shared grid corpus: each
// cell paired with the cells in its grid disk, where local IJ is defined.
func localIJPairs(t *testing.T) [][2]Cell {
	t.Helper()

	var pairs [][2]Cell

	for _, origin := range gridCorpus(t) {
		disk, err := origin.GridDisk(2)
		if err != nil {
			continue
		}

		for _, target := range disk {
			pairs = append(pairs, [2]Cell{origin, target})
		}
	}

	return pairs
}

// TestLocalIJRoundTrip checks that CellToLocalIJ followed by LocalIJToCell
// recovers the original cell across the corpus, exercising the hexagon and
// pentagon unfolding paths in both directions.
func TestLocalIJRoundTrip(t *testing.T) {
	t.Parallel()

	for _, pair := range localIJPairs(t) {
		origin, target := pair[0], pair[1]

		ij, err := CellToLocalIJ(origin, target)
		if err != nil {
			continue
		}

		back, err := LocalIJToCell(origin, ij)
		if err != nil {
			t.Fatalf("LocalIJToCell(%015x, %v): %v", uint64(origin), ij, err)
		}

		if back != target {
			t.Fatalf("round trip (%015x, %015x): got %015x via %v", uint64(origin), uint64(target), uint64(back), ij)
		}
	}
}

// TestCellToLocalIJErrors covers the input-validation and unfolding failure
// paths of CellToLocalIJ.
func TestCellToLocalIJErrors(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	parent, err := origin.ImmediateParent()
	if err != nil {
		t.Fatalf("ImmediateParent: %v", err)
	}

	if _, err := CellToLocalIJ(origin, parent); !errors.Is(err, ErrResolutionMismatch) {
		t.Fatalf("CellToLocalIJ(res mismatch): got %v, want ErrResolutionMismatch", err)
	}

	// Base cells far apart cannot be unfolded into a common coordinate space.
	far := CellFromString("85283473fffffff")
	farSibling := CellFromString("85f29263fffffff")

	if _, err := CellToLocalIJ(far, farSibling); !errors.Is(err, ErrFailed) {
		t.Fatalf("CellToLocalIJ(non-neighbor base cells): got %v, want ErrFailed", err)
	}

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(NumBaseCells)<<baseCellOffset
	if _, err := CellToLocalIJ(bad, bad); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("CellToLocalIJ(bad base cell): got %v, want ErrCellInvalid", err)
	}

	if _, err := CellToLocalIJ(origin, origin.setBaseCell(NumBaseCells)); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("CellToLocalIJ(bad target base cell): got %v, want ErrCellInvalid", err)
	}
}

// TestLocalIJToCellErrors covers the failure paths of LocalIJToCell: an
// out-of-range coordinate and an invalid origin base cell.
func TestLocalIJToCellErrors(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	if _, err := LocalIJToCell(origin, CoordIJ{I: 1 << 20, J: -(1 << 20)}); err == nil {
		t.Fatal("LocalIJToCell(out of range): got nil error, want failure")
	}

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(NumBaseCells)<<baseCellOffset
	if _, err := LocalIJToCell(bad, CoordIJ{I: 0, J: 0}); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(bad origin): got %v, want ErrCellInvalid", err)
	}
}

// TestLocalIJRes0 covers the resolution-0 base-cell path of LocalIJToCell,
// including moving off a base cell to a neighbor and an invalid move.
func TestLocalIJRes0(t *testing.T) {
	t.Parallel()

	res0, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	for _, origin := range res0 {
		ij, err := CellToLocalIJ(origin, origin)
		if err != nil {
			t.Fatalf("CellToLocalIJ(%015x, self): %v", uint64(origin), err)
		}

		back, err := LocalIJToCell(origin, ij)
		if err != nil {
			t.Fatalf("LocalIJToCell(%015x, %v): %v", uint64(origin), ij, err)
		}

		if back != origin {
			t.Fatalf("res0 self round trip (%015x): got %015x", uint64(origin), uint64(back))
		}
	}
}

// TestGridDistance checks grid distance: zero to self, symmetry, and agreement
// with the ring a cell lies on, across the corpus.
func TestGridDistance(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		if got, err := origin.GridDistance(origin); err != nil || got != 0 {
			t.Fatalf("GridDistance(self) for %015x: got %d, %v, want 0, nil", uint64(origin), got, err)
		}

		rings, err := origin.GridDiskDistances(2)
		if err != nil {
			t.Fatalf("GridDiskDistances(%015x): %v", uint64(origin), err)
		}

		for distance, ring := range rings {
			for _, target := range ring {
				got, err := GridDistance(origin, target)
				if err != nil {
					continue
				}

				if got != distance {
					t.Fatalf("GridDistance(%015x, %015x): got %d, want %d", uint64(origin), uint64(target), got, distance)
				}

				if rev, err := target.GridDistance(origin); err == nil && rev != got {
					t.Fatalf("GridDistance not symmetric: %d vs %d", got, rev)
				}
			}
		}
	}
}

// TestGridPath checks the line of cells: correct length, endpoints, and that
// consecutive cells are neighbors, across the corpus.
func TestGridPath(t *testing.T) {
	t.Parallel()

	for _, pair := range localIJPairs(t) {
		origin, target := pair[0], pair[1]

		distance, err := origin.GridDistance(target)
		if err != nil {
			continue
		}

		path, err := GridPath(origin, target)
		if err != nil {
			continue
		}

		if len(path) != distance+1 {
			t.Fatalf("GridPath(%015x, %015x): len %d, want %d", uint64(origin), uint64(target), len(path), distance+1)
		}

		if path[0] != origin || path[len(path)-1] != target {
			t.Fatalf("GridPath(%015x, %015x): endpoints %015x..%015x", uint64(origin), uint64(target), uint64(path[0]), uint64(path[len(path)-1]))
		}

		for i := 1; i < len(path); i++ {
			neighbor, err := path[i-1].IsNeighbor(path[i])
			if err != nil {
				t.Fatalf("IsNeighbor(%015x, %015x): %v", uint64(path[i-1]), uint64(path[i]), err)
			}

			if !neighbor {
				t.Fatalf("GridPath(%015x, %015x): step %d not adjacent", uint64(origin), uint64(target), i)
			}
		}
	}
}

// TestGridPathSelf covers the zero-distance short circuit of GridPath.
func TestGridPathSelf(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	path, err := origin.GridPath(origin)
	if err != nil {
		t.Fatalf("GridPath(self): %v", err)
	}

	if len(path) != 1 || path[0] != origin {
		t.Fatalf("GridPath(self): got %v, want [origin]", path)
	}
}

// TestGridMetricsErrorsPropagate checks that GridDistance and GridPath surface
// the failure when the two cells cannot share a local coordinate space.
func TestGridMetricsErrorsPropagate(t *testing.T) {
	t.Parallel()

	far := CellFromString("85283473fffffff")
	farSibling := CellFromString("85f29263fffffff")

	if _, err := GridDistance(far, farSibling); err == nil {
		t.Fatal("GridDistance(far): got nil error, want failure")
	}

	if _, err := GridPath(far, farSibling); err == nil {
		t.Fatal("GridPath(far): got nil error, want failure")
	}

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(NumBaseCells)<<baseCellOffset
	if _, err := GridDistance(bad, bad); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("GridDistance(bad): got %v, want ErrCellInvalid", err)
	}
}

// TestGridDistanceRegressions ports the testGridDistance.c regression cases:
// distances onto and across pentagons, base-cell neighbors, and the
// resolution-mismatch and invalid-cell error contracts.
func TestGridDistanceRegressions(t *testing.T) {
	t.Parallel()

	t.Run("onto_pentagon", func(t *testing.T) {
		t.Parallel()

		origin := setH3Index(1, 17, centerDigit)
		tests := map[string]struct {
			giveDigit int
			want      int
		}{
			"digit_0": {giveDigit: 0, want: 3},
			"digit_2": {giveDigit: 2, want: 2},
			"digit_3": {giveDigit: 3, want: 3},
			"digit_6": {giveDigit: 6, want: 2},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				target := setH3Index(1, 14, tt.giveDigit)

				got, err := origin.GridDistance(target)
				if err != nil {
					t.Fatalf("GridDistance: %v", err)
				}

				if got != tt.want {
					t.Fatalf("GridDistance: got %d, want %d", got, tt.want)
				}
			})
		}
	})

	t.Run("across_pentagon_fails", func(t *testing.T) {
		t.Parallel()
		// Both directions are rejected because of pentagon distortion.
		origin := Cell(0x820c4ffffffffff)
		destination := Cell(0x821ce7fffffffff)

		if _, err := GridDistance(destination, origin); err == nil {
			t.Fatal("GridDistance across pentagon: got nil, want error")
		}

		if _, err := GridDistance(origin, destination); err == nil {
			t.Fatal("GridDistance across pentagon (reversed): got nil, want error")
		}
	})

	t.Run("base_cell_neighbors", func(t *testing.T) {
		t.Parallel()

		bc1 := setH3Index(0, 15, centerDigit)
		bc2 := setH3Index(0, 8, centerDigit)
		bc3 := setH3Index(0, 31, centerDigit)
		pent1 := setH3Index(0, 4, centerDigit)

		for _, target := range []Cell{pent1, bc2, bc3} {
			got, err := bc1.GridDistance(target)
			if err != nil {
				t.Fatalf("GridDistance(15, %015x): %v", uint64(target), err)
			}

			if got != 1 {
				t.Fatalf("GridDistance(15, %015x): got %d, want 1", uint64(target), got)
			}
		}

		if _, err := pent1.GridDistance(bc3); err == nil {
			t.Fatal("GridDistance(pent1, 31): got nil, want error")
		}
	})

	t.Run("resolution_mismatch", func(t *testing.T) {
		t.Parallel()

		_, err := GridDistance(Cell(0x832830fffffffff), Cell(0x822837fffffffff))
		if !errors.Is(err, ErrResolutionMismatch) {
			t.Fatalf("got %v, want ErrResolutionMismatch", err)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		invalid := ^Cell(0)
		if _, err := GridDistance(invalid, invalid); !errors.Is(err, ErrCellInvalid) {
			t.Fatalf("GridDistance(invalid, invalid): got %v, want ErrCellInvalid", err)
		}

		bc1 := setH3Index(0, 15, centerDigit)
		if _, err := GridDistance(bc1, invalid); !errors.Is(err, ErrResolutionMismatch) {
			t.Fatalf("GridDistance(bc1, invalid): got %v, want ErrResolutionMismatch", err)
		}
	})
}

// TestGridPathRegressions ports the testGridPathCells.c named cases: lines that
// must fail (crossing multiple faces / pentagon distortion) and one that must
// succeed where a naive forward interpolation would fail.
func TestGridPathRegressions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveOrigin Cell
		giveDest   Cell
		wantErr    bool
	}{
		"across_multiple_faces_fails": {
			giveOrigin: Cell(0x85285aa7fffffff),
			giveDest:   Cell(0x851d9b1bfffffff),
			wantErr:    true,
		},
		"pentagon_reverse_interpolation": {
			giveOrigin: Cell(0x820807fffffffff),
			giveDest:   Cell(0x8208e7fffffffff),
			wantErr:    false,
		},
		"known_failure": {
			giveOrigin: Cell(0x8411b61ffffffff),
			giveDest:   Cell(0x84016d3ffffffff),
			wantErr:    true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path, err := GridPath(tt.giveOrigin, tt.giveDest)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("GridPath: got %v, want error", path)
				}

				return
			}

			if err != nil {
				t.Fatalf("GridPath: %v", err)
			}

			if path[0] != tt.giveOrigin || path[len(path)-1] != tt.giveDest {
				t.Fatalf("GridPath: endpoints %015x..%015x", uint64(path[0]), uint64(path[len(path)-1]))
			}
		})
	}
}

// TestLocalIJPentagonUnfold exercises the pentagon unfolding paths in both
// directions: each pentagon paired with the cells around it as both origin and
// target. Reachable pairs round trip; unreachable ones (across pentagon
// distortion) return an error, exercising the unfold failure branches.
func TestLocalIJPentagonUnfold(t *testing.T) {
	t.Parallel()

	for _, res := range []int{1, 2, 3} {
		pents, err := Pentagons(res)
		if err != nil {
			t.Fatalf("Pentagons(%d): %v", res, err)
		}

		for _, pentagon := range pents {
			neighborhood, err := pentagon.GridDisk(4)
			if err != nil {
				t.Fatalf("GridDisk(%015x): %v", uint64(pentagon), err)
			}

			for _, cell := range neighborhood {
				assertLocalIJRoundTrip(t, pentagon, cell)
				assertLocalIJRoundTrip(t, cell, pentagon)
			}
		}
	}
}

// assertLocalIJRoundTrip checks that, when CellToLocalIJ succeeds, LocalIJToCell
// recovers the target. Failures are allowed (pentagon distortion) and skipped.
func assertLocalIJRoundTrip(t *testing.T, origin, target Cell) {
	t.Helper()

	ij, err := CellToLocalIJ(origin, target)
	if err != nil {
		return
	}

	back, err := LocalIJToCell(origin, ij)
	if err != nil {
		return
	}

	if back != target {
		t.Fatalf("round trip (%015x, %015x): got %015x via %v", uint64(origin), uint64(target), uint64(back), ij)
	}
}

// TestLocalIJRes0Errors covers the resolution-0 failure paths of LocalIJToCell:
// an out-of-range coordinate and a move off a pentagon in the deleted direction.
func TestLocalIJRes0Errors(t *testing.T) {
	t.Parallel()

	res0, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	if _, err := LocalIJToCell(res0[0], CoordIJ{I: 5, J: 5}); !errors.Is(err, ErrFailed) {
		t.Fatalf("LocalIJToCell(res0, out of range): got %v, want ErrFailed", err)
	}

	pents, err := Pentagons(0)
	if err != nil {
		t.Fatalf("Pentagons(0): %v", err)
	}

	// The k-axis direction is deleted at a pentagon; {I:-1,J:-1} maps to it.
	if _, err := LocalIJToCell(pents[0], CoordIJ{I: -1, J: -1}); !errors.Is(err, ErrFailed) {
		t.Fatalf("LocalIJToCell(res0 pentagon, deleted dir): got %v, want ErrFailed", err)
	}
}

// TestGridPathLongLine exercises a longer interpolated line, reaching the
// rounding cases of the cube-coordinate path construction.
func TestGridPathLongLine(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	rings, err := origin.GridDiskDistances(6)
	if err != nil {
		t.Fatalf("GridDiskDistances: %v", err)
	}

	for _, target := range rings[6] {
		path, err := GridPath(origin, target)
		if err != nil {
			continue
		}

		if len(path) != 7 {
			t.Fatalf("GridPath(%015x, %015x): len %d, want 7", uint64(origin), uint64(target), len(path))
		}
	}
}

// res2Cell builds a resolution-2 cell with the given base cell and center digits.
func res2Cell(baseCell int) Cell {
	cell := Cell(h3Init) | Cell(cellMode)<<modeOffset
	cell = cell.setResolution(2).setBaseCell(baseCell)
	cell = cell.setIndexDigit(1, centerDigit).setIndexDigit(2, centerDigit)

	return cell
}

// TestLocalIJDefensive covers the invalid-input branches of the local IJ
// conversions that a validly constructed cell cannot reach, using malformed
// pentagon indexes whose corrupt leading digit drives the failure paths.
func TestLocalIJDefensive(t *testing.T) {
	t.Parallel()

	pent, err := Pentagons(2)
	if err != nil {
		t.Fatalf("Pentagons(2): %v", err)
	}

	pentagon := pent[0]
	baseCell := pentagon.BaseCellNumber()

	neighborBC := invalidBaseCell

	for dir := 1; dir < numDigits; dir++ {
		candidate := baseCellNeighbors[baseCell][dir]
		if candidate != invalidBaseCell && candidate != baseCell {
			neighborBC = candidate
			break
		}
	}

	neighbor := res2Cell(neighborBC)

	// A pentagon index whose leading digit is the unused sentinel.
	corruptLead := pentagon.setIndexDigit(1, invalidDigit)
	// A pentagon index whose leading digit is the deleted k axis.
	corruptK := pentagon.setIndexDigit(1, kAxesDigit)

	// cellToLocalIjk: invalid leading digit on the pentagon origin / index.
	if _, err := corruptLead.cellToLocalIjk(neighbor); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("cellToLocalIjk(corrupt origin): got %v, want ErrCellInvalid", err)
	}

	if _, err := neighbor.cellToLocalIjk(corruptLead); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("cellToLocalIjk(corrupt index): got %v, want ErrCellInvalid", err)
	}

	if _, err := corruptLead.cellToLocalIjk(pentagon); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("cellToLocalIjk(corrupt same pentagon): got %v, want ErrCellInvalid", err)
	}

	// cellToLocalIjk: a k-axis leading digit selects a -1 rotation entry.
	if _, err := corruptK.cellToLocalIjk(neighbor); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("cellToLocalIjk(k-axis origin): got %v, want ErrCellInvalid", err)
	}

	// localIjkToCell via LocalIJToCell, using coordinates that map to a neighbor
	// (dir != center) and to the origin itself (dir == center).
	disk, err := pentagon.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	var neighborCell Cell

	for _, cell := range disk {
		if cell != pentagon {
			neighborCell = cell
			break
		}
	}

	neighborIJ, err := CellToLocalIJ(pentagon, neighborCell)
	if err != nil {
		t.Fatalf("CellToLocalIJ(neighbor): %v", err)
	}

	selfIJ, err := CellToLocalIJ(pentagon, pentagon)
	if err != nil {
		t.Fatalf("CellToLocalIJ(self): %v", err)
	}

	if _, err := LocalIJToCell(corruptLead, neighborIJ); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(corrupt origin, neighbor): got %v, want ErrCellInvalid", err)
	}

	if _, err := LocalIJToCell(corruptLead, selfIJ); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(corrupt origin, self): got %v, want ErrCellInvalid", err)
	}

	if _, err := LocalIJToCell(corruptK, neighborIJ); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(k-axis origin, neighbor): got %v, want ErrCellInvalid", err)
	}

	if _, err := LocalIJToCell(corruptK, selfIJ); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(k-axis origin, self): got %v, want ErrCellInvalid", err)
	}

	// A coordinate that resolves to a different base cell (dir != center at the
	// base-cell level) drives the localIjkToCell pentagon-origin branches that a
	// same-base-cell neighbor cannot reach. Such a cell is several rings out.
	disk4, err := pentagon.GridDisk(4)
	if err != nil {
		t.Fatalf("GridDisk(4): %v", err)
	}

	var crossCell Cell

	for _, cell := range disk4 {
		if cell.BaseCellNumber() != baseCell {
			crossCell = cell
			break
		}
	}

	crossIJ, err := CellToLocalIJ(pentagon, crossCell)
	if err != nil {
		t.Fatalf("CellToLocalIJ(cross): %v", err)
	}

	if _, err := LocalIJToCell(corruptLead, crossIJ); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(corrupt origin, cross): got %v, want ErrCellInvalid", err)
	}

	if _, err := LocalIJToCell(corruptK, crossIJ); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LocalIJToCell(k-axis origin, cross): got %v, want ErrCellInvalid", err)
	}

	// A coordinate pointing in the deleted k direction is undefined at a pentagon.
	kOffset := coordIJK{}.neighbor(kAxesDigit)

	for r := pentagon.Resolution() - 1; r >= 0; r-- {
		if isResClassIII(r + 1) {
			kOffset.downAp7()
		} else {
			kOffset.downAp7r()
		}
	}

	if _, err := pentagon.localIjkToCell(kOffset); !errors.Is(err, ErrPentagon) {
		t.Fatalf("localIjkToCell(k direction): got %v, want ErrPentagon", err)
	}
}

// TestCellToLocalIJPentagonUnfoldFails covers the index-on-pentagon failed
// unfold branch: a hexagon origin and an index in a neighboring pentagon base
// cell whose relative direction cannot be unfolded across an icosahedron face.
func TestCellToLocalIJPentagonUnfoldFails(t *testing.T) {
	t.Parallel()

	origin := CellFromString("81003ffffffffff")
	index := CellFromString("8108bffffffffff")

	if _, err := CellToLocalIJ(origin, index); !errors.Is(err, ErrFailed) {
		t.Fatalf("CellToLocalIJ(hex, pentagon-base-cell across faces): got %v, want ErrFailed", err)
	}
}

// TestGridPathInterpolateError covers the self-conversion error path of the
// interpolation helper, reached when its anchor origin is itself invalid.
func TestGridPathInterpolateError(t *testing.T) {
	t.Parallel()

	pent, err := Pentagons(2)
	if err != nil {
		t.Fatalf("Pentagons(2): %v", err)
	}

	corrupt := pent[0].setIndexDigit(1, invalidDigit)
	out := make([]Cell, 2)

	if err := gridPathInterpolate(corrupt, pent[0], 1, out, 0, 1); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("gridPathInterpolate(corrupt anchor): got %v, want ErrCellInvalid", err)
	}
}

// TestLocalIJToCellOverflow ports the testCellToLocalIj.c overflow regressions:
// high-magnitude IJ coordinates (and a few hand-picked particular cases) must
// fail rather than overflow the internal cube-coordinate conversion, at every
// resolution.
func TestLocalIJToCellOverflow(t *testing.T) {
	t.Parallel()

	coords := []CoordIJ{
		{I: math.MinInt32, J: math.MaxInt32},
		{I: math.MaxInt32, J: math.MinInt32},
		{I: math.MinInt32, J: math.MinInt32},
		{I: 553648127, J: -2145378272},
		{I: math.MaxInt32 - 10, J: -11},
		{I: math.MaxInt32 - 10, J: -10},
		{I: math.MaxInt32 - 10, J: -9},
	}

	for res := 0; res <= MaxResolution; res++ {
		origin := setH3Index(res, 2, centerDigit)
		for _, ij := range coords {
			if _, err := LocalIJToCell(origin, ij); err == nil {
				t.Fatalf("LocalIJToCell(res %d, %+v): got nil error, want failure", res, ij)
			}
		}
	}
}

// TestLocalIJToCellNegative ports the testCellToLocalIj.c invalid_negativeIj
// regression: a specific index with large-negative IJ components must fail.
func TestLocalIJToCellNegative(t *testing.T) {
	t.Parallel()

	index := Cell(0x200f202020202020)
	ij := CoordIJ{I: -14671840, J: math.MinInt32}

	if _, err := LocalIJToCell(index, ij); err == nil {
		t.Fatal("LocalIJToCell(negative): got nil error, want failure")
	}
}

// TestLocalIJToCellBaseCells ports the testCellToLocalIj.c ijBaseCells
// regression: exact IJ->cell mappings around a res-0 base cell, including the
// out-of-range failures.
func TestLocalIJToCellBaseCells(t *testing.T) {
	t.Parallel()

	origin := Cell(0x8029fffffffffff)

	tests := map[string]struct {
		giveIJ  CoordIJ
		want    Cell
		wantErr bool
	}{
		"self":         {giveIJ: CoordIJ{I: 0, J: 0}, want: 0x8029fffffffffff},
		"i1":           {giveIJ: CoordIJ{I: 1, J: 0}, want: 0x8051fffffffffff},
		"i2_out":       {giveIJ: CoordIJ{I: 2, J: 0}, wantErr: true},
		"j2_out":       {giveIJ: CoordIJ{I: 0, J: 2}, wantErr: true},
		"negative_out": {giveIJ: CoordIJ{I: -2, J: -2}, wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := LocalIJToCell(origin, tt.giveIJ)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %015x, want error", uint64(got))
				}

				return
			}

			if err != nil || got != tt.want {
				t.Fatalf("got %015x (%v), want %015x", uint64(got), err, uint64(tt.want))
			}
		})
	}
}

// TestLocalIJToCellOutOfRange ports the testCellToLocalIj.c ijOutOfRange
// regression: exact IJ->cell mappings along the i axis, with the far
// coordinates failing.
func TestLocalIJToCellOutOfRange(t *testing.T) {
	t.Parallel()

	origin := Cell(0x81283ffffffffff)

	tests := map[string]struct {
		giveIJ  CoordIJ
		want    Cell
		wantErr bool
	}{
		"i0":      {giveIJ: CoordIJ{I: 0, J: 0}, want: 0x81283ffffffffff},
		"i1":      {giveIJ: CoordIJ{I: 1, J: 0}, want: 0x81293ffffffffff},
		"i2":      {giveIJ: CoordIJ{I: 2, J: 0}, want: 0x8150bffffffffff},
		"i3":      {giveIJ: CoordIJ{I: 3, J: 0}, want: 0x8151bffffffffff},
		"i4_out":  {giveIJ: CoordIJ{I: 4, J: 0}, wantErr: true},
		"in4_out": {giveIJ: CoordIJ{I: -4, J: 0}, wantErr: true},
		"j4_out":  {giveIJ: CoordIJ{I: 0, J: 4}, wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := LocalIJToCell(origin, tt.giveIJ)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("got %015x, want error", uint64(got))
				}

				return
			}

			if err != nil || got != tt.want {
				t.Fatalf("got %015x (%v), want %015x", uint64(got), err, uint64(tt.want))
			}
		})
	}
}
