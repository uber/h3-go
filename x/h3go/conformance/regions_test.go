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

package conformance

import (
	"math"
	"slices"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// TestRegions runs every regions record in the suite against h3go.
func TestRegions(t *testing.T) {
	t.Parallel()

	tolerances := loadSuite(t).Tolerances

	runGroup(t, "regions/", func(t *testing.T, record RegionsRecord) {
		t.Helper()
		checkRegions(t, record, tolerances)
	})
}

// checkRegions asserts every field of one regions record against h3go.
func checkRegions(t *testing.T, record RegionsRecord, tolerances Tolerances) {
	t.Helper()

	polygon := toGeoPolygon(record.Polygon)

	cells, err := h3go.PolygonToCells(polygon, record.Res)
	if got := cellsResult(cells, err); !equalResults(got, record.Cells, slices.Equal) {
		t.Errorf("PolygonToCells(%d) = %s, want %s", record.Res, formatResult(got), formatResult(record.Cells))
	}

	modes := []struct {
		name string
		mode h3go.ContainmentMode
		want Result[[]string]
	}{
		{"center", h3go.ContainmentCenter, record.Center},
		{"full", h3go.ContainmentFull, record.Full},
		{"overlapping", h3go.ContainmentOverlapping, record.Overlapping},
		{"overlappingBbox", h3go.ContainmentOverlappingBbox, record.OverlappingBbox},
	}

	for _, mode := range modes {
		filled, err := h3go.PolygonToCellsExperimental(polygon, record.Res, mode.mode)
		if got := cellsResult(filled, err); !equalResults(got, mode.want, slices.Equal) {
			t.Errorf("PolygonToCellsExperimental(%d, %s) = %s, want %s", record.Res, mode.name, formatResult(got), formatResult(mode.want))
		}
	}

	if record.Cells.Err != "" {
		if record.MultiPolygon.Err != record.Cells.Err {
			t.Errorf("record multiPolygon error %q differs from cells error %q", record.MultiPolygon.Err, record.Cells.Err)
		}

		return
	}

	input := make([]h3go.Cell, len(record.Cells.Value))
	for i, index := range record.Cells.Value {
		input[i] = h3go.CellFromString(index)
	}

	polygons, err := h3go.CellsToMultiPolygon(input)
	if name := ErrorName(err); name != record.MultiPolygon.Err {
		t.Fatalf("CellsToMultiPolygon() error = %q, want %q", name, record.MultiPolygon.Err)
	}

	if err != nil {
		return
	}

	want := record.MultiPolygon.Value
	if len(polygons) != len(want) {
		t.Fatalf("CellsToMultiPolygon() has %d polygons, want %d", len(polygons), len(want))
	}

	for i, got := range polygons {
		if !loopsMatch(got.GeoLoop, want[i].Outer, tolerances.AngularDeg) {
			t.Errorf("polygon %d outer loop = %v, want %v", i, got.GeoLoop, want[i].Outer)
		}

		if !holesMatch(got.Holes, want[i].Holes, tolerances.AngularDeg) {
			t.Errorf("polygon %d holes = %v, want %v", i, got.Holes, want[i].Holes)
		}
	}
}

// toGeoPolygon converts a record polygon to the h3go input type.
func toGeoPolygon(polygon Polygon) h3go.GeoPolygon {
	out := h3go.GeoPolygon{GeoLoop: toGeoLoop(polygon.Outer)}
	for _, hole := range polygon.Holes {
		out.Holes = append(out.Holes, toGeoLoop(hole))
	}

	return out
}

// toGeoLoop converts a sequence of record points to an h3go loop.
func toGeoLoop(points []GeoCoord) h3go.GeoLoop {
	loop := make(h3go.GeoLoop, len(points))
	for i, point := range points {
		loop[i] = h3go.LatLng{Lat: point[0], Lng: point[1]}
	}

	return loop
}

// withinAngular reports whether two points are within angularDeg degrees of
// arc of each other on the sphere.
func withinAngular(got h3go.LatLng, want GeoCoord, angularDeg float64) bool {
	rads := h3go.GreatCircleDistanceRads(got, h3go.LatLng{Lat: want[0], Lng: want[1]})

	return rads*180/math.Pi <= angularDeg
}

// loopsMatch reports whether two loops visit the same points in the same
// direction, allowing the actual loop to start at any vertex.
func loopsMatch(got h3go.GeoLoop, want []GeoCoord, angularDeg float64) bool {
	if len(got) != len(want) {
		return false
	}

	for offset := range got {
		matched := true

		for i := range want {
			if !withinAngular(got[(offset+i)%len(got)], want[i], angularDeg) {
				matched = false

				break
			}
		}

		if matched {
			return true
		}
	}

	return len(got) == 0
}

// holesMatch reports whether every expected hole has a distinct matching
// actual hole, in any order.
func holesMatch(got []h3go.GeoLoop, want [][]GeoCoord, angularDeg float64) bool {
	if len(got) != len(want) {
		return false
	}

	used := make([]bool, len(got))

	for _, hole := range want {
		found := false

		for i, candidate := range got {
			if !used[i] && loopsMatch(candidate, hole, angularDeg) {
				used[i] = true
				found = true

				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}
