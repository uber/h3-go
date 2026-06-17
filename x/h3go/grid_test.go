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
	"testing"
)

// gridCorpus builds a deterministic set of cells exercising hexagons, pentagons,
// pentagon children (which reach the distortion paths), and cells around the
// poles and antimeridian. It is shared read-only across the grid tests.
func gridCorpus(t *testing.T) []Cell {
	t.Helper()

	res0, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	cells := append([]Cell{}, res0...)

	points := []LatLng{
		{Lat: 0, Lng: 0},
		{Lat: 37.7749, Lng: -122.4194},
		{Lat: 67.1509, Lng: -168.3908},
		{Lat: -45, Lng: 170},
		{Lat: 89.9, Lng: 0},
	}

	for res := 0; res <= 6; res++ {
		for _, point := range points {
			if cell, err := LatLngToCell(point, res); err == nil {
				cells = append(cells, cell)
			}
		}

		pents, err := Pentagons(res)
		if err != nil {
			t.Fatalf("Pentagons(%d): %v", res, err)
		}

		cells = append(cells, pents...)

		// Children of a pentagon reach the pentagon-distortion branches when
		// their disks and rings are traced; two generations down places cells
		// right against the deleted edge, where the fast traversals fail.
		for _, pent := range pents {
			children, err := pent.Children(res + 2)
			if err != nil {
				t.Fatalf("Children(%015x): %v", uint64(pent), err)
			}

			cells = append(cells, children...)
		}
	}

	return cells
}

// cellSet returns the cells of in as a set for membership tests, dropping zeros.
func cellSet(in []Cell) map[Cell]bool {
	set := make(map[Cell]bool, len(in))
	for _, cell := range in {
		if cell != 0 {
			set[cell] = true
		}
	}

	return set
}

// TestGridDiskDistancesConsistency checks the internal invariants of the disk
// family over the corpus: the origin sits alone in ring 0, GridDisk equals the
// flattened distances, and the safe and (when it succeeds) unsafe traversals
// agree as sets.
func TestGridDiskDistancesConsistency(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		for _, k := range []int{0, 1, 2, 3} {
			rings, err := origin.GridDiskDistances(k)
			if err != nil {
				t.Fatalf("GridDiskDistances(%015x, %d): %v", uint64(origin), k, err)
			}

			if len(rings) != k+1 {
				t.Fatalf("GridDiskDistances(%015x, %d): %d rings, want %d", uint64(origin), k, len(rings), k+1)
			}

			if len(rings[0]) != 1 || rings[0][0] != origin {
				t.Fatalf("GridDiskDistances(%015x, %d): ring 0 = %v, want [origin]", uint64(origin), k, rings[0])
			}

			disk, err := origin.GridDisk(k)
			if err != nil {
				t.Fatalf("GridDisk(%015x, %d): %v", uint64(origin), k, err)
			}

			var flat int
			for _, ring := range rings {
				flat += len(ring)
			}

			if len(disk) != flat {
				t.Fatalf("GridDisk(%015x, %d): %d cells, flattened distances had %d", uint64(origin), k, len(disk), flat)
			}

			safe, err := origin.GridDiskDistancesSafe(k)
			if err != nil {
				t.Fatalf("GridDiskDistancesSafe(%015x, %d): %v", uint64(origin), k, err)
			}

			unsafe, unsafeErr := origin.GridDiskDistancesUnsafe(k)
			if unsafeErr == nil {
				for ring := range safe {
					assertSameSet(t, unsafe[ring], safe[ring], "unsafe vs safe ring")
				}
			}
		}
	}
}

// TestGridRingMatchesDiskRing checks that GridRing yields exactly the cells at
// distance k from the disk's ring k, over the corpus, and that the free function
// delegates to the method.
func TestGridRingMatchesDiskRing(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		for _, k := range []int{0, 1, 2, 3} {
			rings, err := origin.GridDiskDistances(k)
			if err != nil {
				t.Fatalf("GridDiskDistances(%015x, %d): %v", uint64(origin), k, err)
			}

			ring, err := GridRing(origin, k)
			if err != nil {
				t.Fatalf("GridRing(%015x, %d): %v", uint64(origin), k, err)
			}

			assertSameSet(t, ring, rings[k], "GridRing vs disk ring")
		}
	}
}

// TestGridRingUnsafeMatchesGridRing checks that when GridRingUnsafe succeeds its
// result matches GridRing as a set, exercising the fast single-loop traversal.
func TestGridRingUnsafeMatchesGridRing(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		for _, k := range []int{1, 2, 3, 4, 5} {
			unsafe, err := GridRingUnsafe(origin, k)
			if err != nil {
				continue
			}

			ring, err := origin.GridRing(k)
			if err != nil {
				t.Fatalf("GridRing(%015x, %d): %v", uint64(origin), k, err)
			}

			assertSameSet(t, unsafe, ring, "GridRingUnsafe vs GridRing")
		}
	}
}

