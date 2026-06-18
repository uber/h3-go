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

// TestPolygonToCellsExperimentalWithHole runs a holed polygon through the
// boundary-based modes, exercising the hole branches of the boundary checks.
func TestPolygonToCellsExperimentalWithHole(t *testing.T) {
	t.Parallel()

	hole := GeoLoop{
		{Lat: 37.78, Lng: -122.45},
		{Lat: 37.78, Lng: -122.42},
		{Lat: 37.76, Lng: -122.42},
		{Lat: 37.76, Lng: -122.45},
	}
	holed := GeoPolygon{GeoLoop: sfSquareLoop, Holes: []GeoLoop{hole}}
	solid := GeoPolygon{GeoLoop: sfSquareLoop}

	for _, mode := range []ContainmentMode{ContainmentFull, ContainmentOverlapping} {
		withHole, err := PolygonToCellsExperimental(holed, 9, mode)
		if err != nil {
			t.Fatalf("mode %d holed: %v", mode, err)
		}

		without, err := PolygonToCellsExperimental(solid, 9, mode)
		if err != nil {
			t.Fatalf("mode %d solid: %v", mode, err)
		}

		if len(withHole) >= len(without) {
			t.Fatalf("mode %d: hole did not reduce count: with=%d without=%d", mode, len(withHole), len(without))
		}
	}
}

// TestBBoxScaled covers the latitude clamps and the longitude wrap branches of
// the scaling helper.
func TestBBoxScaled(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		box   bbox
		scale float64
	}{
		"latitude_clamps":  {bbox{north: 89, south: -89, east: 10, west: -10}, 1.4},
		"east_west_wrap":   {bbox{north: 10, south: -10, east: 179, west: -179}, 1.1},
		"west_over_pi":     {bbox{north: 10, south: -10, east: 195, west: 188}, 2},
		"east_under_negpi": {bbox{north: 10, south: -10, east: -185, west: -190}, 2},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tt.box.scaled(tt.scale)
			if got.north > halfPiDeg || got.south < -halfPiDeg {
				t.Fatalf("latitude out of domain: %+v", got)
			}

			if got.east > piDeg || got.east < -piDeg || got.west > piDeg || got.west < -piDeg {
				t.Fatalf("longitude out of domain: %+v", got)
			}
		})
	}
}

// TestBBoxNormalizationEastTrend covers the eastward-default normalization of a
// standard box paired with a far-east transmeridian box.
func TestBBoxNormalizationEastTrend(t *testing.T) {
	t.Parallel()

	standardEast := bbox{north: 5, south: -5, east: 160, west: 150}
	trans := bbox{north: 10, south: -10, east: -170, west: 170}

	firstNorm, secondNorm := normalizationFor(standardEast, trans)
	if firstNorm != normalizeNone {
		t.Fatalf("first normalization: got %d, want normalizeNone", firstNorm)
	}

	if secondNorm != normalizeEast {
		t.Fatalf("second normalization: got %d, want normalizeEast", secondNorm)
	}
}

// transmeridianLoop is a square straddling the antimeridian, used to exercise
// the longitude-normalization branches.
var transmeridianLoop = GeoLoop{
	{Lat: 10, Lng: 178},
	{Lat: 10, Lng: -178},
	{Lat: -10, Lng: -178},
	{Lat: -10, Lng: 178},
}

// TestPolygonToCellsExperimentalModes runs every containment mode and checks the
// output is non-empty and, for center mode, that every cell center is inside.
func TestPolygonToCellsExperimentalModes(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: sfSquareLoop}
	bboxes := polygon.toBboxes()

	modes := []ContainmentMode{
		ContainmentCenter,
		ContainmentFull,
		ContainmentOverlapping,
		ContainmentOverlappingBbox,
	}

	for _, mode := range modes {
		cells, err := PolygonToCellsExperimental(polygon, 8, mode)
		if err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}

		if len(cells) == 0 {
			t.Fatalf("mode %d: got no cells", mode)
		}

		if mode != ContainmentCenter {
			continue
		}

		for _, cell := range cells {
			center, err := cell.LatLng()
			if err != nil {
				t.Fatalf("LatLng: %v", err)
			}

			if !pointInsidePolygon(polygon, bboxes, center) {
				t.Fatalf("center mode: cell %015x center not inside polygon", uint64(cell))
			}
		}
	}
}

