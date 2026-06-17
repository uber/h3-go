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

// sfSquareLoop is a small square over San Francisco used across region tests.
var sfSquareLoop = GeoLoop{
	{Lat: 37.813, Lng: -122.513},
	{Lat: 37.813, Lng: -122.345},
	{Lat: 37.700, Lng: -122.345},
	{Lat: 37.700, Lng: -122.513},
}

// radLoop converts a loop given in radians (as the C test fixtures are) into a
// degrees GeoLoop.
func radLoop(radVerts [][2]float64) GeoLoop {
	loop := make(GeoLoop, len(radVerts))
	for i, vert := range radVerts {
		loop[i] = LatLng{Lat: vert[0] * RadsToDegs, Lng: vert[1] * RadsToDegs}
	}

	return loop
}

// TestPolygonToCellsReported ports the testPolygonToCellsReported.c regression
// cases: the entire world split into two polygons, and several real-world
// polygons with exact expected cell counts.
func TestPolygonToCellsReported(t *testing.T) {
	t.Parallel()

	t.Run("entire_world", func(t *testing.T) {
		t.Parallel()

		world1 := GeoPolygon{GeoLoop: GeoLoop{
			{Lat: -90, Lng: -180}, {Lat: 90, Lng: -180},
			{Lat: 90, Lng: 0}, {Lat: -90, Lng: 0},
		}}
		world2 := GeoPolygon{GeoLoop: GeoLoop{
			{Lat: -90, Lng: 0}, {Lat: 90, Lng: 0},
			{Lat: 90, Lng: 180}, {Lat: -90, Lng: 180},
		}}

		for res := range 3 {
			cells1, err := PolygonToCells(world1, res)
			if err != nil {
				t.Fatalf("PolygonToCells(world1, %d): %v", res, err)
			}

			cells2, err := PolygonToCells(world2, res)
			if err != nil {
				t.Fatalf("PolygonToCells(world2, %d): %v", res, err)
			}

			if got, want := len(cells1)+len(cells2), NumCells(res); got != want {
				t.Fatalf("res %d: got %d cells, want %d (entire world)", res, got, want)
			}

			seen := make(map[Cell]bool, len(cells1))
			for _, cell := range cells1 {
				seen[cell] = true
			}

			for _, cell := range cells2 {
				if seen[cell] {
					t.Fatalf("res %d: cell %015x found in both halves", res, uint64(cell))
				}
			}
		}
	})

	t.Run("exact_counts", func(t *testing.T) {
		t.Parallel()

		// https://github.com/uber/h3/issues/595: a vertex due east of the center
		// at exactly the same latitude.
		center595, err := Cell(0x85283473fffffff).LatLng()
		if err != nil {
			t.Fatalf("center LatLng: %v", err)
		}

		tests := map[string]struct {
			giveLoop GeoLoop
			giveRes  int
			want     int
		}{
			"h3js_67": {
				giveLoop: GeoLoop{
					{Lat: -33.13755119234615, Lng: -56.25},
					{Lat: -34.30714385628804, Lng: -56.25},
					{Lat: -34.30714385628804, Lng: -57.65625},
					{Lat: -33.13755119234615, Lng: -57.65625},
				},
				giveRes: 7,
				want:    4499,
			},
			"h3js_67_2nd": {
				giveLoop: GeoLoop{
					{Lat: -34.30714385628804, Lng: -57.65625},
					{Lat: -35.4606699514953, Lng: -57.65625},
					{Lat: -35.4606699514953, Lng: -59.0625},
					{Lat: -34.30714385628804, Lng: -59.0625},
				},
				giveRes: 7,
				want:    4609,
			},
			"h3_136": {
				giveLoop: radLoop([][2]float64{
					{0.10068990369902957, 0.8920772174196191},
					{0.10032914690616246, 0.8915914753447348},
					{0.10033349237998787, 0.8915860128746426},
					{0.10069496685903621, 0.8920742194546231},
				}),
				giveRes: 13,
				want:    4353,
			},
			"issue_595": {
				giveLoop: radLoop([][2]float64{
					{center595.Lat * DegsToRads, -2.121207808248113},
					{0.6565301558937859, -2.1281107217935986},
					{0.6515463604919347, -2.1345342663428695},
					{0.6466583305904194, -2.1276313527973842},
				}),
				giveRes: 5,
				want:    8,
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				cells, err := PolygonToCells(GeoPolygon{GeoLoop: tt.giveLoop}, tt.giveRes)
				if err != nil {
					t.Fatalf("PolygonToCells: %v", err)
				}

				if len(cells) != tt.want {
					t.Fatalf("got %d cells, want %d", len(cells), tt.want)
				}
			})
		}
	})
}