// TestGridDisksUnsafe covers the batch traversal: the empty-input short circuit
// and equivalence with per-origin GridDiskDistancesUnsafe over a hexagon-only
// subset (the batch fails fast on any pentagon).
func TestGridDisksUnsafe(t *testing.T) {
	t.Parallel()

	if out, err := GridDisksUnsafe(nil, 2); out != nil || err != nil {
		t.Fatalf("GridDisksUnsafe(nil): got %v, %v, want nil, nil", out, err)
	}

	origins := []Cell{
		CellFromString("8928308280fffff"),
		CellFromString("85283473fffffff"),
	}

	const k = 2

	batch, err := GridDisksUnsafe(origins, k)
	if err != nil {
		t.Fatalf("GridDisksUnsafe: %v", err)
	}

	for i, origin := range origins {
		rings, err := origin.GridDiskDistancesUnsafe(k)
		if err != nil {
			t.Fatalf("GridDiskDistancesUnsafe(%015x): %v", uint64(origin), err)
		}

		var flat []Cell
		for _, ring := range rings {
			flat = append(flat, ring...)
		}

		assertSameSet(t, batch[i], flat, "batch vs per-origin disk")
	}
}

// TestGridDisksUnsafePentagonFails checks that the batch traversal propagates the
// pentagon failure from any single origin.
func TestGridDisksUnsafePentagonFails(t *testing.T) {
	t.Parallel()

	pents, err := Pentagons(5)
	if err != nil {
		t.Fatalf("Pentagons(5): %v", err)
	}

	origins := append([]Cell{CellFromString("8928308280fffff")}, pents[0])
	if _, err := GridDisksUnsafe(origins, 1); !errors.Is(err, ErrPentagon) {
		t.Fatalf("GridDisksUnsafe with pentagon: got %v, want ErrPentagon", err)
	}
}

// TestGridDomainErrors checks that every variant rejects a negative k with
// ErrDomain.
func TestGridDomainErrors(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	if _, err := origin.GridDiskDistancesUnsafe(-1); !errors.Is(err, ErrDomain) {
		t.Fatalf("GridDiskDistancesUnsafe(-1): got %v, want ErrDomain", err)
	}

	if _, err := origin.GridDiskDistancesSafe(-1); !errors.Is(err, ErrDomain) {
		t.Fatalf("GridDiskDistancesSafe(-1): got %v, want ErrDomain", err)
	}

	if _, err := origin.GridDisk(-1); !errors.Is(err, ErrDomain) {
		t.Fatalf("GridDisk(-1): got %v, want ErrDomain", err)
	}

	if _, err := origin.GridRingUnsafe(-1); !errors.Is(err, ErrDomain) {
		t.Fatalf("GridRingUnsafe(-1): got %v, want ErrDomain", err)
	}

	if _, err := origin.GridRing(-1); !errors.Is(err, ErrDomain) {
		t.Fatalf("GridRing(-1): got %v, want ErrDomain", err)
	}
}

// TestGridPentagonFallback checks that the unsafe variants report ErrPentagon on
// a pentagon origin while the safe/auto variants succeed via fallback.
func TestGridPentagonFallback(t *testing.T) {
	t.Parallel()

	pents, err := Pentagons(3)
	if err != nil {
		t.Fatalf("Pentagons(3): %v", err)
	}

	pentagon := pents[0]

	if _, err := pentagon.GridDiskDistancesUnsafe(2); !errors.Is(err, ErrPentagon) {
		t.Fatalf("GridDiskDistancesUnsafe(pentagon): got %v, want ErrPentagon", err)
	}

	if _, err := pentagon.GridRingUnsafe(2); !errors.Is(err, ErrPentagon) {
		t.Fatalf("GridRingUnsafe(pentagon): got %v, want ErrPentagon", err)
	}

	if _, err := pentagon.GridDisk(2); err != nil {
		t.Fatalf("GridDisk(pentagon): %v", err)
	}

	if _, err := pentagon.GridRing(2); err != nil {
		t.Fatalf("GridRing(pentagon): %v", err)
	}
}

