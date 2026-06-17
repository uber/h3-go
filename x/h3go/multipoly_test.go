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

// TestUnionFind covers the union-find rank-swap and already-merged branches.
func TestUnionFind(t *testing.T) {
	t.Parallel()

	first := &arc{rank: 1}
	first.parent = first
	second := &arc{rank: 1}
	second.parent = second
	third := &arc{rank: 1}
	third.parent = third

	// Equal ranks: no swap; first absorbs second.
	union(first, second)

	// Lower-rank first absorbs into higher-rank component: triggers the swap.
	union(third, first)

	// Already in the same component: the merge body is skipped.
	union(second, third)

	if first.root() != second.root() || second.root() != third.root() {
		t.Fatal("all arcs should share one root after the unions")
	}
}

// TestCellsToMultiPolygonSingle covers a single hexagon: one polygon, one outer
// loop, no holes.
func TestCellsToMultiPolygonSingle(t *testing.T) {
	t.Parallel()

	origin, err := LatLngToCell(LatLng{Lat: 37.78, Lng: -122.42}, 7)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	polys, err := CellsToMultiPolygon([]Cell{origin})
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polys) != 1 || len(polys[0].Holes) != 0 {
		t.Fatalf("single cell: got %d polygons, %d holes", len(polys), len(polys[0].Holes))
	}

	if len(polys[0].GeoLoop) != 6 {
		t.Fatalf("single hexagon outline: got %d verts, want 6", len(polys[0].GeoLoop))
	}
}

// TestCellsToMultiPolygonDisk covers a contiguous blob, exercising edge
// cancellation and union-find merging into a single outer loop.
func TestCellsToMultiPolygonDisk(t *testing.T) {
	t.Parallel()

	origin, err := LatLngToCell(LatLng{Lat: 0, Lng: 0}, 6)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	disk, err := origin.GridDisk(2)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	polys, err := CellsToMultiPolygon(disk)
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polys) != 1 || len(polys[0].Holes) != 0 {
		t.Fatalf("disk: got %d polygons, %d holes", len(polys), len(polys[0].Holes))
	}
}

// TestCellsToMultiPolygonRing covers a hollow ring, which yields one polygon with
// a single hole.
func TestCellsToMultiPolygonRing(t *testing.T) {
	t.Parallel()

	origin, err := LatLngToCell(LatLng{Lat: 10, Lng: 10}, 6)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	ring, err := origin.GridRing(2)
	if err != nil {
		t.Fatalf("GridRing: %v", err)
	}

	polys, err := CellsToMultiPolygon(ring)
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polys) != 1 || len(polys[0].Holes) != 1 {
		t.Fatalf("ring: got %d polygons, %d holes (want 1, 1)", len(polys), len(polys[0].Holes))
	}
}

// TestCellsToMultiPolygonDisjoint covers two far-apart blobs, which yield two
// separate polygons.
func TestCellsToMultiPolygonDisjoint(t *testing.T) {
	t.Parallel()

	first, err := LatLngToCell(LatLng{Lat: 0, Lng: 0}, 6)
	if err != nil {
		t.Fatalf("LatLngToCell first: %v", err)
	}

	second, err := LatLngToCell(LatLng{Lat: -30, Lng: 60}, 6)
	if err != nil {
		t.Fatalf("LatLngToCell second: %v", err)
	}

	firstDisk, err := first.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk first: %v", err)
	}

	secondDisk, err := second.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk second: %v", err)
	}

	polys, err := CellsToMultiPolygon(append(firstDisk, secondDisk...))
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polys) != 2 {
		t.Fatalf("disjoint: got %d polygons, want 2", len(polys))
	}
}

// TestCellsToMultiPolygonPentagon covers a pentagon-centered set, exercising the
// five-edge arc ordering.
func TestCellsToMultiPolygonPentagon(t *testing.T) {
	t.Parallel()

	pentagon := setH3Index(5, 4, centerDigit)
	if !pentagon.IsPentagon() {
		t.Fatalf("fixture %015x is not a pentagon", uint64(pentagon))
	}

	disk, err := pentagon.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	polys, err := CellsToMultiPolygon(disk)
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polys) != 1 {
		t.Fatalf("pentagon disk: got %d polygons, want 1", len(polys))
	}
}

