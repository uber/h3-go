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

package h3

import (
	"errors"
	"math"
	"slices"
	"strconv"
	"testing"
)

// Fixtures mirror the reference library's geodesic tests, given in radians.
var (
	// geodesicWestUS spans roughly 1000 km by 1000 km over the western United
	// States.
	geodesicWestUS = GeoPolygon{GeoLoop: radLoop([][2]float64{
		{0.698132, -2.094395}, {0.698132, -1.919862}, {0.610865, -1.919862}, {0.610865, -2.094395},
		{0.628319, -2.181662}, {0.663225, -2.181662}, {0.680678, -2.138763}, {0.680678, -2.050796},
	})}

	// geodesicNevadaHole is a hole roughly 300 km by 200 km inside geodesicWestUS.
	geodesicNevadaHole = radLoop([][2]float64{
		{0.663225, -2.007129}, {0.645772, -2.007129}, {0.645772, -1.989676}, {0.663225, -1.989676},
		{0.680678, -1.998403},
	})

	// geodesicPacific spans roughly 6000 km by 4000 km across the antimeridian.
	geodesicPacific = GeoPolygon{GeoLoop: radLoop([][2]float64{
		{0.698132, -2.967060}, {0.698132, 2.967060}, {0.174533, 2.967060}, {0.174533, -2.967060},
		{0.436332, -3.141593}, {0.610865, -3.141593},
	})}

	// geodesicDegenerate has three vertices within 1e-12 radians of each other.
	geodesicDegenerate = GeoPolygon{GeoLoop: radLoop([][2]float64{
		{0.659966917655, -2.1364398519394}, {0.659966917656, -2.1364398519395}, {0.659966917657, -2.1364398519396},
	})}

	// geodesicPoint and geodesicLine have too few vertices for a loop.
	geodesicPoint = radLoop([][2]float64{{0.6595072188743, -2.1371053983433}})
	geodesicLine  = radLoop([][2]float64{{0.6595072188743, -2.1371053983433}, {0.6591482046471, -2.1373141048153}})
)

// degLoop builds a loop from latitude and longitude pairs in degrees.
func degLoop(verts [][2]float64) GeoLoop {
	loop := make(GeoLoop, len(verts))
	for i, vert := range verts {
		loop[i] = LatLng{Lat: vert[0], Lng: vert[1]}
	}

	return loop
}

// geodesicFill runs the geodesic fill and returns the sorted cells.
func geodesicFill(t *testing.T, polygon GeoPolygon, res int, mode ContainmentMode) []Cell {
	t.Helper()

	cells, err := PolygonToCellsExperimental(polygon, res, mode|ContainmentGeodesic)
	if err != nil {
		t.Fatalf("PolygonToCellsExperimental(geodesic %d): %v", mode, err)
	}

	slices.Sort(cells)

	return cells
}

// TestGeodesicCounts checks the geodesic fill against the reference library's
// cell counts, and that center mode sits between full and overlapping.
func TestGeodesicCounts(t *testing.T) {
	t.Parallel()

	holed := GeoPolygon{GeoLoop: geodesicWestUS.GeoLoop, Holes: []GeoLoop{geodesicNevadaHole}}

	tests := map[string]struct {
		givePolygon     GeoPolygon
		giveRes         int
		wantFull        int
		wantOverlapping int
	}{
		"west_us": {givePolygon: geodesicWestUS, giveRes: 5, wantFull: 2198, wantOverlapping: 2469},
		"hole":    {givePolygon: holed, giveRes: 5, wantFull: 2123, wantOverlapping: 2436},
		"pacific": {givePolygon: geodesicPacific, giveRes: 4, wantFull: 2246, wantOverlapping: 2554},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			full := geodesicFill(t, tt.givePolygon, tt.giveRes, ContainmentFull)
			center := geodesicFill(t, tt.givePolygon, tt.giveRes, ContainmentCenter)
			overlapping := geodesicFill(t, tt.givePolygon, tt.giveRes, ContainmentOverlapping)

			if len(full) != tt.wantFull {
				t.Errorf("full: got %d cells, want %d", len(full), tt.wantFull)
			}

			if len(overlapping) != tt.wantOverlapping {
				t.Errorf("overlapping: got %d cells, want %d", len(overlapping), tt.wantOverlapping)
			}

			if len(center) < len(full) || len(center) > len(overlapping) {
				t.Errorf("center count %d outside [full %d, overlapping %d]", len(center), len(full), len(overlapping))
			}

			// The three modes nest: every fully contained cell has its center
			// inside, and every cell with its center inside overlaps.
			for _, cell := range full {
				if _, ok := slices.BinarySearch(center, cell); !ok {
					t.Fatalf("full cell %s missing from center", cell)
				}
			}

			for _, cell := range center {
				if _, ok := slices.BinarySearch(overlapping, cell); !ok {
					t.Fatalf("center cell %s missing from overlapping", cell)
				}
			}
		})
	}
}

// TestGeodesicEmpty covers the loops that yield nothing without an error: no
// vertices at all, and a loop of vertices too close together to hold a cell.
func TestGeodesicEmpty(t *testing.T) {
	t.Parallel()

	cells, err := PolygonToCellsExperimental(GeoPolygon{}, 5, ContainmentOverlapping|ContainmentGeodesic)
	if err != nil || cells != nil {
		t.Fatalf("no vertices: got %v (%v), want nil, nil", cells, err)
	}

	if got := geodesicFill(t, geodesicDegenerate, 5, ContainmentFull); len(got) != 0 {
		t.Fatalf("degenerate loop: got %d cells, want 0", len(got))
	}
}

