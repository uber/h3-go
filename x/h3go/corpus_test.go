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

import "testing"

// corpus builds a deterministic set of cells spanning base cells, hexagons, and
// pentagons across every resolution, for the white-box property and round-trip
// tests that exercise the public API directly.
func corpus(t *testing.T) []Cell {
	t.Helper()

	var cells []Cell

	res0, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	cells = append(cells, res0...)

	lats := []float64{-89.9, -45, -23.43, 0, 11.7, 37.7749, 45, 67.1509, 89.9}
	lngs := []float64{-179.9, -122.4194, -45, 0, 13.4, 100.5, 151.2093, 179.9}

	for res := 0; res <= MaxResolution; res++ {
		pents, err := Pentagons(res)
		if err != nil {
			t.Fatalf("Pentagons(%d): %v", res, err)
		}

		cells = append(cells, pents...)

		for _, lat := range lats {
			for _, lng := range lngs {
				c, err := LatLngToCell(LatLng{Lat: lat, Lng: lng}, res)
				if err != nil {
					t.Fatalf("LatLngToCell(%v, %v, %d): %v", lat, lng, res, err)
				}

				cells = append(cells, c)
			}
		}
	}

	return cells
}
