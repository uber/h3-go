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
	"math"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// measureRelTolerance is the maximum allowed relative difference between the
// pure-Go and cgo measure results. The two implementations run the same formulas,
// but Go's math package and C's libm differ by a few ULPs in the transcendental
// functions; that fixed-magnitude error grows in relative terms as cells shrink,
// reaching ~1e-6 at the finest resolution — so we compare to a small relative
// tolerance rather than bit-for-bit. measureAbsFloor handles the want==0 case
// (the distance from a point to itself).
const (
	measureRelTolerance = 1e-5
	measureAbsFloor     = 1e-9
)

// TestCellAreaMatchesCgo asserts the pure-Go cell area matches the cgo-backed
// reference in radians², km², and m² for every cell in the shared corpus.
func TestCellAreaMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, ref := range referenceCorpus(t) {
		goCell := h3goCell(ref)

		wantRads2, wErr := h3.CellAreaRads2(ref)
		gotRads2, gErr := h3go.CellAreaRads2(goCell)

		if !bothErr(wErr, gErr) {
			t.Fatalf("CellAreaRads2(%015x) error mismatch: cgo=%v h3go=%v", uint64(ref), wErr, gErr)
		}

		assertRelClose(t, gotRads2, wantRads2, ref, "rads2")

		wantKm2, _ := h3.CellAreaKm2(ref)
		gotKm2, _ := h3go.CellAreaKm2(goCell)
		assertRelClose(t, gotKm2, wantKm2, ref, "km2")

		wantM2, _ := h3.CellAreaM2(ref)
		gotM2, _ := h3go.CellAreaM2(goCell)
		assertRelClose(t, gotM2, wantM2, ref, "m2")
	}
}

// TestGreatCircleDistanceMatchesCgo asserts the pure-Go haversine distance
// matches the cgo-backed reference (radians/km/m) for every pair of corpus
// points.
func TestGreatCircleDistanceMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, a := range corpusPoints {
		for _, b := range corpusPoints {
			goA := h3go.LatLng{Lat: a.Lat, Lng: a.Lng}
			goB := h3go.LatLng{Lat: b.Lat, Lng: b.Lng}

			assertRelClose(t, h3go.GreatCircleDistanceRads(goA, goB), h3.GreatCircleDistanceRads(a, b), 0, "dist rads")
			assertRelClose(t, h3go.GreatCircleDistanceKm(goA, goB), h3.GreatCircleDistanceKm(a, b), 0, "dist km")
			assertRelClose(t, h3go.GreatCircleDistanceM(goA, goB), h3.GreatCircleDistanceM(a, b), 0, "dist m")
		}
	}
}

// TestHexagonAvgMatchesCgo asserts the average area and edge-length getters match
// the cgo-backed reference for every resolution, including the out-of-range
// error.
func TestHexagonAvgMatchesCgo(t *testing.T) {
	t.Parallel()

	for res := -1; res <= h3.MaxResolution+1; res++ {
		areaKmWant, areaKmErr := h3.HexagonAreaAvgKm2(res)
		areaKmGot, areaKmGotErr := h3go.HexagonAreaAvgKm2(res)

		if !bothErr(areaKmErr, areaKmGotErr) {
			t.Fatalf("HexagonAreaAvgKm2(%d) error mismatch: cgo=%v h3go=%v", res, areaKmErr, areaKmGotErr)
		}

		assertRelClose(t, areaKmGot, areaKmWant, 0, "areaAvgKm2")

		areaMWant, _ := h3.HexagonAreaAvgM2(res)
		areaMGot, _ := h3go.HexagonAreaAvgM2(res)
		assertRelClose(t, areaMGot, areaMWant, 0, "areaAvgM2")

		lenKmWant, _ := h3.HexagonEdgeLengthAvgKm(res)
		lenKmGot, _ := h3go.HexagonEdgeLengthAvgKm(res)
		assertRelClose(t, lenKmGot, lenKmWant, 0, "edgeLenAvgKm")

		lenMWant, _ := h3.HexagonEdgeLengthAvgM(res)
		lenMGot, _ := h3go.HexagonEdgeLengthAvgM(res)
		assertRelClose(t, lenMGot, lenMWant, 0, "edgeLenAvgM")
	}
}

// assertRelClose fails if got and want differ by more than measureRelTolerance
// relative to want (or by more than measureAbsFloor when want is zero). cell is
// included in the failure message when non-zero.
func assertRelClose(t *testing.T, got, want float64, cell h3.Cell, label string) {
	t.Helper()

	diff := math.Abs(got - want)

	if want == 0 {
		if diff > measureAbsFloor {
			t.Fatalf("%s (cell %015x): got %.17g, want %.17g", label, uint64(cell), got, want)
		}

		return
	}

	if diff/math.Abs(want) > measureRelTolerance {
		t.Fatalf("%s (cell %015x): got %.17g, want %.17g", label, uint64(cell), got, want)
	}
}
