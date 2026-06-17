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
	"slices"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// gridKValues is the set of grid distances exercised by the traversal parity
// tests, spanning the origin-only case through a few full rings.
var gridKValues = []int{0, 1, 2, 3, 5}

// dropZeros returns the non-zero cells of in. The cgo reference leaves zero
// slots in its grid output when crossing pentagons or sizing for the worst
// case; the pure-Go implementation emits only real cells, so parity is compared
// as sets after removing zeros.
func dropZeros(in []h3go.Cell) []h3go.Cell {
	out := make([]h3go.Cell, 0, len(in))
	for _, c := range in {
		if c != 0 {
			out = append(out, c)
		}
	}

	return out
}

// assertSameRings fails if two ringed grid results differ ring-by-ring as sets,
// ignoring order and zero padding.
func assertSameRings(t *testing.T, got [][]h3go.Cell, want [][]h3.Cell, msg string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: ring count got=%d want=%d", msg, len(got), len(want))
	}

	for ring := range got {
		assertSameCellSet(t, dropZeros(got[ring]), dropZeros(toGoCells(want[ring])), msg)
	}
}

// TestGridDiskMatchesCgo asserts the flat disk, ringed distances, and their
// unsafe/safe variants match the cgo reference (as zero-pruned sets) over the
// corpus and a range of k values, including error parity.
func TestGridDiskMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, ref := range referenceCorpus(t) {
		goCell := h3goCell(ref)

		for _, k := range gridKValues {
			wantDisk, wantErr := h3.GridDisk(ref, k)
			gotDisk, gotErr := h3go.GridDisk(goCell, k)

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("GridDisk(%015x, %d) error mismatch: cgo=%v h3go=%v", uint64(ref), k, wantErr, gotErr)
			}

			if wantErr == nil {
				assertSameCellSet(t, dropZeros(gotDisk), dropZeros(toGoCells(wantDisk)), "GridDisk")
			}

			wantSafe, _ := h3.GridDiskDistancesSafe(ref, k)

			gotSafe, gotSafeErr := h3go.GridDiskDistancesSafe(goCell, k)
			if gotSafeErr != nil {
				t.Fatalf("GridDiskDistancesSafe(%015x, %d): %v", uint64(ref), k, gotSafeErr)
			}

			assertSameRings(t, gotSafe, wantSafe, "GridDiskDistancesSafe")

			wantAuto, _ := h3.GridDiskDistances(ref, k)

			gotAuto, gotAutoErr := h3go.GridDiskDistances(goCell, k)
			if gotAutoErr != nil {
				t.Fatalf("GridDiskDistances(%015x, %d): %v", uint64(ref), k, gotAutoErr)
			}

			assertSameRings(t, gotAuto, wantAuto, "GridDiskDistances")

			wantUnsafe, wantUnsafeErr := h3.GridDiskDistancesUnsafe(ref, k)
			gotUnsafe, gotUnsafeErr := h3go.GridDiskDistancesUnsafe(goCell, k)

			if !bothErr(wantUnsafeErr, gotUnsafeErr) {
				t.Fatalf("GridDiskDistancesUnsafe(%015x, %d) error mismatch: cgo=%v h3go=%v", uint64(ref), k, wantUnsafeErr, gotUnsafeErr)
			}

			if wantUnsafeErr == nil {
				assertSameRings(t, gotUnsafe, wantUnsafe, "GridDiskDistancesUnsafe")
			}
		}
	}
}

// TestGridRingMatchesCgo asserts the hollow-ring variants match the cgo
// reference (as zero-pruned sets), including error parity for the unsafe form.
func TestGridRingMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, ref := range referenceCorpus(t) {
		goCell := h3goCell(ref)

		for _, k := range gridKValues {
			wantRing, wantErr := h3.GridRing(ref, k)
			gotRing, gotErr := h3go.GridRing(goCell, k)

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("GridRing(%015x, %d) error mismatch: cgo=%v h3go=%v", uint64(ref), k, wantErr, gotErr)
			}

			if wantErr == nil {
				assertSameCellSet(t, dropZeros(gotRing), dropZeros(toGoCells(wantRing)), "GridRing")
			}

			wantUnsafe, wantUnsafeErr := h3.GridRingUnsafe(ref, k)
			gotUnsafe, gotUnsafeErr := h3go.GridRingUnsafe(goCell, k)

			if !bothErr(wantUnsafeErr, gotUnsafeErr) {
				t.Fatalf("GridRingUnsafe(%015x, %d) error mismatch: cgo=%v h3go=%v", uint64(ref), k, wantUnsafeErr, gotUnsafeErr)
			}

			if wantUnsafeErr == nil {
				assertSameCellSet(t, dropZeros(gotUnsafe), dropZeros(toGoCells(wantUnsafe)), "GridRingUnsafe")
			}
		}
	}
}

// TestGridDisksUnsafeMatchesCgo asserts the batch disk traversal matches the cgo
// reference per origin (as zero-pruned sets), including error parity.
func TestGridDisksUnsafeMatchesCgo(t *testing.T) {
	t.Parallel()

	corpus := referenceCorpus(t)

	for _, k := range gridKValues {
		want, wantErr := h3.GridDisksUnsafe(corpus, k)
		got, gotErr := h3go.GridDisksUnsafe(toGoCells(corpus), k)

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("GridDisksUnsafe(k=%d) error mismatch: cgo=%v h3go=%v", k, wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		if len(got) != len(want) {
			t.Fatalf("GridDisksUnsafe(k=%d): outer len got=%d want=%d", k, len(got), len(want))
		}

		for i := range got {
			assertSameCellSet(t, dropZeros(got[i]), dropZeros(toGoCells(want[i])), "GridDisksUnsafe")
		}
	}
}

// TestIsNeighborMatchesCgo asserts neighbor detection matches the cgo reference
// for bool and error, covering true neighbors (each cell against its ring-1
// cells), non-neighbors, and resolution-mismatched pairs.
func TestIsNeighborMatchesCgo(t *testing.T) {
	t.Parallel()

	corpus := referenceCorpus(t)

	for _, ref := range corpus {
		neighbors, err := h3.GridDisk(ref, 1)
		if err != nil {
			continue
		}

		others := append(slices.Clone(neighbors), corpus...)
		for _, other := range others {
			if other == 0 {
				continue
			}

			wantNeighbor, wantErr := ref.IsNeighbor(other)
			gotNeighbor, gotErr := h3goCell(ref).IsNeighbor(h3goCell(other))

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("IsNeighbor(%015x, %015x) error mismatch: cgo=%v h3go=%v", uint64(ref), uint64(other), wantErr, gotErr)
			}

			if wantErr == nil && wantNeighbor != gotNeighbor {
				t.Fatalf("IsNeighbor(%015x, %015x): cgo=%v h3go=%v", uint64(ref), uint64(other), wantNeighbor, gotNeighbor)
			}
		}
	}
}