// TestPolygonToCellsExperimentalTransmeridian fills a polygon crossing the
// antimeridian, exercising the longitude-normalization paths in the algorithm.
func TestPolygonToCellsExperimentalTransmeridian(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: transmeridianLoop}

	cells, err := PolygonToCellsExperimental(polygon, 4, ContainmentOverlappingBbox)
	if err != nil {
		t.Fatalf("PolygonToCellsExperimental: %v", err)
	}

	if len(cells) == 0 {
		t.Fatal("got no cells for transmeridian polygon")
	}
}

// TestPolygonToCellsExperimentalEmpty covers the empty-loop short circuit.
func TestPolygonToCellsExperimentalEmpty(t *testing.T) {
	t.Parallel()

	cells, err := PolygonToCellsExperimental(GeoPolygon{}, 7, ContainmentCenter)
	if err != nil || cells != nil {
		t.Fatalf("empty: got %v (%v), want nil, nil", cells, err)
	}
}

// TestPolygonToCellsExperimentalErrors covers the resolution and mode validation
// paths.
func TestPolygonToCellsExperimentalErrors(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: sfSquareLoop}

	for _, res := range []int{-1, MaxResolution + 1} {
		if _, err := PolygonToCellsExperimental(polygon, res, ContainmentCenter); !errors.Is(err, ErrResolutionDomain) {
			t.Fatalf("res %d: got %v, want ErrResolutionDomain", res, err)
		}
	}

	for _, mode := range []ContainmentMode{ContainmentInvalid, ContainmentMode(16)} {
		if _, err := PolygonToCellsExperimental(polygon, 6, mode); !errors.Is(err, ErrOptionInvalid) {
			t.Fatalf("mode %d: got %v, want ErrOptionInvalid", mode, err)
		}
	}
}

// TestPolygonToCellsExperimentalBounds covers the cell-cap path, where exceeding
// the maximum returns ErrMemoryBounds.
func TestPolygonToCellsExperimentalBounds(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: sfSquareLoop}
	if _, err := PolygonToCellsExperimental(polygon, 8, ContainmentCenter, 1); !errors.Is(err, ErrMemoryBounds) {
		t.Fatalf("bounds: got %v, want ErrMemoryBounds", err)
	}
}

// TestPolygonCompactCellsEarlyStop stops iteration on the first coarse cell and
// on the first target cell, covering both early-return paths of the iterator.
func TestPolygonCompactCellsEarlyStop(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: sfSquareLoop}
	bboxes := polygon.toBboxes()

	res := 9

	sawCoarse := false

	for cell := range polygonCompactCells(polygon, bboxes, res, ContainmentFull) {
		if cell.Resolution() < res {
			sawCoarse = true

			break
		}
	}

	if !sawCoarse {
		t.Fatal("expected at least one coarse compact cell")
	}

	sawTarget := false

	for cell := range polygonCompactCells(polygon, bboxes, res, ContainmentFull) {
		if cell.Resolution() == res {
			sawTarget = true

			break
		}
	}

	if !sawTarget {
		t.Fatal("expected at least one target-resolution compact cell")
	}
}

// TestCellToBBoxPoles covers the pole-cell branches, where the bounding box is
// expanded to a full circle around the pole.
func TestCellToBBoxPoles(t *testing.T) {
	t.Parallel()

	for res := 0; res <= 3; res++ {
		north := cellToBBox(northPoleCells[res], false)
		if north.north != halfPiDeg || north.east != piDeg || north.west != -piDeg {
			t.Fatalf("north pole res %d: got %+v", res, north)
		}

		south := cellToBBox(southPoleCells[res], true)
		if south.south != -halfPiDeg || south.east != piDeg || south.west != -piDeg {
			t.Fatalf("south pole res %d: got %+v", res, south)
		}
	}
}

// TestNextCellPentagonSkip covers the missing-pentagon-child skip in nextCell.
func TestNextCellPentagonSkip(t *testing.T) {
	t.Parallel()

	// Base cell 4 is a pentagon; its center child has digit 0 at resolution 1.
	centerChild := setH3Index(1, 4, centerDigit)

	next := nextCell(centerChild)
	if got := indexDigit(next, 1); got != 2 {
		t.Fatalf("nextCell pentagon center child: digit %d, want 2 (skipped the deleted 1)", got)
	}
}

// TestBaseCellNumToCellRange covers the in-range and out-of-range cases.
func TestBaseCellNumToCellRange(t *testing.T) {
	t.Parallel()

	if got := baseCellNumToCell(0); got.BaseCellNumber() != 0 || got.Resolution() != 0 {
		t.Fatalf("baseCellNumToCell(0): got %015x", uint64(got))
	}

	for _, num := range []int{-1, NumBaseCells} {
		if got := baseCellNumToCell(num); got != 0 {
			t.Fatalf("baseCellNumToCell(%d): got %015x, want 0", num, uint64(got))
		}
	}
}

