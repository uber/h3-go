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
	"fmt"
	"sort"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// cgoLoopKey renders a cgo loop as a rounded, ordered vertex string for set
// comparison.
func cgoLoopKey(loop h3.GeoLoop) string {
	parts := make([]string, len(loop))
	for i, point := range loop {
		parts[i] = fmt.Sprintf("%.7f,%.7f", point.Lat, point.Lng)
	}

	return fmt.Sprint(parts)
}

// goLoopKey renders a pure-Go loop the same way as cgoLoopKey.
func goLoopKey(loop h3go.GeoLoop) string {
	parts := make([]string, len(loop))
	for i, point := range loop {
		parts[i] = fmt.Sprintf("%.7f,%.7f", point.Lat, point.Lng)
	}

	return fmt.Sprint(parts)
}

// cgoLoopSet returns every loop (outer and holes) across all polygons, sorted.
func cgoLoopSet(polygons []h3.GeoPolygon) []string {
	var keys []string
	for _, poly := range polygons {
		keys = append(keys, cgoLoopKey(poly.GeoLoop))
		for _, hole := range poly.Holes {
			keys = append(keys, cgoLoopKey(hole))
		}
	}

	sort.Strings(keys)

	return keys
}

// goLoopSet returns every loop (outer and holes) across all polygons, sorted.
func goLoopSet(polygons []h3go.GeoPolygon) []string {
	var keys []string
	for _, poly := range polygons {
		keys = append(keys, goLoopKey(poly.GeoLoop))
		for _, hole := range poly.Holes {
			keys = append(keys, goLoopKey(hole))
		}
	}

	sort.Strings(keys)

	return keys
}

// holeCounts returns the per-polygon hole counts, sorted, for either result.
func holeCounts[P any](polygons []P, count func(P) int) []int {
	out := make([]int, len(polygons))
	for i, poly := range polygons {
		out[i] = count(poly)
	}

	sort.Ints(out)

	return out
}

// assertSameMultiPolygon compares two multipolygon results: same polygon count,
// same per-polygon hole counts, and the same set of loops.
func assertSameMultiPolygon(t *testing.T, got []h3go.GeoPolygon, want []h3.GeoPolygon, msg string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: polygon count cgo=%d h3go=%d", msg, len(want), len(got))
	}

	gotHoles := holeCounts(got, func(p h3go.GeoPolygon) int { return len(p.Holes) })
	wantHoles := holeCounts(want, func(p h3.GeoPolygon) int { return len(p.Holes) })

	if fmt.Sprint(gotHoles) != fmt.Sprint(wantHoles) {
		t.Fatalf("%s: hole counts cgo=%v h3go=%v", msg, wantHoles, gotHoles)
	}

	gotLoops := goLoopSet(got)
	wantLoops := cgoLoopSet(want)

	if len(gotLoops) != len(wantLoops) {
		t.Fatalf("%s: loop count cgo=%d h3go=%d", msg, len(wantLoops), len(gotLoops))
	}

	for i := range wantLoops {
		if gotLoops[i] != wantLoops[i] {
			t.Fatalf("%s: loop mismatch:\ncgo  %s\nh3go %s", msg, wantLoops[i], gotLoops[i])
		}
	}
}

// multiPolygonCellSets returns named cell sets covering single cells, contiguous
// blobs, rings (which produce a hole), disjoint regions, and a pentagon area.
func multiPolygonCellSets(t *testing.T) map[string][]h3.Cell {
	t.Helper()

	origin, err := h3.LatLngToCell(h3.LatLng{Lat: 37.78, Lng: -122.42}, 7)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	disk, err := origin.GridDisk(2)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	ring, err := origin.GridRing(2)
	if err != nil {
		t.Fatalf("GridRing: %v", err)
	}

	far, err := h3.LatLngToCell(h3.LatLng{Lat: -20, Lng: 30}, 7)
	if err != nil {
		t.Fatalf("LatLngToCell far: %v", err)
	}

	farDisk, err := far.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk far: %v", err)
	}

	disjoint := append(append([]h3.Cell{}, disk...), farDisk...)

	// A pentagon base cell and its neighborhood.
	pentagon := h3.Cell(0x85080003fffffff)

	pentDisk, err := pentagon.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk pentagon: %v", err)
	}

	return map[string][]h3.Cell{
		"single":   {origin},
		"disk":     disk,
		"ring":     ring,
		"disjoint": disjoint,
		"pentagon": pentDisk,
	}
}

// TestCellsToMultiPolygonMatchesCgo asserts CellsToMultiPolygon matches the cgo
// reference across several cell sets.
func TestCellsToMultiPolygonMatchesCgo(t *testing.T) {
	t.Parallel()

	for name, cells := range multiPolygonCellSets(t) {
		want, wantErr := h3.CellsToMultiPolygon(cells)
		got, gotErr := h3go.CellsToMultiPolygon(toGoCells(cells))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("%s: error mismatch cgo=%v h3go=%v", name, wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		assertSameMultiPolygon(t, got, want, name)
	}
}

// TestCellsToMultiPolygonGlobe asserts the whole-globe case (all base cells)
// matches the cgo reference.
func TestCellsToMultiPolygonGlobe(t *testing.T) {
	t.Parallel()

	cells, err := h3.Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	want, wantErr := h3.CellsToMultiPolygon(cells)
	got, gotErr := h3go.CellsToMultiPolygon(toGoCells(cells))

	if !bothErr(wantErr, gotErr) {
		t.Fatalf("globe error mismatch cgo=%v h3go=%v", wantErr, gotErr)
	}

	assertSameMultiPolygon(t, got, want, "globe")
}

// TestCellsToMultiPolygonErrors asserts validation errors match the cgo
// reference for empty, mismatched-resolution, duplicate, and invalid inputs.
func TestCellsToMultiPolygonErrors(t *testing.T) {
	t.Parallel()

	origin, err := h3.LatLngToCell(h3.LatLng{Lat: 0, Lng: 0}, 7)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	parent, err := origin.Parent(6)
	if err != nil {
		t.Fatalf("Parent: %v", err)
	}

	cases := map[string][]h3.Cell{
		"empty":     {},
		"mixed_res": {origin, parent},
		"duplicate": {origin, origin},
		"invalid":   {h3.Cell(0)},
	}

	for name, cells := range cases {
		want, wantErr := h3.CellsToMultiPolygon(cells)
		got, gotErr := h3go.CellsToMultiPolygon(toGoCells(cells))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("%s: error mismatch cgo=%v h3go=%v", name, wantErr, gotErr)
		}

		if wantErr == nil && len(got) != len(want) {
			t.Fatalf("%s: polygon count cgo=%d h3go=%d", name, len(want), len(got))
		}
	}
}