// TestPolygonToCellsBasic checks a simple polygon fills with contained cells
// whose centers are all inside, and that the method form agrees.
func TestPolygonToCellsBasic(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: sfSquareLoop}

	cells, err := PolygonToCells(polygon, 7)
	if err != nil {
		t.Fatalf("PolygonToCells: %v", err)
	}

	if len(cells) == 0 {
		t.Fatal("PolygonToCells: got no cells, want some")
	}

	bboxes := bboxesFromGeoPolygon(polygon)

	for _, cell := range cells {
		center, err := cell.LatLng()
		if err != nil {
			t.Fatalf("LatLng: %v", err)
		}

		if !pointInsidePolygon(polygon, bboxes, center) {
			t.Fatalf("cell %015x center not inside polygon", uint64(cell))
		}
	}

	viaMethod, err := polygon.Cells(7)
	if err != nil || len(viaMethod) != len(cells) {
		t.Fatalf("Cells method: got %d (%v), want %d", len(viaMethod), err, len(cells))
	}
}

// TestPolygonToCellsWithHole checks that cells whose centers are inside a hole
// are excluded.
func TestPolygonToCellsWithHole(t *testing.T) {
	t.Parallel()

	hole := GeoLoop{
		{Lat: 37.78, Lng: -122.45},
		{Lat: 37.78, Lng: -122.42},
		{Lat: 37.76, Lng: -122.42},
		{Lat: 37.76, Lng: -122.45},
	}
	polygon := GeoPolygon{GeoLoop: sfSquareLoop, Holes: []GeoLoop{hole}}

	withHole, err := PolygonToCells(polygon, 8)
	if err != nil {
		t.Fatalf("PolygonToCells(with hole): %v", err)
	}

	withoutHole, err := PolygonToCells(GeoPolygon{GeoLoop: sfSquareLoop}, 8)
	if err != nil {
		t.Fatalf("PolygonToCells(no hole): %v", err)
	}

	if len(withHole) >= len(withoutHole) {
		t.Fatalf("hole did not reduce cell count: with=%d without=%d", len(withHole), len(withoutHole))
	}
}

// TestPolygonToCellsEmpty covers the empty-loop short circuit.
func TestPolygonToCellsEmpty(t *testing.T) {
	t.Parallel()

	cells, err := PolygonToCells(GeoPolygon{}, 7)
	if err != nil || cells != nil {
		t.Fatalf("PolygonToCells(empty): got %v (%v), want nil, nil", cells, err)
	}
}

// TestPolygonToCellsResolutionError covers the invalid-resolution path, surfaced
// while tracing the loop edges.
func TestPolygonToCellsResolutionError(t *testing.T) {
	t.Parallel()

	for _, res := range []int{-1, MaxResolution + 1} {
		if _, err := PolygonToCells(GeoPolygon{GeoLoop: sfSquareLoop}, res); err == nil {
			t.Fatalf("PolygonToCells(res %d): got nil error, want failure", res)
		}
	}
}

