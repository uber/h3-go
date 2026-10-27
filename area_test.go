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
	"math"
	"strconv"
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

// TestCellAreaNullIslandTable ports the testH3CellArea.c specific_cell_area
// regression: the exact area in km² of the cell containing (0, 0) at each
// resolution, to a tight absolute tolerance.
func TestCellAreaNullIslandTable(t *testing.T) {
	t.Parallel()

	areasKm2 := []float64{
		2.562182162955496e+06, 4.476842017201860e+05, 6.596162242711056e+04,
		9.228872919002590e+03, 1.318694490797110e+03, 1.879593512281298e+02,
		2.687164354763186e+01, 3.840848847060638e+00, 5.486939641329893e-01,
		7.838600808637444e-02, 1.119834221989390e-02, 1.599777169186614e-03,
		2.285390931423380e-04, 3.264850232091780e-05, 4.664070326136774e-06,
		6.662957615868888e-07,
	}

	// The C fixture checks res 0..MAX_H3_RES-1.
	for res := 0; res < MaxResolution; res++ {
		cell, err := LatLngToCell(LatLng{Lat: 0, Lng: 0}, res)
		if err != nil {
			t.Fatalf("LatLngToCell(res %d): %v", res, err)
		}

		area, err := CellAreaKm2(cell)
		if err != nil {
			t.Fatalf("CellAreaKm2(res %d): %v", res, err)
		}

		if math.Abs(area-areasKm2[res]) >= 1e-8 {
			t.Fatalf("CellAreaKm2(res %d): got %.15e, want %.15e", res, area, areasKm2[res])
		}
	}
}

// TestCellAreaInvalidBaseCell covers the error branch of CellAreaRads2/Km2/M2,
// reached when the boundary cannot be computed for an out-of-range base cell.
func TestCellAreaInvalidBaseCell(t *testing.T) {
	t.Parallel()

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(NumBaseCells)<<baseCellOffset

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

func TestBoundaryAreaEmpty(t *testing.T) {
	t.Parallel()

	if got := (CellBoundary{}).areaRads2(); got != 0 {
		t.Fatalf("empty boundary area = %v, want 0", got)
	}
}

// TestCellAreaAntimeridian checks that a cell straddling the antimeridian has
// the same area as a neighbouring cell at the same resolution that does not.
// Subtracting raw longitudes across ±180° loses the tiny true step to the
// rounding of a value near 360°, which made straddling cells too small.
func TestCellAreaAntimeridian(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveStraddling string
		giveNeighbor   string
	}{
		"res13": {"8d056898db628bf", "8d056898db62d7f"},
		"res14": {"8e32b605c51e067", "8e32b605c51e15f"},
		"res15": {"8f7f809893b5b9d", "8f7f809893b5b9c"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			straddling, err := CellAreaRads2(CellFromString(tt.giveStraddling))
			if err != nil {
				t.Fatalf("CellAreaRads2(%s): %v", tt.giveStraddling, err)
			}

			neighbor, err := CellAreaRads2(CellFromString(tt.giveNeighbor))
			if err != nil {
				t.Fatalf("CellAreaRads2(%s): %v", tt.giveNeighbor, err)
			}

			const tol = 1e-5
			if rel := math.Abs(straddling-neighbor) / neighbor; rel > tol {
				t.Fatalf("straddling area %.17g, neighbor %.17g: relative difference %.2e > %.0e",
					straddling, neighbor, rel, tol)
			}
		})
	}
}

// TestCellAreaAntimeridianRotation checks, at every resolution, that a cell on
// the antimeridian keeps its area when its boundary is rotated 180° about the
// polar axis, which moves it onto the prime meridian where no edge crosses
// ±180°. A rotation cannot change an area, so no reference values are needed.
func TestCellAreaAntimeridianRotation(t *testing.T) {
	t.Parallel()

	for res := 0; res <= MaxResolution; res++ {
		t.Run(strconv.Itoa(res), func(t *testing.T) {
			t.Parallel()

			for _, lat := range []float64{-60, 0, 45} {
				cell, err := LatLngToCell(NewLatLng(lat, 180), res)
				if err != nil {
					t.Fatalf("LatLngToCell: %v", err)
				}

				boundary, err := cell.Boundary()
				if err != nil {
					t.Fatalf("Boundary: %v", err)
				}

				rotated := make(CellBoundary, len(boundary))
				for i := range boundary {
					rotated[i] = LatLng{Lat: boundary[i].Lat, Lng: rotateHalfTurn(boundary[i].Lng)}
				}

				const tol = 1e-6

				got, want := boundary.areaRads2(), rotated.areaRads2()
				if rel := math.Abs(got-want) / want; rel > tol {
					t.Fatalf("%s: area %.17g, rotated %.17g: relative difference %.2e > %.0e",
						cell, got, want, rel, tol)
				}
			}
		})
	}
}

// TestBoundaryAreaAntimeridianTriangle checks that a small triangle across
// ±180° has the same area as the same triangle across 0°. A triangle rather
// than a rectangle: a rectangle's two crossing edges have identical longitude
// differences, so their rounding errors cancel and hide the defect.
func TestBoundaryAreaAntimeridianTriangle(t *testing.T) {
	t.Parallel()

	const halfWidth = 1e-6

	prime := CellBoundary{
		{Lat: 0, Lng: -halfWidth},
		{Lat: 0, Lng: halfWidth},
		{Lat: 2 * halfWidth, Lng: 0},
	}

	anti := make(CellBoundary, len(prime))
	for i := range prime {
		anti[i] = LatLng{Lat: prime[i].Lat, Lng: rotateHalfTurn(prime[i].Lng)}
	}

	const tol = 1e-9

	got, want := anti.areaRads2(), prime.areaRads2()
	if rel := math.Abs(got-want) / want; rel > tol {
		t.Fatalf("antimeridian triangle area %.17g, prime meridian %.17g: relative difference %.2e > %.0e",
			got, want, rel, tol)
	}
}

// rotateHalfTurn returns the longitude 180° east of lng, wrapped to (-180, 180].
func rotateHalfTurn(lng float64) float64 {
	rotated := lng + piDeg
	if rotated > piDeg {
		rotated -= twoPiDeg
	}

	return rotated
}

// assertRelClose fails if got and want differ by more than a small relative
// tolerance, suitable for comparing values that span many orders of magnitude.
func assertRelClose(t *testing.T, got, want float64, label string) {
	t.Helper()

	// Every comparison against NaN is false, so test for it explicitly or a
	// NaN result would pass.
	const tol = 1e-9
	if math.IsNaN(got) || math.Abs(got-want) > tol*math.Max(1, math.Abs(want)) {
		t.Fatalf("%s = %.17g, want %.17g", label, got, want)
	}
}