// TestCellsToMultiPolygonGlobe covers the whole-globe case: all base cells leave
// no outline, producing the eight-triangle globe representation.
func TestCellsToMultiPolygonGlobe(t *testing.T) {
	t.Parallel()

	cells, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	polys, err := CellsToMultiPolygon(cells)
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polys) != 8 {
		t.Fatalf("globe: got %d polygons, want 8", len(polys))
	}

	for _, poly := range polys {
		if len(poly.GeoLoop) != 3 {
			t.Fatalf("globe triangle: got %d verts, want 3", len(poly.GeoLoop))
		}
	}
}

// TestCellsToMultiPolygonEmpty covers the empty-input short circuit.
func TestCellsToMultiPolygonEmpty(t *testing.T) {
	t.Parallel()

	polys, err := CellsToMultiPolygon(nil)
	if err != nil || polys != nil {
		t.Fatalf("empty: got %v (%v), want nil, nil", polys, err)
	}
}

// TestCellsToMultiPolygonErrors covers the validation paths: invalid cell,
// mismatched resolution, and duplicate input.
func TestCellsToMultiPolygonErrors(t *testing.T) {
	t.Parallel()

	origin, err := LatLngToCell(LatLng{Lat: 0, Lng: 0}, 7)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	parent, err := origin.Parent(6)
	if err != nil {
		t.Fatalf("Parent: %v", err)
	}

	tests := map[string]struct {
		cells   []Cell
		wantErr error
	}{
		"invalid":   {[]Cell{0}, ErrCellInvalid},
		"mixed_res": {[]Cell{origin, parent}, ErrResolutionMismatch},
		"duplicate": {[]Cell{origin, origin}, ErrDuplicateInput},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := CellsToMultiPolygon(tt.cells); !errors.Is(err, tt.wantErr) {
				t.Fatalf("%s: got %v, want %v", name, err, tt.wantErr)
			}
		})
	}
}

// TestCellsToMultiPolygonReported ports the testCellsToLinkedMultiPolygon.c
// cases: exact polygon, loop, and coordinate counts for single cells, contiguous
// and non-contiguous sets, a ring with a hole, and a pentagon.
func TestCellsToMultiPolygonReported(t *testing.T) {
	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		if _, err := CellsToMultiPolygon([]Cell{0xfffffffffffffff}); !errors.Is(err, ErrCellInvalid) {
			t.Fatalf("got %v, want ErrCellInvalid", err)
		}
	})

	tests := map[string]struct {
		giveCells    []Cell
		wantPolygons int
		wantOuter    int // coords on the first polygon's outer loop
		wantHole     int // coords on the first polygon's single hole, 0 if none
	}{
		"single_hex": {
			giveCells:    []Cell{0x890dab6220bffff},
			wantPolygons: 1,
			wantOuter:    6,
		},
		"contiguous2": {
			giveCells:    []Cell{0x8928308291bffff, 0x89283082957ffff},
			wantPolygons: 1,
			wantOuter:    10,
		},
		"non_contiguous2": {
			giveCells:    []Cell{0x8928308291bffff, 0x89283082943ffff},
			wantPolygons: 2,
			wantOuter:    6,
		},
		"contiguous3": {
			giveCells:    []Cell{0x8928308288bffff, 0x892830828d7ffff, 0x8928308289bffff},
			wantPolygons: 1,
			wantOuter:    12,
		},
		"hole": {
			giveCells: []Cell{
				0x892830828c7ffff, 0x892830828d7ffff, 0x8928308289bffff,
				0x89283082813ffff, 0x8928308288fffff, 0x89283082883ffff,
			},
			wantPolygons: 1,
			wantOuter:    18,
			wantHole:     6,
		},
		"pentagon": {
			giveCells:    []Cell{0x851c0003fffffff},
			wantPolygons: 1,
			wantOuter:    10,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			polygons, err := CellsToMultiPolygon(tt.giveCells)
			if err != nil {
				t.Fatalf("CellsToMultiPolygon: %v", err)
			}

			if len(polygons) != tt.wantPolygons {
				t.Fatalf("polygons: got %d, want %d", len(polygons), tt.wantPolygons)
			}

			if got := len(polygons[0].GeoLoop); got != tt.wantOuter {
				t.Fatalf("outer coords: got %d, want %d", got, tt.wantOuter)
			}

			if tt.wantHole == 0 {
				if len(polygons[0].Holes) != 0 {
					t.Fatalf("holes: got %d, want 0", len(polygons[0].Holes))
				}

				return
			}

			if len(polygons[0].Holes) != 1 {
				t.Fatalf("holes: got %d, want 1", len(polygons[0].Holes))
			}

			if got := len(polygons[0].Holes[0]); got != tt.wantHole {
				t.Fatalf("hole coords: got %d, want %d", got, tt.wantHole)
			}
		})
	}
}