// TestPolygonToCellsDegenerate covers the degenerate bounding box path, where the
// polygon has zero height (all vertices on one parallel).
func TestPolygonToCellsDegenerate(t *testing.T) {
	t.Parallel()

	flat := GeoPolygon{GeoLoop: GeoLoop{
		{Lat: 1, Lng: -1},
		{Lat: 1, Lng: 0},
		{Lat: 1, Lng: 1},
	}}

	if _, err := PolygonToCells(flat, 7); !errors.Is(err, ErrFailed) {
		t.Fatalf("PolygonToCells(degenerate): got %v, want ErrFailed", err)
	}
}

// TestGetEdgeHexagonsError covers the resolution error path of the loop tracer.
func TestGetEdgeHexagonsError(t *testing.T) {
	t.Parallel()

	search := make(map[Cell]bool)
	if err := getEdgeHexagons(sfSquareLoop, -1, search); err == nil {
		t.Fatal("getEdgeHexagons(res -1): got nil error, want failure")
	}
}

// TestPolygonFloodStepValidCells confirms the flood step expands a valid search
// cell into contained neighbors.
func TestPolygonFloodStepValidCells(t *testing.T) {
	t.Parallel()

	polygon := GeoPolygon{GeoLoop: sfSquareLoop}
	bboxes := bboxesFromGeoPolygon(polygon)

	center := LatLng{Lat: 37.76, Lng: -122.43}

	seed, err := LatLngToCell(center, 8)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	found := map[Cell]bool{}

	next := polygonFloodStep(polygon, bboxes, []Cell{seed}, found)
	if len(found) == 0 || len(next) == 0 {
		t.Fatalf("polygonFloodStep found %d cells, next %d; want some", len(found), len(next))
	}
}

// TestMaxPolygonToCellsSizeVertexFloor covers the branch where the vertex count
// exceeds the bounding-box estimate, so the vertex count is used.
func TestMaxPolygonToCellsSizeVertexFloor(t *testing.T) {
	t.Parallel()

	// A tiny polygon at a coarse resolution estimates very few cells, so the
	// vertex count dominates.
	loop := GeoLoop{
		{Lat: 0.0, Lng: 0.0},
		{Lat: 0.0, Lng: 0.001},
		{Lat: 0.001, Lng: 0.001},
		{Lat: 0.001, Lng: 0.0},
	}

	size, err := maxPolygonToCellsSize(GeoPolygon{GeoLoop: loop}, 0)
	if err != nil {
		t.Fatalf("maxPolygonToCellsSize: %v", err)
	}

	if size < len(loop)+polygonToCellsBuffer {
		t.Fatalf("size %d should be at least vertex count plus buffer", size)
	}
}

// TestBBoxFromGeoLoop covers the empty, normal, and transmeridian cases.
func TestBBoxFromGeoLoop(t *testing.T) {
	t.Parallel()

	if got := bboxFromGeoLoop(GeoLoop{}); got != (bbox{}) {
		t.Fatalf("bboxFromGeoLoop(empty): got %+v, want zero", got)
	}

	normal := bboxFromGeoLoop(sfSquareLoop)
	if normal.north <= normal.south || normal.east <= normal.west {
		t.Fatalf("bboxFromGeoLoop(normal): unexpected %+v", normal)
	}

	if normal.isTransmeridian() {
		t.Fatal("sf square should not be transmeridian")
	}

	trans := bboxFromGeoLoop(GeoLoop{
		{Lat: 10, Lng: 178},
		{Lat: 10, Lng: -178},
		{Lat: -10, Lng: -178},
		{Lat: -10, Lng: 178},
	})
	if !trans.isTransmeridian() {
		t.Fatalf("expected transmeridian bbox, got %+v", trans)
	}
}