// TestGeodesicErrors covers every rejection of the geodesic fill: too few
// vertices in the outer loop or a hole, non-finite vertices, modes it does not
// support, flag bits it does not know, loops that do not fit in a hemisphere
// and loops whose vertices all lie on one great circle.
func TestGeodesicErrors(t *testing.T) {
	t.Parallel()

	inf := math.Inf(1)
	nan := math.NaN()

	tests := map[string]struct {
		givePolygon GeoPolygon
		giveMode    ContainmentMode
		wantErr     error
	}{
		"point":         {givePolygon: GeoPolygon{GeoLoop: geodesicPoint}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"line":          {givePolygon: GeoPolygon{GeoLoop: geodesicLine}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"infinite":      {givePolygon: GeoPolygon{GeoLoop: GeoLoop{{Lat: inf, Lng: inf}, {Lat: -inf, Lng: -inf}, {Lat: 0, Lng: 0}}}, giveMode: ContainmentFull, wantErr: ErrDomain},
		"nan":           {givePolygon: GeoPolygon{GeoLoop: GeoLoop{{Lat: nan, Lng: nan}, {Lat: 0, Lng: 0}, {Lat: 1, Lng: 1}}}, giveMode: ContainmentFull, wantErr: ErrDomain},
		"empty_hole":    {givePolygon: GeoPolygon{GeoLoop: geodesicWestUS.GeoLoop, Holes: []GeoLoop{{}}}, giveMode: ContainmentFull, wantErr: ErrDomain},
		"point_hole":    {givePolygon: GeoPolygon{GeoLoop: geodesicWestUS.GeoLoop, Holes: []GeoLoop{geodesicPoint}}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"line_hole":     {givePolygon: GeoPolygon{GeoLoop: geodesicWestUS.GeoLoop, Holes: []GeoLoop{geodesicLine}}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"bbox_mode":     {givePolygon: geodesicWestUS, giveMode: ContainmentOverlappingBbox, wantErr: ErrOptionInvalid},
		"invalid_mode":  {givePolygon: geodesicWestUS, giveMode: ContainmentInvalid, wantErr: ErrOptionInvalid},
		"beyond_mode":   {givePolygon: geodesicWestUS, giveMode: ContainmentInvalid + 1, wantErr: ErrOptionInvalid},
		"unknown_flag":  {givePolygon: geodesicWestUS, giveMode: ContainmentFull | ContainmentGeodesic<<1, wantErr: ErrOptionInvalid},
		"wide_wrap":     {givePolygon: GeoPolygon{GeoLoop: geodesicWideWrap()}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"tall_slice":    {givePolygon: GeoPolygon{GeoLoop: degLoop([][2]float64{{60, 0}, {0, 120}, {-60, 0}, {0, -120}})}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"equator_wrap":  {givePolygon: GeoPolygon{GeoLoop: degLoop([][2]float64{{10, 0}, {10, 120}, {10, -120}, {-10, -120}, {-10, 120}, {-10, 0}})}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"great_circle":  {givePolygon: GeoPolygon{GeoLoop: degLoop([][2]float64{{0, -60}, {0, -20}, {0, 20}, {0, 60}})}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
		"meridian_loop": {givePolygon: GeoPolygon{GeoLoop: degLoop([][2]float64{{-30, 10}, {0, 10}, {30, 10}, {60, 10}})}, giveMode: ContainmentOverlapping, wantErr: ErrDomain},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := PolygonToCellsExperimental(tt.givePolygon, 2, tt.giveMode|ContainmentGeodesic)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// geodesicWideWrap is a thin band around almost every longitude: too wide for
// any hemisphere.
func geodesicWideWrap() GeoLoop {
	var loop GeoLoop
	for lng := -170.0; lng <= 170; lng += 60 {
		loop = append(loop, LatLng{Lat: 10, Lng: lng})
	}

	for lng := 170.0; lng >= -170; lng -= 60 {
		loop = append(loop, LatLng{Lat: -10, Lng: lng})
	}

	return loop
}

// TestGeodesicOrientationInvariant fills every res 0 and res 1 cell's own
// boundary, wound both ways, and requires identical results in full and
// overlapping modes. A cell edge lying on a polygon edge is the coincident-arc
// case that a too-tight same-side epsilon once made sensitive to vertex order.
func TestGeodesicOrientationInvariant(t *testing.T) {
	t.Parallel()

	for res := 0; res <= 1; res++ {
		t.Run(strconv.Itoa(res), func(t *testing.T) {
			t.Parallel()

			for cell := range sampledCellsAtRes(t, res, 1+6*res) {
				boundary, err := cell.Boundary()
				if err != nil {
					t.Fatal(err)
				}

				ccw := GeoPolygon{GeoLoop: GeoLoop(boundary)}
				cw := GeoPolygon{GeoLoop: slices.Clone(GeoLoop(boundary))}
				slices.Reverse(cw.GeoLoop)

				for _, mode := range []ContainmentMode{ContainmentFull, ContainmentOverlapping} {
					gotCCW := geodesicFill(t, ccw, res, mode)
					gotCW := geodesicFill(t, cw, res, mode)

					if !slices.Equal(gotCCW, gotCW) {
						t.Fatalf("%s mode %d: ccw %v, cw %v", cell, mode, gotCCW, gotCW)
					}
				}
			}
		})
	}
}

// squareAround returns a loop of the given half-width in degrees about a point.
func squareAround(center LatLng, halfWidth float64) GeoLoop {
	return degLoop([][2]float64{
		{center.Lat + halfWidth, center.Lng - halfWidth}, {center.Lat + halfWidth, center.Lng + halfWidth},
		{center.Lat - halfWidth, center.Lng + halfWidth}, {center.Lat - halfWidth, center.Lng - halfWidth},
	})
}

// cellsInBox returns the cells at res whose centers lie in the box, from a
// planar center fill, so that an oracle test need not visit the whole globe.
func cellsInBox(t *testing.T, latLo, latHi, lngLo, lngHi float64, res int) []Cell {
	t.Helper()

	box := GeoPolygon{GeoLoop: degLoop([][2]float64{{latLo, lngLo}, {latLo, lngHi}, {latHi, lngHi}, {latHi, lngLo}})}

	cells, err := PolygonToCellsExperimental(box, res, ContainmentCenter)
	if err != nil {
		t.Fatal(err)
	}

	return cells
}

// sampledCellsAtRes yields every stride-th cell at res, and every pentagon,
// in the order of allCellsAtRes. The race detector makes exhaustive sweeps
// too slow for CI, and the geometry repeats across a resolution.
func sampledCellsAtRes(t *testing.T, res, stride int) func(func(Cell) bool) {
	t.Helper()

	return func(yield func(Cell) bool) {
		index := 0

		for cell := range allCellsAtRes(t, res) {
			index++

			if (index%stride == 0 || cell.IsPentagon()) && !yield(cell) {
				return
			}
		}
	}
}

// allCellsAtRes yields every cell at the resolution.
func allCellsAtRes(t *testing.T, res int) func(func(Cell) bool) {
	t.Helper()

	return func(yield func(Cell) bool) {
		bases, err := Res0Cells()
		if err != nil {
			t.Fatal(err)
		}

		for _, base := range bases {
			children, err := base.Children(res)
			if err != nil {
				t.Fatal(err)
			}

			for _, child := range children {
				if !yield(child) {
					return
				}
			}
		}
	}
}

// TestGeodesicFullSmallerThanCell checks that a polygon much smaller than a
// cell, centered on the cell's center, overlaps the cell but does not fully
// contain it.
func TestGeodesicFullSmallerThanCell(t *testing.T) {
	t.Parallel()

	const res = 5

	cell, err := LatLngToCell(NewLatLng(37.5, -122.0), res)
	if err != nil {
		t.Fatal(err)
	}

	center, err := cell.LatLng()
	if err != nil {
		t.Fatal(err)
	}

	// About 1.3 km across, well inside a res 5 cell.
	const d = 0.0002 * RadsToDegs

	triangle := GeoPolygon{GeoLoop: GeoLoop{
		{Lat: center.Lat + d, Lng: center.Lng},
		{Lat: center.Lat - d, Lng: center.Lng + d},
		{Lat: center.Lat - d, Lng: center.Lng - d},
	}}

	if got := geodesicFill(t, triangle, res, ContainmentOverlapping); !slices.Contains(got, cell) {
		t.Fatalf("overlapping: %v does not contain the enclosing cell %s", got, cell)
	}

	if got := geodesicFill(t, triangle, res, ContainmentFull); len(got) != 0 {
		t.Fatalf("full: got %v, want no cells", got)
	}
}

// TestGeodesicFullHoleInsideCell checks that a tiny hole inside a cell, away
// from its center and crossing no edge, removes the cell from the full fill.
func TestGeodesicFullHoleInsideCell(t *testing.T) {
	t.Parallel()

	const res = 5

	cell, err := LatLngToCell(NewLatLng(37.5, -117.5), res)
	if err != nil {
		t.Fatal(err)
	}

	center, err := cell.LatLng()
	if err != nil {
		t.Fatal(err)
	}

	const (
		offset = 0.0005 * RadsToDegs
		radius = 0.0001 * RadsToDegs
	)

	hole := GeoLoop{
		{Lat: center.Lat + offset + radius, Lng: center.Lng},
		{Lat: center.Lat + offset - radius, Lng: center.Lng + radius},
		{Lat: center.Lat + offset - radius, Lng: center.Lng - radius},
	}
	outer := squareAround(center, 0.5)
	holed := GeoPolygon{GeoLoop: outer, Holes: []GeoLoop{hole}}

	if got := geodesicFill(t, GeoPolygon{GeoLoop: outer}, res, ContainmentFull); !slices.Contains(got, cell) {
		t.Fatalf("without the hole, %s is not fully contained", cell)
	}

	if got := geodesicFill(t, holed, res, ContainmentFull); slices.Contains(got, cell) {
		t.Fatalf("with the hole inside it, %s is still fully contained", cell)
	}

	// Overlapping mode keeps the cell: the hole is inside it, so the filled
	// region around the hole overlaps the cell even though its center is out.
	if got := geodesicFill(t, holed, res, ContainmentOverlapping); !slices.Contains(got, cell) {
		t.Fatalf("with the hole inside it, %s does not overlap", cell)
	}
}

// TestGeodesicDiskRoundtrip builds the outline of every grid disk at the
// configured resolutions and radii with CellsToMultiPolygon and requires the
// overlapping geodesic fill to return every cell of the disk. Disks whose
// outline is more than one polygon or has holes are skipped.
func TestGeodesicDiskRoundtrip(t *testing.T) {
	t.Parallel()

	configs := []struct{ res, k, stride int }{{0, 0, 1}, {0, 1, 1}, {0, 2, 1}, {1, 0, 3}, {1, 1, 3}, {2, 0, 12}}

	for _, config := range configs {
		t.Run("res"+strconv.Itoa(config.res)+"k"+strconv.Itoa(config.k), func(t *testing.T) {
			t.Parallel()

			checked := 0

			for center := range sampledCellsAtRes(t, config.res, config.stride) {
				disk, err := center.GridDisk(config.k)
				if err != nil {
					t.Fatal(err)
				}

				polygons, err := CellsToMultiPolygon(disk)
				if err != nil {
					t.Fatal(err)
				}

				if len(polygons) != 1 || len(polygons[0].Holes) != 0 {
					continue
				}

				got := geodesicFill(t, polygons[0], config.res, ContainmentOverlapping)
				for _, cell := range disk {
					if _, ok := slices.BinarySearch(got, cell); !ok {
						t.Fatalf("disk around %s: %s missing from the fill of its outline", center, cell)
					}
				}

				checked++
			}

			if checked == 0 {
				t.Fatal("no disks were checked")
			}
		})
	}
}

// TestGeodesicSphericalCircle measures the fill against an oracle independent
// of the algorithm: a 360-gon inscribed in a spherical circle of radius 20°
// about a point. A cell center is inside the circle exactly when its angular
// distance from the center is below the radius, and the polygon differs from
// the circle only in the thin band between each inscribed chord and its arc.
// Outside that band the geodesic center fill must match the oracle exactly.
func TestGeodesicSphericalCircle(t *testing.T) {
	t.Parallel()

	const (
		res       = 3
		sides     = 360
		radius    = 20 * DegsToRads
		centerLat = 45.0
		centerLng = 20.0
	)

	center := latLngToVec3(centerLat*DegsToRads, centerLng*DegsToRads)
	north, east := center.tangentBasis()

	loop := make(GeoLoop, sides)
	for i := range loop {
		theta := 2 * math.Pi * float64(i) / sides
		dir := east.linComb(math.Cos(theta), math.Sin(theta), north)
		loop[i] = center.linComb(math.Cos(radius), math.Sin(radius), dir).toLatLng()
	}

	// The inscribed chord sags below the arc by at most this much.
	band := radius - math.Atan(math.Tan(radius)*math.Cos(math.Pi/sides))

	geodesic := geodesicFill(t, GeoPolygon{GeoLoop: loop}, res, ContainmentCenter)

	if len(geodesic) == 0 {
		t.Fatal("the fill is empty")
	}

	candidates := cellsInBox(t, centerLat-25, centerLat+25, centerLng-50, centerLng+50, res)
	slices.Sort(candidates)

	inside := 0

	for _, cell := range candidates {
		point, err := cell.centerVec3()
		if err != nil {
			t.Fatal(err)
		}

		distance := math.Acos(math.Min(1, point.dot(center)))
		_, got := slices.BinarySearch(geodesic, cell)

		switch {
		case distance < radius-band:
			inside++

			if !got {
				t.Errorf("%s at %.4f rad is inside the circle but not in the fill", cell, distance)
			}
		case distance > radius && got:
			t.Errorf("%s at %.4f rad is outside the circle but in the fill", cell, distance)
		}
	}

	if inside == 0 {
		t.Fatal("no cells inside the circle")
	}

	for _, cell := range geodesic {
		if _, ok := slices.BinarySearch(candidates, cell); !ok {
			t.Fatalf("%s is in the fill but outside the box of candidates", cell)
		}
	}
}

// TestGeodesicParallelQuadrilateral is the case the geodesic mode exists for.
// The polygon joins 40°N and 30°N between 120°W and 60°W. Its northern edge,
// read as a great-circle arc, climbs to 44.1°N at 90°W; read as a straight
// line in latitude and longitude it stays at 40°N. The arc between two points
// on one parallel has a closed form for its latitude at any longitude, so
// every cell center can be classified without the algorithm under test. The
// geodesic center fill must match that oracle on every cell, and the planar
// fill must disagree with it on the cells between the chord and the arc.
func TestGeodesicParallelQuadrilateral(t *testing.T) {
	t.Parallel()

	const (
		res          = 4
		north, south = 40.0, 30.0
		west, east   = -120.0, -60.0
		edgeTol      = 1e-9 // degrees; a center this close to an edge is a knife edge
	)

	polygon := GeoPolygon{GeoLoop: degLoop([][2]float64{{north, west}, {north, east}, {south, east}, {south, west}})}

	// arcLat is the latitude at lng of the great circle through (lat, west)
	// and (lat, east).
	arcLat := func(lat, lng float64) float64 {
		tanLat := math.Tan(lat * DegsToRads)
		num := tanLat * (math.Sin((east-lng)*DegsToRads) + math.Sin((lng-west)*DegsToRads))

		return math.Atan(num/math.Sin((east-west)*DegsToRads)) * RadsToDegs
	}

	geodesic := geodesicFill(t, polygon, res, ContainmentCenter)

	planar, err := PolygonToCellsExperimental(polygon, res, ContainmentCenter)
	if err != nil {
		t.Fatal(err)
	}

	slices.Sort(planar)

	var inside, planarWrong int

	for _, cell := range cellsInBox(t, south-1, north+6, west-1, east+1, res) {
		center, err := cell.LatLng()
		if err != nil {
			t.Fatal(err)
		}

		top, bottom := arcLat(north, center.Lng), arcLat(south, center.Lng)
		if math.Abs(center.Lng-west) < edgeTol || math.Abs(center.Lng-east) < edgeTol ||
			math.Abs(center.Lat-top) < edgeTol || math.Abs(center.Lat-bottom) < edgeTol {
			continue
		}

		want := center.Lng > west && center.Lng < east && center.Lat > bottom && center.Lat < top
		if want {
			inside++
		}

		if _, got := slices.BinarySearch(geodesic, cell); got != want {
			t.Errorf("%s at %v: geodesic fill %v, oracle %v", cell, center, got, want)
		}

		if _, got := slices.BinarySearch(planar, cell); got != want {
			planarWrong++
		}
	}

	if inside == 0 {
		t.Fatal("no cells inside the quadrilateral")
	}

	if planarWrong == 0 {
		t.Fatal("the planar fill agreed with the oracle everywhere, so this test shows nothing")
	}

	t.Logf("%d cells inside; the planar fill is wrong on %d cells", inside, planarWrong)
}

// TestSphereCapEnclosesDescendants checks that a cell's bounding cap, the
// coarse-level pruning test, holds every boundary vertex of its descendants
// three resolutions down at res 0, where the overshoot is largest, and two
// and one levels down at res 1 and 2 to keep the sweep exhaustive but short.
// Children stay within their parent's edge length but grandchildren reach
// beyond it, which is why the cap is wider than the reference library's.
func TestSphereCapEnclosesDescendants(t *testing.T) {
	t.Parallel()

	for res := 0; res <= 2; res++ {
		t.Run(strconv.Itoa(res), func(t *testing.T) {
			t.Parallel()

			for cell := range allCellsAtRes(t, res) {
				region, err := cell.sphereCap()
				if err != nil {
					t.Fatal(err)
				}

				children, err := cell.Children(3)
				if err != nil {
					t.Fatal(err)
				}

				for _, child := range children {
					var buf [MaxCellBndryVerts]vec3d

					boundary, err := child.boundaryVec3(buf[:0])
					if err != nil {
						t.Fatal(err)
					}

					for _, vert := range boundary {
						if vert.dot(region.center) < region.cosRadius {
							t.Fatalf("%s: vertex of descendant %s is outside the cap", cell, child)
						}
					}
				}
			}
		})
	}
}

// TestGeodesicPruneKeepsOutlyingGrandchildren finds, for every base cell,
// the res 2 grandchild vertex farthest from the base cell's center, puts a
// tiny triangle on it and requires the overlapping fill to return that
// grandchild. With a cap that does not reach the outlying vertex, the base
// cell is pruned and the grandchild is lost, which is the defect the wider
// cap prevents.
func TestGeodesicPruneKeepsOutlyingGrandchildren(t *testing.T) {
	t.Parallel()

	const (
		depth  = 2
		radius = 1e-4 // radians, about 640 m
	)

	for base := range allCellsAtRes(t, 0) {
		center, err := base.centerVec3()
		if err != nil {
			t.Fatal(err)
		}

		grandchildren, err := base.Children(depth)
		if err != nil {
			t.Fatal(err)
		}

		var (
			farthest     Cell
			farthestVert vec3d
			minDot       = 1.0
		)

		for _, child := range grandchildren {
			var buf [MaxCellBndryVerts]vec3d

			boundary, err := child.boundaryVec3(buf[:0])
			if err != nil {
				t.Fatal(err)
			}

			for _, vert := range boundary {
				if dot := vert.dot(center); dot < minDot {
					farthest, farthestVert, minDot = child, vert, dot
				}
			}
		}

		north, east := farthestVert.tangentBasis()

		loop := make(GeoLoop, 0, 3)

		for _, theta := range []float64{0, 2 * math.Pi / 3, 4 * math.Pi / 3} {
			dir := east.linComb(math.Cos(theta), math.Sin(theta), north)
			loop = append(loop, farthestVert.linComb(math.Cos(radius), math.Sin(radius), dir).toLatLng())
		}

		got := geodesicFill(t, GeoPolygon{GeoLoop: loop}, depth, ContainmentOverlapping)
		if _, ok := slices.BinarySearch(got, farthest); !ok {
			t.Fatalf("a triangle on the farthest vertex of %s (under %s) did not return it: %v", farthest, base, got)
		}
	}
}

// TestGeodesicOverlappingLoopInsideCell covers the overlapping-mode cases where
// the cell's center is outside the filled polygon and no edge crosses the cell
// boundary: the whole polygon sits inside the cell, or a hole does.
func TestGeodesicOverlappingLoopInsideCell(t *testing.T) {
	t.Parallel()

	const res = 5

	cell, err := LatLngToCell(NewLatLng(37.5, -122.0), res)
	if err != nil {
		t.Fatal(err)
	}

	center, err := cell.LatLng()
	if err != nil {
		t.Fatal(err)
	}

	const (
		offset = 0.0005 * RadsToDegs
		radius = 0.0001 * RadsToDegs
	)

	// A tiny triangle inside the cell but away from its center.
	tiny := GeoLoop{
		{Lat: center.Lat + offset + radius, Lng: center.Lng},
		{Lat: center.Lat + offset - radius, Lng: center.Lng + radius},
		{Lat: center.Lat + offset - radius, Lng: center.Lng - radius},
	}

	if got := geodesicFill(t, GeoPolygon{GeoLoop: tiny}, res, ContainmentOverlapping); !slices.Equal(got, []Cell{cell}) {
		t.Fatalf("a polygon inside %s overlaps %v", cell, got)
	}

	// A tiny hole on the cell's center: the center is outside the filled
	// region, yet the region around the hole overlaps the cell.
	hole := GeoLoop{
		{Lat: center.Lat + radius, Lng: center.Lng},
		{Lat: center.Lat - radius, Lng: center.Lng + radius},
		{Lat: center.Lat - radius, Lng: center.Lng - radius},
	}
	holed := GeoPolygon{GeoLoop: squareAround(center, 0.5), Holes: []GeoLoop{hole}}

	if got := geodesicFill(t, holed, res, ContainmentOverlapping); !slices.Contains(got, cell) {
		t.Fatalf("a hole on the center of %s removed it from the overlapping fill", cell)
	}

	if got := geodesicFill(t, holed, res, ContainmentFull); slices.Contains(got, cell) {
		t.Fatalf("a hole on the center of %s left it fully contained", cell)
	}
}

// TestGeodesicCellsEarlyStop stops the traversal at its first cell, covering
// the iterator's early return.
func TestGeodesicCellsEarlyStop(t *testing.T) {
	t.Parallel()

	polygon, err := newGeodesicPolygon(geodesicWestUS)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	for range polygon.cells(3, ContainmentOverlapping) {
		count++

		break
	}

	if count != 1 {
		t.Fatalf("visited %d cells, want 1", count)
	}
}

// TestGeodesicInvalidCell covers the error returns of the vector-valued cell
// helpers, which the fill itself never reaches because it only visits valid
// cells.
func TestGeodesicInvalidCell(t *testing.T) {
	t.Parallel()

	invalid := setH3Index(0, NumBaseCells+1, centerDigit)

	if _, err := invalid.centerVec3(); err == nil {
		t.Error("centerVec3 accepted an invalid cell")
	}

	if _, err := invalid.sphereCap(); err == nil {
		t.Error("sphereCap accepted an invalid cell")
	}

	if _, err := invalid.boundaryVec3(nil); err == nil {
		t.Error("boundaryVec3 accepted an invalid cell")
	}
}

// --- Internal predicates ---

// loopFromDegrees builds a geodesic loop for the internal tests.
func loopFromDegrees(t *testing.T, verts [][2]float64) geodesicLoop {
	t.Helper()

	loop, err := newGeodesicLoop(degLoop(verts), false)
	if err != nil {
		t.Fatal(err)
	}

	return loop
}

// TestGeodesicEdgesCross covers each branch of the arc intersection test.
func TestGeodesicEdgesCross(t *testing.T) {
	t.Parallel()

	point := func(lat, lng float64) vec3d { return latLngToVec3(lat*DegsToRads, lng*DegsToRads) }

	// Two arcs on the equator: collinear, overlapping and disjoint.
	a1, a2 := point(0, 0), point(0, 10)
	b1, b2 := point(0, 5), point(0, 15)
	c1, c2 := point(0, 20), point(0, 30)
	equatorNormal := point(0, 0).cross(point(0, 90))

	if !geodesicEdgesCross(a1, a2, b1, b2, equatorNormal) {
		t.Error("overlapping collinear arcs do not cross")
	}

	if geodesicEdgesCross(a1, a2, c1, c2, equatorNormal) {
		t.Error("disjoint collinear arcs cross")
	}

	// A degenerate arc of one point on the equator has no direction to project
	// along.
	if geodesicEdgesCross(a1, a1, b1, b2, equatorNormal) {
		t.Error("a zero-length arc crosses")
	}

	// A meridian arc crossing the equator, and one stopping short of it.
	m1, m2 := point(-5, 5), point(5, 5)
	if !geodesicEdgesCross(m1, m2, a1, a2, equatorNormal) {
		t.Error("a meridian arc through the equator does not cross it")
	}

	n1, n2 := point(1, 5), point(5, 5)
	if geodesicEdgesCross(n1, n2, a1, a2, equatorNormal) {
		t.Error("a meridian arc above the equator crosses it")
	}

	// Exactly antipodal endpoints leave the arc's midpoint undefined and count
	// as crossing. Points converted from degrees miss exact antipodes by a
	// rounding error, so the arc is given as unit vectors.
	antipode1, antipode2 := vec3d{x: 1}, vec3d{x: -1}
	if !geodesicEdgesCross(m1, m2, antipode1, antipode2, vec3d{z: 1}) {
		t.Error("an arc against antipodal endpoints does not cross")
	}

	// As the first arc, the antipodes have no plane and fall through the
	// collinear case.
	if !geodesicEdgesCross(antipode1, antipode2, m1, m2, m1.cross(m2)) {
		t.Error("an arc with antipodal endpoints does not cross")
	}

	// Two arcs whose great circles meet, but away from both arcs.
	if geodesicEdgesCross(point(10, 10), point(20, 10), point(0, 50), point(0, 60), equatorNormal) {
		t.Error("arcs far from their circles' intersection cross")
	}

	if geodesicEdgesCross(point(-10, 10), point(10, 10), point(0, 50), point(0, 60), equatorNormal) {
		t.Error("an arc crossing the equator far from the other arc crosses it")
	}
}

// TestGeodesicBoundaryIntersectsParallel ports the reference's coverage cases:
// a boundary edge running between two nearly parallel polygon edges, and a
// boundary whose projections come out reversed.
func TestGeodesicBoundaryIntersectsParallel(t *testing.T) {
	t.Parallel()

	point := func(lat, lng float64) vec3d { return latLngToVec3(lat*DegsToRads, lng*DegsToRads) }

	thin, err := newGeodesicPolygon(GeoPolygon{GeoLoop: degLoop([][2]float64{{0, 0}, {0, 1}, {0.01, 1}, {0.01, 0}})})
	if err != nil {
		t.Fatal(err)
	}

	boundary := []vec3d{point(0.005, 0), point(0.005, 1)}
	if !thin.boundaryIntersects(boundary, sphereCap{center: boundary[0], cosRadius: math.Cos(2 * DegsToRads)}) {
		t.Error("a boundary spanning a thin polygon does not intersect it")
	}

	triangle, err := newGeodesicPolygon(GeoPolygon{GeoLoop: degLoop([][2]float64{{10, 0}, {0, 0}, {0, 10}})})
	if err != nil {
		t.Fatal(err)
	}

	boundary = []vec3d{point(5, 5), point(0, 5), point(2.5, 2.5)}
	if !triangle.boundaryIntersects(boundary, sphereCap{center: boundary[0], cosRadius: -0.5}) {
		t.Error("a boundary across the triangle's edge does not intersect it")
	}

	// A cap far from every edge prunes them all.
	if triangle.boundaryIntersects(boundary, sphereCap{center: point(-80, 180), cosRadius: math.Cos(DegsToRads)}) {
		t.Error("a distant cap intersects")
	}
}

// TestGeodesicBoundingCap checks that the outer loop's bounding cap sits on
// the hemisphere pole and reaches the farthest vertex.
func TestGeodesicBoundingCap(t *testing.T) {
	t.Parallel()

	verts := [][2]float64{{0, 0}, {0, 10}, {10, 5}, {5, -3}}

	loop, err := newGeodesicLoop(degLoop(verts), true)
	if err != nil {
		t.Fatal(err)
	}

	want := 0.0
	for _, vert := range degLoop(verts) {
		want = math.Max(want, math.Acos(loop.hemispherePole.dot(latLngToVec3(vert.Lat*DegsToRads, vert.Lng*DegsToRads))))
	}

	center, radius := loop.boundingCap()
	if center != loop.hemispherePole {
		t.Errorf("center %v, want the hemisphere pole %v", center, loop.hemispherePole)
	}

	if math.Abs(radius-want) > 1e-12 || want == 0 {
		t.Errorf("radius %.15f rad, want %.15f", radius, want)
	}
}

// TestGeodesicContainsPoint covers the hemisphere-pole rejection and a polygon
// near the pole against a point near the opposite pole, whose box lies off the
// unit sphere's far side.
func TestGeodesicContainsPoint(t *testing.T) {
	t.Parallel()

	point := func(lat, lng float64) vec3d { return latLngToVec3(lat*DegsToRads, lng*DegsToRads) }

	polar, err := newGeodesicPolygon(GeoPolygon{GeoLoop: degLoop([][2]float64{{89, 0}, {89, 1}, {89.5, 0.5}})})
	if err != nil {
		t.Fatal(err)
	}

	if polar.containsPoint(point(-89, 180)) {
		t.Error("a point by the south pole is inside a polygon by the north pole")
	}

	if !polar.containsPoint(point(89.2, 0.5)) {
		t.Error("a point inside the polar triangle is outside")
	}

	if polar.capIntersects(sphereCap{center: point(-89, 180), cosRadius: math.Cos(DegsToRads)}, 0) {
		t.Error("a cap by the south pole intersects a polygon by the north pole")
	}

	// The polygon's box reaches about 14° toward a res 4 cell on the same
	// parallel, but its bounding cap does not; both tests must agree.
	nearby, err := newGeodesicPolygon(GeoPolygon{GeoLoop: degLoop([][2]float64{{41, -109}, {41, -102}, {37, -102}, {37, -109}})})
	if err != nil {
		t.Fatal(err)
	}

	cell, err := LatLngToCell(NewLatLng(39, -116), 4)
	if err != nil {
		t.Fatal(err)
	}

	region, err := cell.sphereCap()
	if err != nil {
		t.Fatal(err)
	}

	if !region.overlapsAABB(nearby.box) {
		t.Error("the box test prunes a cell 7° from the polygon; this test no longer shows the cap test at work")
	}

	if nearby.capIntersects(region, 4) {
		t.Error("a res 4 cell 7° west of the polygon survives the bounding cap")
	}

	inside, err := LatLngToCell(NewLatLng(39, -105), 4)
	if err != nil {
		t.Fatal(err)
	}

	region, err = inside.sphereCap()
	if err != nil {
		t.Fatal(err)
	}

	if !nearby.capIntersects(region, 4) {
		t.Error("a res 4 cell inside the polygon is pruned")
	}

	// A hole excludes its interior.
	holed, err := newGeodesicPolygon(GeoPolygon{
		GeoLoop: degLoop([][2]float64{{0, 0}, {0, 10}, {10, 10}, {10, 0}}),
		Holes:   []GeoLoop{degLoop([][2]float64{{4, 4}, {4, 6}, {6, 6}, {6, 4}})},
	})
	if err != nil {
		t.Fatal(err)
	}

	if holed.containsPoint(point(5, 5)) {
		t.Error("a point inside the hole is inside the polygon")
	}

	if !holed.containsPoint(point(2, 2)) {
		t.Error("a point outside the hole is outside the polygon")
	}
}

// TestSphereCapOverlapsAABB covers both rejections of the cap-box test: the
// box corner nearest the cap outside the cap, and a box entirely outside the
// unit sphere.
func TestSphereCapOverlapsAABB(t *testing.T) {
	t.Parallel()

	region := sphereCap{center: vec3d{z: 1}, cosRadius: math.Cos(10 * DegsToRads)}

	near := aabb{min: vec3d{x: -0.1, y: -0.1, z: 0.9}, max: vec3d{x: 0.1, y: 0.1, z: 1}}
	if !region.overlapsAABB(near) {
		t.Error("a box around the cap center does not overlap")
	}

	far := aabb{min: vec3d{x: -0.1, y: -0.1, z: -1}, max: vec3d{x: 0.1, y: 0.1, z: -0.9}}
	if region.overlapsAABB(far) {
		t.Error("a box at the antipode overlaps")
	}

	// Within the cap's cone but beyond the sphere.
	outside := aabb{min: vec3d{x: 1.5, y: 1.5, z: 1.5}, max: vec3d{x: 2, y: 2, z: 2}}
	if (sphereCap{center: vec3d{x: 1, y: 1, z: 1}.normalized(), cosRadius: -1}).overlapsAABB(outside) {
		t.Error("a box off the sphere overlaps")
	}
}

// TestAABBArcExtrema checks that an arc's box covers its interior extremum and
// skips the degenerate cases: a zero normal, and an axis along the normal.
func TestAABBArcExtrema(t *testing.T) {
	t.Parallel()

	point := func(lat, lng float64) vec3d { return latLngToVec3(lat*DegsToRads, lng*DegsToRads) }

	// An equatorial arc from 80°E to 100°E peaks at y = 1 between its endpoints.
	v1, v2 := point(0, 80), point(0, 100)
	box := emptyAABB()
	box.update(v1)
	box.update(v2)
	box.updateWithArcExtrema(v1, v2, v1.cross(v2))

	if box.max.y != 1 {
		t.Errorf("max y = %v, want 1", box.max.y)
	}

	// The z axis is the equator's normal and has no extremum on it; the box
	// must not grow in z.
	if box.min.z != 0 || box.max.z != 0 {
		t.Errorf("z range [%v, %v], want [0, 0]", box.min.z, box.max.z)
	}

	// A degenerate edge has a zero normal and leaves the box alone.
	before := box
	box.updateWithArcExtrema(v1, v1, vec3d{})

	if box != before {
		t.Error("a zero normal changed the box")
	}
}

// TestHemispherePole covers each candidate in turn: the centroid, a vertex, a
// vertex-pair cross product, its opposite, and no pole at all.
func TestHemispherePole(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveVerts [][2]float64
		wantOK    bool
	}{
		// A small triangle: the centroid works.
		"centroid": {giveVerts: [][2]float64{{0, 0}, {0, 10}, {10, 5}}, wantOK: true},
		// The vertex at 60°N pulls the centroid north and east until it
		// fails on the western vertex; the vertex (0, 0) is a pole.
		"vertex": {giveVerts: [][2]float64{{0, -90}, {0, 90}, {0, 0}, {60, 90}}, wantOK: true},
		// Four points on the equator 90° apart sum to zero and none bounds
		// the others; the cross product of two of them, a pole, does.
		"cross": {giveVerts: [][2]float64{{0, 0}, {0, 90}, {0, 180}, {0, -90}}, wantOK: true},
		// The same four with a northern vertex: the first pair's cross
		// product points south and fails, and its opposite is the pole.
		"opposite": {giveVerts: [][2]float64{{0, 0}, {0, -90}, {0, 180}, {0, 90}, {45, 0}}, wantOK: true},
		// Spread over more than a hemisphere: the tall slice the fill rejects
		// through the great-circle check first; here the pole search alone
		// runs.
		"none": {giveVerts: [][2]float64{{60, 0}, {0, 120}, {-60, 0}, {0, -120}, {10, 10}}, wantOK: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			loop := degLoop(tt.giveVerts)
			edges := make([]geodesicEdge, len(loop))

			for i, vert := range loop {
				edges[i].vert = latLngToVec3(vert.Lat*DegsToRads, vert.Lng*DegsToRads)
			}

			pole, ok := hemispherePole(edges)
			if ok != tt.wantOK {
				t.Fatalf("got ok=%v pole=%v, want ok=%v", ok, pole, tt.wantOK)
			}

			if ok && !candidateContainsVerts(pole, edges) {
				t.Fatalf("pole %v does not hold every vertex", pole)
			}
		})
	}

	if candidateContainsVerts(vec3d{}, nil) {
		t.Error("a zero candidate is a pole")
	}
}

// TestVerticesOnGreatCircle covers the short-loop exemption, a loop with no
// usable normal, a loop on a great circle and a loop off it.
func TestVerticesOnGreatCircle(t *testing.T) {
	t.Parallel()

	edgesOf := func(verts [][2]float64) []geodesicEdge {
		loop := degLoop(verts)
		edges := make([]geodesicEdge, len(loop))

		for i, vert := range loop {
			edges[i].vert = latLngToVec3(vert.Lat*DegsToRads, vert.Lng*DegsToRads)
		}

		return edges
	}

	if verticesOnGreatCircle(edgesOf([][2]float64{{0, 0}, {0, 10}, {0, 20}})) {
		t.Error("a three-vertex loop is rejected as a great circle")
	}

	if verticesOnGreatCircle(edgesOf([][2]float64{{0, 0}, {0, 0}, {0, 0}, {0, 0}})) {
		t.Error("coincident vertices with no normal are a great circle")
	}

	if !verticesOnGreatCircle(edgesOf([][2]float64{{0, 0}, {0, 10}, {0, 20}, {0, 30}})) {
		t.Error("four equatorial vertices are not a great circle")
	}

	if verticesOnGreatCircle(edgesOf([][2]float64{{0, 0}, {0, 10}, {1, 20}, {0, 30}})) {
		t.Error("a loop with one vertex off the circle is a great circle")
	}
}

// TestGeodesicLoopCentroid checks the loop's normalized centroid for a small
// square about a known point, and the box of a loop enclosing a cardinal
// point.
func TestGeodesicLoopCentroid(t *testing.T) {
	t.Parallel()

	loop := loopFromDegrees(t, [][2]float64{{-1, -1}, {-1, 1}, {1, 1}, {1, -1}})
	if math.Abs(loop.centroid.x-1) > 1e-6 {
		t.Errorf("centroid %v, want the x axis", loop.centroid)
	}

	// A loop around the north pole encloses (0, 0, 1), so its box reaches z = 1.
	polar := loopFromDegrees(t, [][2]float64{{80, 0}, {80, 90}, {80, 180}, {80, -90}})
	if box := polar.toAABB(); box.max.z != 1 {
		t.Errorf("polar loop box max z = %v, want 1", box.max.z)
	}
}