// TestNeighborRotationsErrors covers the error paths of neighborRotations that a
// validly constructed cell cannot reach through the public traversal API.
func TestNeighborRotationsErrors(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	if _, _, err := origin.neighborRotations(invalidDigit, 0); !errors.Is(err, ErrFailed) {
		t.Fatalf("neighborRotations(invalidDigit): got %v, want ErrFailed", err)
	}

	if _, _, err := origin.neighborRotations(-1, 0); !errors.Is(err, ErrFailed) {
		t.Fatalf("neighborRotations(-1): got %v, want ErrFailed", err)
	}

	badBaseCell := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(numBaseCells)<<baseCellOffset
	if _, _, err := badBaseCell.neighborRotations(kAxesDigit, 0); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("neighborRotations(bad base cell): got %v, want ErrCellInvalid", err)
	}

	badDigit := origin.setIndexDigit(origin.Resolution(), invalidDigit)
	if _, _, err := badDigit.neighborRotations(kAxesDigit, 0); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("neighborRotations(invalid digit): got %v, want ErrCellInvalid", err)
	}

	// A malformed pentagon index can land in the deleted k subsequence with a
	// leading digit the pentagon adjustment does not define, which is rejected.
	if _, _, err := Cell(0x810840000000000).neighborRotations(centerDigit, 0); !errors.Is(err, ErrFailed) {
		t.Fatalf("neighborRotations(undefined pentagon move): got %v, want ErrFailed", err)
	}
}

// TestGridDefensiveErrors covers the error-propagation paths in the traversals
// that a validly constructed cell cannot reach, using malformed indexes whose
// invalid digits make the internal neighbor step fail partway through.
func TestGridDefensiveErrors(t *testing.T) {
	t.Parallel()

	base := CellFromString("8928308280fffff")

	// firstStep fails on the move to the next ring; secondStep passes that move
	// but fails while tracing the ring. Together they exercise both neighbor-step
	// error returns in the fast disk and ring traversals.
	firstStep := base.setIndexDigit(9, invalidDigit)
	secondStep := base.setIndexDigit(8, invalidDigit)

	for _, malformed := range []Cell{firstStep, secondStep} {
		if _, err := malformed.GridDiskDistancesUnsafe(1); err == nil {
			t.Fatalf("GridDiskDistancesUnsafe(%015x): got nil error, want failure", uint64(malformed))
		}

		if _, err := malformed.GridRingUnsafe(1); err == nil {
			t.Fatalf("GridRingUnsafe(%015x): got nil error, want failure", uint64(malformed))
		}
	}

	// The safe traversal surfaces the same non-pentagon failure, and the disk
	// and ring fallbacks propagate it.
	if _, err := firstStep.GridDiskDistancesSafe(1); err == nil {
		t.Fatal("GridDiskDistancesSafe(malformed): got nil error, want failure")
	}

	if _, err := firstStep.GridDisk(1); err == nil {
		t.Fatal("GridDisk(malformed): got nil error, want failure")
	}

	if _, err := firstStep.GridRing(1); err == nil {
		t.Fatal("GridRing(malformed): got nil error, want failure")
	}
}

// TestNeighborRotationsResetByRotations checks that the rotations argument is
// reduced modulo 6 and applied to the direction, so a six-fold rotation is a
// no-op relative to none.
func TestNeighborRotationsResetByRotations(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	base, baseRot, err := origin.neighborRotations(kAxesDigit, 0)
	if err != nil {
		t.Fatalf("neighborRotations(0 rotations): %v", err)
	}

	six, sixRot, err := origin.neighborRotations(kAxesDigit, 6)
	if err != nil {
		t.Fatalf("neighborRotations(6 rotations): %v", err)
	}

	if base != six || baseRot != sixRot {
		t.Fatalf("rotations not modulo 6: (%015x,%d) vs (%015x,%d)", uint64(base), baseRot, uint64(six), sixRot)
	}
}

// TestIsNeighbor covers the neighbor relation: self is never a neighbor, true
// neighbors from the ring, non-neighbors, resolution mismatch, and non-cell mode.
func TestIsNeighbor(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	if got, err := origin.IsNeighbor(origin); err != nil || got {
		t.Fatalf("IsNeighbor(self): got %v, %v, want false, nil", got, err)
	}

	ring, err := origin.GridRing(1)
	if err != nil {
		t.Fatalf("GridRing(1): %v", err)
	}

	for _, neighbor := range ring {
		if got, err := origin.IsNeighbor(neighbor); err != nil || !got {
			t.Fatalf("IsNeighbor(%015x, %015x): got %v, %v, want true, nil", uint64(origin), uint64(neighbor), got, err)
		}
	}

	far, err := LatLngToCell(LatLng{Lat: -33.8688, Lng: 151.2093}, 9)
	if err != nil {
		t.Fatalf("LatLngToCell(far): %v", err)
	}

	if got, err := origin.IsNeighbor(far); err != nil || got {
		t.Fatalf("IsNeighbor(far): got %v, %v, want false, nil", got, err)
	}

	parent, err := origin.ImmediateParent()
	if err != nil {
		t.Fatalf("ImmediateParent: %v", err)
	}

	if _, err := origin.IsNeighbor(parent); !errors.Is(err, ErrResolutionMismatch) {
		t.Fatalf("IsNeighbor(parent): got %v, want ErrResolutionMismatch", err)
	}

	notACell := origin &^ (Cell(modeMask) << modeOffset)
	if _, err := notACell.IsNeighbor(origin); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("IsNeighbor(non-cell): got %v, want ErrCellInvalid", err)
	}

	if _, err := origin.IsNeighbor(notACell); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("IsNeighbor(other non-cell): got %v, want ErrCellInvalid", err)
	}
}

