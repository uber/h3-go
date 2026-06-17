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

// regionPolygons returns a set of named test polygons spanning simple squares,
// a polygon with a hole, a transmeridian polygon, and a pentagon-spanning area.
func regionPolygons() map[string]h3.GeoPolygon {
	sfSquare := h3.GeoLoop{
		{Lat: 37.813, Lng: -122.513},
		{Lat: 37.813, Lng: -122.345},
		{Lat: 37.700, Lng: -122.345},
		{Lat: 37.700, Lng: -122.513},
	}

	return map[string]h3.GeoPolygon{
		"sf_square": {GeoLoop: sfSquare},
		"with_hole": {
			GeoLoop: sfSquare,
			Holes: []h3.GeoLoop{{
				{Lat: 37.78, Lng: -122.45},
				{Lat: 37.78, Lng: -122.42},
				{Lat: 37.76, Lng: -122.42},
				{Lat: 37.76, Lng: -122.45},
			}},
		},
		"equator": {GeoLoop: h3.GeoLoop{
			{Lat: 1, Lng: -1},
			{Lat: 1, Lng: 1},
			{Lat: -1, Lng: 1},
			{Lat: -1, Lng: -1},
		}},
		"transmeridian": {GeoLoop: h3.GeoLoop{
			{Lat: 10, Lng: 178},
			{Lat: 10, Lng: -178},
			{Lat: -10, Lng: -178},
			{Lat: -10, Lng: 178},
		}},
		"near_pentagon": {GeoLoop: h3.GeoLoop{
			{Lat: 64.7, Lng: 10.5},
			{Lat: 64.7, Lng: 11.5},
			{Lat: 63.7, Lng: 11.5},
			{Lat: 63.7, Lng: 10.5},
		}},
	}
}

// toGoPolygon converts a cgo GeoPolygon to the pure-Go type.
func toGoPolygon(polygon h3.GeoPolygon) h3go.GeoPolygon {
	out := h3go.GeoPolygon{GeoLoop: toGoLoop(polygon.GeoLoop)}
	for _, hole := range polygon.Holes {
		out.Holes = append(out.Holes, toGoLoop(hole))
	}

	return out
}

// toGoLoop converts a cgo GeoLoop to the pure-Go type.
func toGoLoop(loop h3.GeoLoop) h3go.GeoLoop {
	out := make(h3go.GeoLoop, len(loop))
	for i, point := range loop {
		out[i] = h3go.LatLng{Lat: point.Lat, Lng: point.Lng}
	}

	return out
}

// TestPolygonToCellsMatchesCgo asserts PolygonToCells matches the cgo reference
// as a set across several polygons and resolutions, including error parity.
func TestPolygonToCellsMatchesCgo(t *testing.T) {
	t.Parallel()

	for name, polygon := range regionPolygons() {
		for res := 4; res <= 8; res++ {
			want, wantErr := h3.PolygonToCells(polygon, res)
			got, gotErr := h3go.PolygonToCells(toGoPolygon(polygon), res)

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("PolygonToCells(%s, %d) error mismatch: cgo=%v h3go=%v", name, res, wantErr, gotErr)
			}

			if wantErr != nil {
				continue
			}

			assertSameCellSet(t, got, dropZeroCells(want), name)
		}
	}
}

// TestPolygonToCellsResolutionError asserts an out-of-range resolution fails the
// same way in both implementations.
func TestPolygonToCellsResolutionError(t *testing.T) {
	t.Parallel()

	polygon := regionPolygons()["sf_square"]

	for _, res := range []int{-1, 16} {
		_, wantErr := h3.PolygonToCells(polygon, res)
		_, gotErr := h3go.PolygonToCells(toGoPolygon(polygon), res)

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("PolygonToCells(res %d) error mismatch: cgo=%v h3go=%v", res, wantErr, gotErr)
		}
	}
}

// dropZeroCells removes the zero padding the cgo reference leaves in its output.
func dropZeroCells(cells []h3.Cell) []h3.Cell {
	out := cells[:0:0]
	for _, cell := range cells {
		if cell != 0 {
			out = append(out, cell)
		}
	}

	return out
}

// experimentalModes pairs each cgo containment mode with the pure-Go equivalent.
func experimentalModes() []struct {
	name string
	cgo  h3.ContainmentMode
	h3go h3go.ContainmentMode
} {
	return []struct {
		name string
		cgo  h3.ContainmentMode
		h3go h3go.ContainmentMode
	}{
		{"center", h3.ContainmentCenter, h3go.ContainmentCenter},
		{"full", h3.ContainmentFull, h3go.ContainmentFull},
		{"overlapping", h3.ContainmentOverlapping, h3go.ContainmentOverlapping},
		{"overlapping_bbox", h3.ContainmentOverlappingBbox, h3go.ContainmentOverlappingBbox},
	}
}

// TestPolygonToCellsExperimentalMatchesCgo asserts PolygonToCellsExperimental
// matches the cgo reference as a set across polygons, resolutions, and every
// containment mode.
func TestPolygonToCellsExperimentalMatchesCgo(t *testing.T) {
	t.Parallel()

	for name, polygon := range regionPolygons() {
		for _, mode := range experimentalModes() {
			for res := 4; res <= 7; res++ {
				want, wantErr := h3.PolygonToCellsExperimental(polygon, res, mode.cgo)
				got, gotErr := h3go.PolygonToCellsExperimental(toGoPolygon(polygon), res, mode.h3go)

				if !bothErr(wantErr, gotErr) {
					t.Fatalf("PolygonToCellsExperimental(%s, %s, %d) error mismatch: cgo=%v h3go=%v", name, mode.name, res, wantErr, gotErr)
				}

				if wantErr != nil {
					continue
				}

				assertSameCellSet(t, got, dropZeroCells(want), name+"/"+mode.name)
			}
		}
	}
}

// TestPolygonToCellsExperimentalErrors asserts resolution and mode errors match
// the cgo reference, and that a tight cell cap surfaces the same bounds error.
func TestPolygonToCellsExperimentalErrors(t *testing.T) {
	t.Parallel()

	polygon := regionPolygons()["sf_square"]

	for _, res := range []int{-1, 16} {
		_, wantErr := h3.PolygonToCellsExperimental(polygon, res, h3.ContainmentCenter)
		_, gotErr := h3go.PolygonToCellsExperimental(toGoPolygon(polygon), res, h3go.ContainmentCenter)

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("res %d error mismatch: cgo=%v h3go=%v", res, wantErr, gotErr)
		}
	}

	_, wantErr := h3.PolygonToCellsExperimental(polygon, 6, h3.ContainmentInvalid)
	_, gotErr := h3go.PolygonToCellsExperimental(toGoPolygon(polygon), 6, h3go.ContainmentInvalid)

	if !bothErr(wantErr, gotErr) {
		t.Fatalf("invalid mode error mismatch: cgo=%v h3go=%v", wantErr, gotErr)
	}

	_, wantErr = h3.PolygonToCellsExperimental(polygon, 7, h3.ContainmentCenter, 1)
	_, gotErr = h3go.PolygonToCellsExperimental(toGoPolygon(polygon), 7, h3go.ContainmentCenter, 1)

	if !bothErr(wantErr, gotErr) {
		t.Fatalf("bounds error mismatch: cgo=%v h3go=%v", wantErr, gotErr)
	}
}