// TestValidateContainmentMode covers the valid, out-of-range, and extra-bit
// cases.
func TestValidateContainmentMode(t *testing.T) {
	t.Parallel()

	for _, mode := range []ContainmentMode{ContainmentCenter, ContainmentOverlappingBbox} {
		if err := validateContainmentMode(mode); err != nil {
			t.Fatalf("validateContainmentMode(%d): %v", mode, err)
		}
	}

	for _, mode := range []ContainmentMode{ContainmentInvalid, ContainmentMode(16)} {
		if err := validateContainmentMode(mode); !errors.Is(err, ErrOptionInvalid) {
			t.Fatalf("validateContainmentMode(%d): got %v, want ErrOptionInvalid", mode, err)
		}
	}
}

// TestBBoxNormalization covers the longitude-normalization helper across the
// standard and transmeridian box-pair combinations.
func TestBBoxNormalization(t *testing.T) {
	t.Parallel()

	standardA := bbox{north: 10, south: -10, east: 10, west: -10}
	standardB := bbox{north: 5, south: -5, east: 20, west: 5}
	transEast := bbox{north: 10, south: -10, east: -170, west: 170}
	transOther := bbox{north: 8, south: -8, east: -160, west: 165}

	tests := map[string]struct {
		first, second  bbox
		wantFirst      longitudeNormalization
		wantSecondNone bool
	}{
		"both_standard": {standardA, standardB, normalizeNone, true},
		"first_trans":   {transEast, standardB, normalizeEast, true},
		"both_trans":    {transEast, transOther, normalizeEast, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			firstNorm, secondNorm := normalizationFor(tt.first, tt.second)
			if firstNorm != tt.wantFirst {
				t.Fatalf("first normalization: got %d, want %d", firstNorm, tt.wantFirst)
			}

			if (secondNorm == normalizeNone) != tt.wantSecondNone {
				t.Fatalf("second normalization: got %d", secondNorm)
			}
		})
	}
}

// TestBBoxNormalizationWestTrend covers the westward-trending normalization of a
// transmeridian box paired with a far-west standard box.
func TestBBoxNormalizationWestTrend(t *testing.T) {
	t.Parallel()

	trans := bbox{north: 10, south: -10, east: -170, west: 170}
	standardWest := bbox{north: 5, south: -5, east: -150, west: -160}

	firstNorm, secondNorm := normalizationFor(trans, standardWest)
	if firstNorm != normalizeWest {
		t.Fatalf("first normalization: got %d, want normalizeWest", firstNorm)
	}

	// Standard second box needs no normalization.
	if secondNorm != normalizeNone {
		t.Fatalf("second normalization: got %d, want normalizeNone", secondNorm)
	}

	// Pairing a standard first box with a transmeridian second exercises the
	// second-box branches.
	firstNorm, secondNorm = normalizationFor(standardWest, trans)
	if firstNorm != normalizeNone {
		t.Fatalf("first normalization (swapped): got %d, want normalizeNone", firstNorm)
	}

	if secondNorm == normalizeNone {
		t.Fatal("second normalization (swapped): got normalizeNone, want a shift")
	}
}

// TestApplyNormalization covers each normalization case and its guard.
func TestApplyNormalization(t *testing.T) {
	t.Parallel()

	if got := applyNormalization(-170, normalizeEast); got != 190 {
		t.Fatalf("east of -170: got %v, want 190", got)
	}

	if got := applyNormalization(10, normalizeEast); got != 10 {
		t.Fatalf("east of 10: got %v, want 10", got)
	}

	if got := applyNormalization(170, normalizeWest); got != -190 {
		t.Fatalf("west of 170: got %v, want -190", got)
	}

	if got := applyNormalization(-10, normalizeWest); got != -10 {
		t.Fatalf("west of -10: got %v, want -10", got)
	}

	if got := applyNormalization(42, normalizeNone); got != 42 {
		t.Fatalf("none: got %v, want 42", got)
	}
}

// TestBBoxOverlapAndContains covers the overlap and containment predicates,
// including transmeridian normalization and the early-reject branches.
func TestBBoxOverlapAndContains(t *testing.T) {
	t.Parallel()

	outer := bbox{north: 20, south: -20, east: 20, west: -20}
	inner := bbox{north: 10, south: -10, east: 10, west: -10}
	disjointLat := bbox{north: 40, south: 30, east: 10, west: -10}
	disjointLng := bbox{north: 10, south: -10, east: 40, west: 30}

	if !outer.overlaps(inner) {
		t.Fatal("outer should overlap inner")
	}

	if outer.overlaps(disjointLat) {
		t.Fatal("latitude-disjoint boxes should not overlap")
	}

	if outer.overlaps(disjointLng) {
		t.Fatal("longitude-disjoint boxes should not overlap")
	}

	if !outer.containsBBox(inner) {
		t.Fatal("outer should contain inner")
	}

	if inner.containsBBox(outer) {
		t.Fatal("inner should not contain outer")
	}

	if outer.containsBBox(disjointLat) {
		t.Fatal("outer should not contain a latitude-disjoint box")
	}
}