// TestCellsToMultiPolygonIssue1049 ports the testCellsToMultiPoly.c issue_1049
// regression: a 168-cell res-2 set must assemble into exactly 12 polygons, each
// with no holes.
func TestCellsToMultiPolygonIssue1049(t *testing.T) {
	t.Parallel()

	cells := []Cell{
		0x827487fffffffff, 0x82748ffffffffff, 0x827497fffffffff, 0x82749ffffffffff,
		0x8274affffffffff, 0x8274c7fffffffff, 0x8274cffffffffff, 0x8274d7fffffffff,
		0x8274e7fffffffff, 0x8274effffffffff, 0x8274f7fffffffff, 0x82754ffffffffff,
		0x827c07fffffffff, 0x827c27fffffffff, 0x827c2ffffffffff, 0x827c37fffffffff,
		0x827d87fffffffff, 0x827d8ffffffffff, 0x827d97fffffffff, 0x827d9ffffffffff,
		0x827da7fffffffff, 0x827daffffffffff, 0x82801ffffffffff, 0x8280a7fffffffff,
		0x8280affffffffff, 0x8280b7fffffffff, 0x828197fffffffff, 0x82819ffffffffff,
		0x8281a7fffffffff, 0x8281b7fffffffff, 0x828207fffffffff, 0x82820ffffffffff,
		0x828227fffffffff, 0x82822ffffffffff, 0x8282e7fffffffff, 0x828307fffffffff,
		0x82830ffffffffff, 0x82831ffffffffff, 0x82832ffffffffff, 0x828347fffffffff,
		0x82834ffffffffff, 0x828357fffffffff, 0x82835ffffffffff, 0x828367fffffffff,
		0x828377fffffffff, 0x82a447fffffffff, 0x82a457fffffffff, 0x82a45ffffffffff,
		0x82a467fffffffff, 0x82a46ffffffffff, 0x82a477fffffffff, 0x82a4c7fffffffff,
		0x82a4cffffffffff, 0x82a4d7fffffffff, 0x82a4e7fffffffff, 0x82a4effffffffff,
		0x82a4f7fffffffff, 0x82a547fffffffff, 0x82a54ffffffffff, 0x82a557fffffffff,
		0x82a55ffffffffff, 0x82a567fffffffff, 0x82a577fffffffff, 0x82a837fffffffff,
		0x82a897fffffffff, 0x82a8a7fffffffff, 0x82a8b7fffffffff, 0x82a917fffffffff,
		0x82a927fffffffff, 0x82a937fffffffff, 0x82a987fffffffff, 0x82a98ffffffffff,
		0x82a997fffffffff, 0x82a99ffffffffff, 0x82a9a7fffffffff, 0x82a9affffffffff,
		0x82ac47fffffffff, 0x82ac57fffffffff, 0x82ac5ffffffffff, 0x82ac67fffffffff,
		0x82ac6ffffffffff, 0x82ac77fffffffff, 0x82ad47fffffffff, 0x82ad4ffffffffff,
		0x82ad57fffffffff, 0x82ad5ffffffffff, 0x82ad67fffffffff, 0x82ad77fffffffff,
		0x82c207fffffffff, 0x82c217fffffffff, 0x82c227fffffffff, 0x82c237fffffffff,
		0x82c287fffffffff, 0x82c28ffffffffff, 0x82c29ffffffffff, 0x82c2a7fffffffff,
		0x82c2affffffffff, 0x82c2b7fffffffff, 0x82c307fffffffff, 0x82c317fffffffff,
		0x82c31ffffffffff, 0x82c337fffffffff, 0x82cfb7fffffffff, 0x82d0c7fffffffff,
		0x82d0d7fffffffff, 0x82d0dffffffffff, 0x82d0e7fffffffff, 0x82d0f7fffffffff,
		0x82d147fffffffff, 0x82d157fffffffff, 0x82d15ffffffffff, 0x82d167fffffffff,
		0x82d177fffffffff, 0x82d187fffffffff, 0x82d18ffffffffff, 0x82d197fffffffff,
		0x82d19ffffffffff, 0x82d1a7fffffffff, 0x82d1affffffffff, 0x82dc47fffffffff,
		0x82dc57fffffffff, 0x82dc5ffffffffff, 0x82dc67fffffffff, 0x82dc6ffffffffff,
		0x82dc77fffffffff, 0x82dcc7fffffffff, 0x82dccffffffffff, 0x82dcd7fffffffff,
		0x82dce7fffffffff, 0x82dceffffffffff, 0x82dcf7fffffffff, 0x82dd1ffffffffff,
		0x82dd47fffffffff, 0x82dd4ffffffffff, 0x82dd57fffffffff, 0x82dd5ffffffffff,
		0x82dd6ffffffffff, 0x82dd87fffffffff, 0x82dd8ffffffffff, 0x82dd97fffffffff,
		0x82dd9ffffffffff, 0x82ddaffffffffff, 0x82ddb7fffffffff, 0x82dec7fffffffff,
		0x82decffffffffff, 0x82ded7fffffffff, 0x82dee7fffffffff, 0x82deeffffffffff,
		0x82def7fffffffff, 0x82df0ffffffffff, 0x82df1ffffffffff, 0x82df47fffffffff,
		0x82df4ffffffffff, 0x82df57fffffffff, 0x82df5ffffffffff, 0x82df77fffffffff,
		0x82df8ffffffffff, 0x82df9ffffffffff, 0x82e6c7fffffffff, 0x82e6cffffffffff,
		0x82e6d7fffffffff, 0x82e6dffffffffff, 0x82e6effffffffff, 0x82e6f7fffffffff,
	}

	polygons, err := CellsToMultiPolygon(cells)
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polygons) != 12 {
		t.Fatalf("polygons: got %d, want 12", len(polygons))
	}

	for i := range polygons {
		if len(polygons[i].Holes) != 0 {
			t.Fatalf("polygon %d: got %d holes, want 0", i, len(polygons[i].Holes))
		}
	}
}