// TestBBoxContains covers the standard and transmeridian containment branches.
func TestBBoxContains(t *testing.T) {
	t.Parallel()

	standard := bbox{north: 10, south: -10, east: 10, west: -10}
	if !standard.contains(LatLng{Lat: 0, Lng: 0}) {
		t.Fatal("standard bbox should contain origin")
	}

	if standard.contains(LatLng{Lat: 20, Lng: 0}) {
		t.Fatal("standard bbox should not contain out-of-range latitude")
	}

	if standard.contains(LatLng{Lat: 0, Lng: 20}) {
		t.Fatal("standard bbox should not contain out-of-range longitude")
	}

	trans := bbox{north: 10, south: -10, east: -170, west: 170}
	if !trans.contains(LatLng{Lat: 0, Lng: 179}) || !trans.contains(LatLng{Lat: 0, Lng: -179}) {
		t.Fatal("transmeridian bbox should contain points on both sides")
	}

	if trans.contains(LatLng{Lat: 0, Lng: 0}) {
		t.Fatal("transmeridian bbox should not contain the prime meridian")
	}
}

// TestHexRadiusKm covers both the hexagon and pentagon boundary branches.
func TestHexRadiusKm(t *testing.T) {
	t.Parallel()

	hexagon := CellFromString("8928308280fffff")
	if !hexagon.IsValid() || hexagon.IsPentagon() {
		t.Fatalf("fixture %015x should be a valid hexagon", uint64(hexagon))
	}

	if got := hexagon.hexRadiusKm(); got <= 0 {
		t.Fatalf("hexRadiusKm(hexagon): got %v, want positive", got)
	}

	pentagons, err := Pentagons(9)
	if err != nil {
		t.Fatalf("Pentagons: %v", err)
	}

	if got := pentagons[0].hexRadiusKm(); got <= 0 {
		t.Fatalf("hexRadiusKm(pentagon): got %v, want positive", got)
	}
}

// TestBBoxEstimatesResolutionError covers the resolution error path of both
// estimators.
func TestBBoxEstimatesResolutionError(t *testing.T) {
	t.Parallel()

	box := bboxFromGeoLoop(sfSquareLoop)
	if _, err := bboxHexEstimate(box, -1); err == nil {
		t.Fatal("bboxHexEstimate(res -1): got nil error, want failure")
	}

	if _, err := lineHexEstimate(sfSquareLoop[0], sfSquareLoop[1], -1); err == nil {
		t.Fatal("lineHexEstimate(res -1): got nil error, want failure")
	}
}

// TestBBoxHexEstimateNonFinite covers the non-finite estimate guard, reached when
// a bounding-box corner is NaN so the diagonal and area become NaN.
func TestBBoxHexEstimateNonFinite(t *testing.T) {
	t.Parallel()

	box := bbox{north: math.NaN(), south: 0, east: 1, west: 0}
	if _, err := bboxHexEstimate(box, 5); !errors.Is(err, ErrFailed) {
		t.Fatalf("bboxHexEstimate(NaN): got %v, want ErrFailed", err)
	}
}

// TestLineHexEstimateNonFinite covers the non-finite distance guard, reached when
// an endpoint is NaN so the great-circle distance is NaN.
func TestLineHexEstimateNonFinite(t *testing.T) {
	t.Parallel()

	origin := LatLng{Lat: math.NaN(), Lng: 0}
	destination := LatLng{Lat: 1, Lng: 1}

	if _, err := lineHexEstimate(origin, destination, 5); !errors.Is(err, ErrFailed) {
		t.Fatalf("lineHexEstimate(NaN): got %v, want ErrFailed", err)
	}
}

// TestPointInsideGeoLoopNudges covers the latitude and longitude nudge branches
// of the ray-casting test, reached when the point lies exactly on a vertex
// latitude or a segment-endpoint longitude.
func TestPointInsideGeoLoopNudges(t *testing.T) {
	t.Parallel()

	loop := GeoLoop{
		{Lat: 0, Lng: 0},
		{Lat: 0, Lng: 2},
		{Lat: 2, Lng: 2},
		{Lat: 2, Lng: 0},
	}
	box := bboxFromGeoLoop(loop)

	// Latitude exactly equal to a vertex latitude triggers the lat nudge; the
	// longitude equal to a vertex longitude triggers the lng nudge.
	onVertex := LatLng{Lat: 0, Lng: 0}
	_ = pointInsideGeoLoop(loop, box, onVertex)

	// A point clearly inside still reports as contained after the nudges.
	inside := LatLng{Lat: 1, Lng: 1}
	if !pointInsideGeoLoop(loop, box, inside) {
		t.Fatal("interior point should be contained")
	}
}