// TestIsNeighborPentagon exercises the optimized same-parent branch around a
// pentagon, where a k-axis digit under a pentagon parent is rejected as invalid.
func TestIsNeighborPentagon(t *testing.T) {
	t.Parallel()

	pents, err := Pentagons(5)
	if err != nil {
		t.Fatalf("Pentagons(5): %v", err)
	}

	pentagon := pents[0]

	children, err := pentagon.Children(6)
	if err != nil {
		t.Fatalf("Children(6): %v", err)
	}

	// Every child shares the pentagon parent; comparing children exercises the
	// same-parent optimization including the deleted-k rejection.
	for _, a := range children {
		for _, b := range children {
			if _, err := a.IsNeighbor(b); err != nil && !errors.Is(err, ErrCellInvalid) {
				t.Fatalf("IsNeighbor(%015x, %015x): unexpected error %v", uint64(a), uint64(b), err)
			}
		}
	}
}

// TestIsNeighborDefensive covers the invalid-input branches of IsNeighbor that a
// validly constructed pair cannot reach: an out-of-range indexing digit under a
// shared parent, a k-axis digit under a shared pentagon parent, and a malformed
// origin that fails the general disk fallback.
func TestIsNeighborDefensive(t *testing.T) {
	t.Parallel()

	base := CellFromString("8928308280fffff")

	// Same res-8 parent, but the origin's finest digit is the unused sentinel.
	invalidChild := base.setIndexDigit(9, invalidDigit)
	sibling := base.setIndexDigit(9, jAxesDigit)

	if _, err := invalidChild.IsNeighbor(sibling); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("IsNeighbor(invalid digit child): got %v, want ErrCellInvalid", err)
	}

	// Same pentagon parent, with a k-axis digit that the deleted subsequence
	// makes invalid.
	pents, err := Pentagons(8)
	if err != nil {
		t.Fatalf("Pentagons(8): %v", err)
	}

	pentParent := pents[0]
	kChild := pentParent.setResolution(9).setIndexDigit(9, kAxesDigit)
	otherChild := pentParent.setResolution(9).setIndexDigit(9, jAxesDigit)

	if _, err := kChild.IsNeighbor(otherChild); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("IsNeighbor(k-axis pentagon child): got %v, want ErrCellInvalid", err)
	}

	// A malformed origin with a different parent than the target skips the
	// optimization and fails in the general disk fallback.
	malformed := base.setIndexDigit(9, invalidDigit)

	far, err := LatLngToCell(LatLng{Lat: -33.8688, Lng: 151.2093}, 9)
	if err != nil {
		t.Fatalf("LatLngToCell(far): %v", err)
	}

	if _, err := malformed.IsNeighbor(far); err == nil {
		t.Fatal("IsNeighbor(malformed, far): got nil error, want failure")
	}
}

// TestGridFreeFunctions checks that the package-level free functions delegate to
// the Cell methods and return matching results.
func TestGridFreeFunctions(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	const k = 2

	disk, err := GridDisk(origin, k)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	method, err := origin.GridDisk(k)
	if err != nil {
		t.Fatalf("Cell.GridDisk: %v", err)
	}

	assertSameSet(t, disk, method, "GridDisk free vs method")

	auto, err := GridDiskDistances(origin, k)
	if err != nil {
		t.Fatalf("GridDiskDistances: %v", err)
	}

	unsafe, err := GridDiskDistancesUnsafe(origin, k)
	if err != nil {
		t.Fatalf("GridDiskDistancesUnsafe: %v", err)
	}

	safe, err := GridDiskDistancesSafe(origin, k)
	if err != nil {
		t.Fatalf("GridDiskDistancesSafe: %v", err)
	}

	for ring := range auto {
		assertSameSet(t, auto[ring], unsafe[ring], "auto vs unsafe ring")
		assertSameSet(t, auto[ring], safe[ring], "auto vs safe ring")
	}
}

// assertSameSet fails if two cell slices differ as sets, ignoring order and
// zero padding.
func assertSameSet(t *testing.T, got, want []Cell, msg string) {
	t.Helper()

	gotSet := cellSet(got)
	wantSet := cellSet(want)

	if len(gotSet) != len(wantSet) {
		t.Fatalf("%s: set sizes differ got=%d want=%d", msg, len(gotSet), len(wantSet))
	}

	for cell := range wantSet {
		if !gotSet[cell] {
			t.Fatalf("%s: missing %015x", msg, uint64(cell))
		}
	}
}
