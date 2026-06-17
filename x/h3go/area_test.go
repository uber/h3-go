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
	"math"
	"testing"
)

// TestCellAreaKnownValues checks CellAreaRads2/Km2/M2 against reference values
// for hexagons and pentagons across several resolutions, including a base cell
// and cells whose boundaries span icosahedron faces.
func TestCellAreaKnownValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveCell  string
		wantRads2 float64
		wantKm2   float64
		wantM2    float64
	}{
		"hex_res9":  {"8928308280fffff", 2.6952182709906241e-09, 0.1093981886467751, 109398.1886467751},
		"base_res0": {"8001fffffffffff", 0.10116268528089556, 4106166.3344639186, 4106166334463.9185},
		"hex_res5":  {"85283473fffffff", 6.5310250106417195e-06, 265.09255812828178, 265092558.1282818},
		"pent_res5": {"851c0003fffffff", 3.1482243104266963e-06, 127.78558260805931, 127785582.60805932},
		"pent_res0": {"8009fffffffffff", 0.06312389871006796, 2562182.1629554993, 2562182162955.499},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c := CellFromString(tt.giveCell)

			rads2, err := CellAreaRads2(c)
			if err != nil {
				t.Fatalf("CellAreaRads2: %v", err)
			}

			assertRelClose(t, rads2, tt.wantRads2, "rads2")

			km2, err := CellAreaKm2(c)
			if err != nil {
				t.Fatalf("CellAreaKm2: %v", err)
			}

			assertRelClose(t, km2, tt.wantKm2, "km2")

			m2, err := CellAreaM2(c)
			if err != nil {
				t.Fatalf("CellAreaM2: %v", err)
			}

			assertRelClose(t, m2, tt.wantM2, "m2")
		})
	}
}

// TestCellAreaRes0SumsToSphere checks that the areas of all resolution-0 cells
// sum to the area of the unit sphere (4π), a global consistency property.
func TestCellAreaRes0SumsToSphere(t *testing.T) {
	t.Parallel()

	res0, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	var sum float64

	for _, c := range res0 {
		area, err := CellAreaRads2(c)
		if err != nil {
			t.Fatalf("CellAreaRads2(%015x): %v", uint64(c), err)
		}

		if area <= 0 {
			t.Fatalf("CellAreaRads2(%015x) = %v, want positive", uint64(c), area)
		}

		sum += area
	}

	assertRelClose(t, sum, 4*math.Pi, "res0 area sum")
}

// TestCellAreaInvalidBaseCell covers the error branch of CellAreaRads2/Km2/M2,
// reached when the boundary cannot be computed for an out-of-range base cell.
func TestCellAreaInvalidBaseCell(t *testing.T) {
	t.Parallel()

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(numBaseCells)<<baseCellOffset

	if _, err := CellAreaRads2(bad); err == nil {
		t.Fatal("CellAreaRads2: got nil error, want failure")
	}

	if _, err := CellAreaKm2(bad); err == nil {
		t.Fatal("CellAreaKm2: got nil error, want failure")
	}

	if _, err := CellAreaM2(bad); err == nil {
		t.Fatal("CellAreaM2: got nil error, want failure")
	}
}

// TestBoundaryAreaClockwiseNormalizes covers the clockwise-loop branch of
// areaRads2: a loop wound clockwise has negative signed area and must be
// normalized into [0, 4π] by adding the full-sphere area. (Real cell boundaries
// are wound counterclockwise, so only a reversed loop reaches this branch.)
func TestBoundaryAreaClockwiseNormalizes(t *testing.T) {
	t.Parallel()

	ccw := CellBoundary{
		{Lat: 0, Lng: 0},
		{Lat: 0, Lng: 1},
		{Lat: 1, Lng: 1},
		{Lat: 1, Lng: 0},
	}

	cw := CellBoundary{ccw[3], ccw[2], ccw[1], ccw[0]}

	small := ccw.areaRads2()
	large := cw.areaRads2()

	if small <= 0 || small >= 2*math.Pi {
		t.Fatalf("ccw area = %v, want a small positive value", small)
	}

	assertRelClose(t, small+large, 4*math.Pi, "ccw + cw area")
}

// assertRelClose fails if got and want differ by more than a small relative
// tolerance, suitable for comparing values that span many orders of magnitude.
func assertRelClose(t *testing.T, got, want float64, label string) {
	t.Helper()

	const tol = 1e-9
	if math.Abs(got-want) > tol*math.Max(1, math.Abs(want)) {
		t.Fatalf("%s = %.17g, want %.17g", label, got, want)
	}
}