// TestPointInsidePolygonHole checks the point-in-polygon test excludes points in
// holes and the normalizeLng helper.
func TestPointInsidePolygonHole(t *testing.T) {
	t.Parallel()

	hole := GeoLoop{
		{Lat: 37.78, Lng: -122.45},
		{Lat: 37.78, Lng: -122.42},
		{Lat: 37.76, Lng: -122.42},
		{Lat: 37.76, Lng: -122.45},
	}
	polygon := GeoPolygon{GeoLoop: sfSquareLoop, Holes: []GeoLoop{hole}}
	bboxes := bboxesFromGeoPolygon(polygon)

	inHole := LatLng{Lat: 37.77, Lng: -122.435}
	if pointInsidePolygon(polygon, bboxes, inHole) {
		t.Fatal("point in hole should not be contained")
	}

	inPolygon := LatLng{Lat: 37.80, Lng: -122.40}
	if !pointInsidePolygon(polygon, bboxes, inPolygon) {
		t.Fatal("point in polygon (outside hole) should be contained")
	}

	outside := LatLng{Lat: 0, Lng: 0}
	if pointInsidePolygon(polygon, bboxes, outside) {
		t.Fatal("point outside should not be contained")
	}
}

// TestNormalizeLng covers the transmeridian normalization branch.
func TestNormalizeLng(t *testing.T) {
	t.Parallel()

	if got := normalizeLng(-1, true); got != -1+twoPiRad {
		t.Fatalf("normalizeLng(-1, true): got %v, want %v", got, -1+twoPiRad)
	}

	if got := normalizeLng(-1, false); got != -1 {
		t.Fatalf("normalizeLng(-1, false): got %v, want -1", got)
	}

	if got := normalizeLng(1, true); got != 1 {
		t.Fatalf("normalizeLng(1, true): got %v, want 1", got)
	}
}

// TestPolygonToCellsTransmeridian ports the testPolygonToCells.c transmeridian
// regressions: a small prime-meridian box, the antimeridian-crossing box, and a
// >4-vertex complex transmeridian polygon, each with an exact expected count.
// The complex case guards the historical bug of using min/max longitude as the
// transmeridian bounds.
func TestPolygonToCellsTransmeridian(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveLoop GeoLoop
		giveRes  int
		want     int
	}{
		"prime_meridian": {
			giveLoop: radLoop([][2]float64{
				{0.01, 0.01}, {0.01, -0.01}, {-0.01, -0.01}, {-0.01, 0.01},
			}),
			giveRes: 7,
			want:    4228,
		},
		"transmeridian": {
			giveLoop: radLoop([][2]float64{
				{0.01, -math.Pi + 0.01}, {0.01, math.Pi - 0.01},
				{-0.01, math.Pi - 0.01}, {-0.01, -math.Pi + 0.01},
			}),
			giveRes: 7,
			want:    4238,
		},
		"complex": {
			giveLoop: radLoop([][2]float64{
				{0.1, -math.Pi + 0.00001}, {0.1, math.Pi - 0.00001},
				{0.05, math.Pi - 0.2}, {-0.1, math.Pi - 0.00001},
				{-0.1, -math.Pi + 0.00001}, {-0.05, -math.Pi + 0.2},
			}),
			giveRes: 4,
			want:    1204,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cells, err := PolygonToCells(GeoPolygon{GeoLoop: tt.giveLoop}, tt.giveRes)
			if err != nil {
				t.Fatalf("PolygonToCells: %v", err)
			}

			if len(cells) != tt.want {
				t.Fatalf("got %d cells, want %d", len(cells), tt.want)
			}
		})
	}
}