// TestCellsToMultiPolygonEquator ports the testCellsToMultiPoly.c equator_cells
// regression: a global band of cells assembles into a single polygon with one
// hole.
func TestCellsToMultiPolygonEquator(t *testing.T) {
	t.Parallel()

	cells := []Cell{
		0x81807ffffffffff, 0x817efffffffffff, 0x81723ffffffffff, 0x817ebffffffffff,
		0x817c3ffffffffff, 0x817e3ffffffffff, 0x817a3ffffffffff, 0x8166fffffffffff,
		0x8172bffffffffff, 0x816afffffffffff, 0x81933ffffffffff, 0x8168fffffffffff,
		0x8188fffffffffff, 0x81853ffffffffff, 0x817f7ffffffffff, 0x8180bffffffffff,
		0x81783ffffffffff, 0x81743ffffffffff, 0x8170bffffffffff, 0x8173bffffffffff,
		0x8179bffffffffff, 0x817cbffffffffff, 0x8188bffffffffff, 0x81857ffffffffff,
		0x816f7ffffffffff, 0x8177bffffffffff, 0x81617ffffffffff, 0x816f3ffffffffff,
		0x8174bffffffffff, 0x8180fffffffffff, 0x817a7ffffffffff, 0x81767ffffffffff,
		0x81757ffffffffff, 0x81957ffffffffff, 0x81787ffffffffff, 0x81847ffffffffff,
		0x81653ffffffffff, 0x817bbffffffffff, 0x816cfffffffffff, 0x816abffffffffff,
		0x815f3ffffffffff, 0x817c7ffffffffff, 0x8168bffffffffff, 0x818cbffffffffff,
		0x818cfffffffffff, 0x818afffffffffff, 0x8174fffffffffff, 0x8172fffffffffff,
		0x8170fffffffffff, 0x816fbffffffffff, 0x81657ffffffffff, 0x816c7ffffffffff,
		0x8186bffffffffff, 0x81763ffffffffff, 0x818a7ffffffffff, 0x8186fffffffffff,
		0x81707ffffffffff, 0x8182bffffffffff, 0x818f3ffffffffff, 0x8182fffffffffff,
	}

	polygons, err := CellsToMultiPolygon(cells)
	if err != nil {
		t.Fatalf("CellsToMultiPolygon: %v", err)
	}

	if len(polygons) != 1 {
		t.Fatalf("polygons: got %d, want 1", len(polygons))
	}

	if len(polygons[0].Holes) != 1 {
		t.Fatalf("holes: got %d, want 1", len(polygons[0].Holes))
	}
}