// TestLineCrossesLine covers the crossing, parallel, and out-of-range cases.
func TestLineCrossesLine(t *testing.T) {
	t.Parallel()

	a1 := LatLng{Lat: 0, Lng: 0}
	a2 := LatLng{Lat: 0, Lng: 10}
	b1 := LatLng{Lat: -5, Lng: 5}
	b2 := LatLng{Lat: 5, Lng: 5}

	if !lineCrossesLine(a1, a2, b1, b2) {
		t.Fatal("crossing segments should report true")
	}

	// Parallel segments never intersect (zero denominator).
	if lineCrossesLine(a1, a2, LatLng{Lat: 1, Lng: 0}, LatLng{Lat: 1, Lng: 10}) {
		t.Fatal("parallel segments should report false")
	}

	// Segment b is to the side, so the intersection parameter is out of range.
	if lineCrossesLine(a1, a2, LatLng{Lat: -5, Lng: 50}, LatLng{Lat: 5, Lng: 50}) {
		t.Fatal("non-overlapping segments should report false")
	}
}

// TestPolygonToCellsExperimentalSFCounts ports the
// testPolygonToCellsExperimental.c regression: exact res-9 cell counts for the
// San Francisco polygon across all four containment modes. The counts encode the
// ordering Full < Center < Overlapping < OverlappingBbox.
func TestPolygonToCellsExperimentalSFCounts(t *testing.T) {
	t.Parallel()

	sf := GeoPolygon{GeoLoop: radLoop([][2]float64{
		{0.659966917655, -2.1364398519396}, {0.6595011102219, -2.1359434279405},
		{0.6583348114025, -2.1354884206045}, {0.6581220034068, -2.1382437718946},
		{0.6594479998527, -2.1384597563896}, {0.6599990002976, -2.1376771158464},
	})}

	tests := map[string]struct {
		giveMode ContainmentMode
		want     int
	}{
		"center":           {giveMode: ContainmentCenter, want: 1253},
		"full":             {giveMode: ContainmentFull, want: 1175},
		"overlapping":      {giveMode: ContainmentOverlapping, want: 1334},
		"overlapping_bbox": {giveMode: ContainmentOverlappingBbox, want: 1416},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cells, err := PolygonToCellsExperimental(sf, 9, tt.giveMode)
			if err != nil {
				t.Fatalf("PolygonToCellsExperimental: %v", err)
			}

			if len(cells) != tt.want {
				t.Fatalf("got %d cells, want %d", len(cells), tt.want)
			}
		})
	}
}

// TestCellToBBoxContainsGeometry ports the testCellToBBoxExhaustive.c
// correctness properties: a cell's bounding box contains all of its own boundary
// vertices, and a parent's child-covering bounding box contains every boundary
// vertex of its descendants.
func TestCellToBBoxContainsGeometry(t *testing.T) {
	t.Parallel()

	res0, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	containsAll := func(t *testing.T, box bbox, cell Cell) {
		t.Helper()

		boundary, err := cell.Boundary()
		if err != nil {
			t.Fatalf("Boundary(%015x): %v", uint64(cell), err)
		}

		for _, vertex := range boundary {
			if !box.contains(vertex) {
				t.Fatalf("bbox does not contain vertex %v of cell %015x", vertex, uint64(cell))
			}
		}
	}

	t.Run("cell_bbox_bounds_self", func(t *testing.T) {
		t.Parallel()

		for _, parent := range res0 {
			for res := 0; res <= 2; res++ {
				cells, err := parent.Children(res)
				if err != nil {
					t.Fatalf("Children(%d): %v", res, err)
				}

				for _, cell := range cells {
					containsAll(t, cellToBBox(cell, false), cell)
				}
			}
		}
	})

	t.Run("parent_bbox_bounds_children", func(t *testing.T) {
		t.Parallel()

		for _, parent := range res0 {
			box := cellToBBox(parent, true)

			children, err := parent.Children(2)
			if err != nil {
				t.Fatalf("Children(2): %v", err)
			}

			for _, child := range children {
				containsAll(t, box, child)
			}
		}
	})
}
