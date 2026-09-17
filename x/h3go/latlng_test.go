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
	"math/rand"
	"testing"
)

// TestLatLngToCellKnownValues pins the pure-Go implementation to well-known H3
// indexes documented elsewhere in the repository.
func TestLatLngToCellKnownValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveLat  float64
		giveLng  float64
		giveRes  int
		wantCell uint64
	}{
		"san_francisco_res_9": {giveLat: 37.775938728915946, giveLng: -122.41795063018799, giveRes: 9, wantCell: 0x8928308280fffff},
		"chukchi_sea_res_5":   {giveLat: 67.1509268640, giveLng: -168.3908885810, giveRes: 5, wantCell: 0x850dab63fffffff},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := LatLngToCell(LatLng{Lat: tt.giveLat, Lng: tt.giveLng}, tt.giveRes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if uint64(got) != tt.wantCell {
				t.Fatalf("got %015x, want %015x", uint64(got), tt.wantCell)
			}
		})
	}
}

// TestLatLngToCellErrors covers the input-validation error paths.
func TestLatLngToCellErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveLat float64
		giveLng float64
		giveRes int
		wantErr error
	}{
		"resolution_below_zero": {giveLat: 0, giveLng: 0, giveRes: -1, wantErr: ErrResolutionDomain},
		"resolution_above_max":  {giveLat: 0, giveLng: 0, giveRes: MaxResolution + 1, wantErr: ErrResolutionDomain},
		"lat_is_nan":            {giveLat: math.NaN(), giveLng: 0, giveRes: 5, wantErr: ErrLatLngDomain},
		"lng_is_nan":            {giveLat: 0, giveLng: math.NaN(), giveRes: 5, wantErr: ErrLatLngDomain},
		"lat_is_pos_inf":        {giveLat: math.Inf(1), giveLng: 0, giveRes: 5, wantErr: ErrLatLngDomain},
		"lng_is_neg_inf":        {giveLat: 0, giveLng: math.Inf(-1), giveRes: 5, wantErr: ErrLatLngDomain},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := LatLngToCell(LatLng{Lat: tt.giveLat, Lng: tt.giveLng}, tt.giveRes); !errors.Is(err, tt.wantErr) {
				t.Fatalf("LatLngToCell: got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestLatLngToCellProjectionSweep exercises the full projection pipeline over a
// structured grid (poles and antimeridian) plus a large deterministic random
// sweep, across every resolution. Each result must be a valid cell at the
// requested resolution. The random portion is what reliably reaches the
// pentagon rotation paths in the projection.
func TestLatLngToCellProjectionSweep(t *testing.T) {
	t.Parallel()

	lats := []float64{-90, -89.9, -67.5, -45, -23.43, 0, 11.7, 37.7749, 45, 67.1509, 89.9, 90}
	lngs := []float64{-180, -179.9, -122.4194, -73.9857, -45, 0, 13.4, 100.5, 151.2093, 179.9, 180}

	for res := 0; res <= MaxResolution; res++ {
		for _, lat := range lats {
			for _, lng := range lngs {
				assertValidAtRes(t, lat, lng, res)
			}
		}
	}

	const (
		seed       = 1
		iterations = 100000
		latSpan    = 180.0
		latOffset  = 90.0
		lngSpan    = 360.0
		lngOffset  = 180.0
		resCount   = MaxResolution + 1
	)

	rng := rand.New(rand.NewSource(seed))
	for range iterations {
		lat := rng.Float64()*latSpan - latOffset
		lng := rng.Float64()*lngSpan - lngOffset
		assertValidAtRes(t, lat, lng, rng.Intn(resCount))
	}
}

// TestCellReverseMethods covers the Cell.LatLng and Cell.Boundary method
// wrappers, which delegate to CellToLatLng and CellToBoundary.
func TestCellReverseMethods(t *testing.T) {
	t.Parallel()

	cell, err := LatLngToCell(LatLng{Lat: 37.7749, Lng: -122.4194}, 9)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	ll, err := cell.LatLng()
	if err != nil {
		t.Fatalf("Cell.LatLng(%015x): %v", uint64(cell), err)
	}

	if got, err := LatLngToCell(ll, cell.Resolution()); err != nil || got != cell {
		t.Fatalf("Cell.LatLng round trip: got %015x err %v, want %015x", uint64(got), err, uint64(cell))
	}

	boundary, err := cell.Boundary()
	if err != nil {
		t.Fatalf("Cell.Boundary(%015x): %v", uint64(cell), err)
	}

	if len(boundary) != numHexVerts {
		t.Fatalf("Cell.Boundary(%015x): %d vertices, want %d", uint64(cell), len(boundary), numHexVerts)
	}
}

// assertValidAtRes fails unless LatLngToCell returns a valid cell at res.
func assertValidAtRes(t *testing.T, lat, lng float64, res int) {
	t.Helper()

	c, err := LatLngToCell(LatLng{Lat: lat, Lng: lng}, res)
	if err != nil {
		t.Fatalf("LatLngToCell(%v, %v, %d): %v", lat, lng, res, err)
	}

	if !c.IsValid() {
		t.Fatalf("LatLngToCell(%v, %v, %d) = %015x: not valid", lat, lng, res, uint64(c))
	}

	if c.Resolution() != res {
		t.Fatalf("LatLngToCell(%v, %v, %d) = %015x: resolution %d", lat, lng, res, uint64(c), c.Resolution())
	}
}

// TestCellToLatLngRoundTrip asserts the strong invariant that re-encoding a
// cell's own center point at the same resolution yields the original cell. This
// exercises the whole reverse projection (h3ToFaceIjk, faceIjkToVec3,
// hex2dToVec3) against the forward pipeline over a rich corpus of base cells,
// hexagons, and pentagons, including pentagon descendants that reach the
// pentagon rotation and overage paths.
func TestCellToLatLngRoundTrip(t *testing.T) {
	t.Parallel()

	for _, c := range reverseCorpus(t) {
		ll, err := CellToLatLng(c)
		if err != nil {
			t.Fatalf("CellToLatLng(%015x): %v", uint64(c), err)
		}

		if math.IsNaN(ll.Lat) || math.IsNaN(ll.Lng) ||
			ll.Lat < -90 || ll.Lat > 90 || ll.Lng < -180 || ll.Lng > 180 {
			t.Fatalf("CellToLatLng(%015x) = %+v: out of range", uint64(c), ll)
		}

		got, err := LatLngToCell(ll, c.Resolution())
		if err != nil {
			t.Fatalf("LatLngToCell(%+v, %d): %v", ll, c.Resolution(), err)
		}

		if got != c {
			t.Fatalf("round trip for %015x: center %+v re-encoded to %015x", uint64(c), ll, uint64(got))
		}
	}
}

// TestCellToLatLngInvalidBaseCell covers the invalid-base-cell error branch of
// the reverse projection, which is unreachable from a validly constructed cell.
func TestCellToLatLngInvalidBaseCell(t *testing.T) {
	t.Parallel()

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(NumBaseCells)<<baseCellOffset

	if _, err := CellToLatLng(bad); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("CellToLatLng(%015x): got %v, want %v", uint64(bad), err, ErrCellInvalid)
	}
}

// TestCellToLatLngInvalidIndex is the regression for an all-ones index (whose
// base cell field is out of range), which must be rejected.
func TestCellToLatLngInvalidIndex(t *testing.T) {
	t.Parallel()

	if _, err := CellToLatLng(Cell(0x7fffffffffffffff)); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("CellToLatLng(0x7fffffffffffffff): got %v, want %v", err, ErrCellInvalid)
	}
}

// TestGreatCircleDistance checks the haversine distance in radians, kilometers,
// and meters against a known city pair, plus the zero-distance and symmetry
// properties.
func TestGreatCircleDistance(t *testing.T) {
	t.Parallel()

	sf := LatLng{Lat: 37.7749, Lng: -122.4194}
	ny := LatLng{Lat: 40.7128, Lng: -74.0060}

	t.Run("known_pair", func(t *testing.T) {
		t.Parallel()

		assertRelClose(t, GreatCircleDistanceRads(sf, ny), 0.64810644562192898, "rads")
		assertRelClose(t, GreatCircleDistanceKm(sf, ny), 4129.090819109696, "km")
		assertRelClose(t, GreatCircleDistanceM(sf, ny), 4129090.819109696, "m")
	})

	t.Run("zero_distance", func(t *testing.T) {
		t.Parallel()

		if got := GreatCircleDistanceRads(sf, sf); math.Abs(got) > 1e-12 {
			t.Fatalf("GreatCircleDistanceRads(sf, sf) = %v, want ~0", got)
		}
	})

	t.Run("symmetric", func(t *testing.T) {
		t.Parallel()

		if a, b := GreatCircleDistanceRads(sf, ny), GreatCircleDistanceRads(ny, sf); a != b {
			t.Fatalf("distance not symmetric: %v vs %v", a, b)
		}
	})
}

// TestHexagonAreaAvg checks the average-area getters: a known resolution-0 value,
// strictly decreasing area with finer resolution, and the resolution-domain error.
func TestHexagonAreaAvg(t *testing.T) {
	t.Parallel()

	t.Run("known_and_monotonic", func(t *testing.T) {
		t.Parallel()

		km0, err := HexagonAreaAvgKm2(0)
		if err != nil {
			t.Fatalf("HexagonAreaAvgKm2(0): %v", err)
		}

		assertRelClose(t, km0, 4.357449416078383e+06, "area km2 res0")

		prevKm, prevM := math.Inf(1), math.Inf(1)

		for res := 0; res <= MaxResolution; res++ {
			km, err := HexagonAreaAvgKm2(res)
			if err != nil {
				t.Fatalf("HexagonAreaAvgKm2(%d): %v", res, err)
			}

			m2, err := HexagonAreaAvgM2(res)
			if err != nil {
				t.Fatalf("HexagonAreaAvgM2(%d): %v", res, err)
			}

			if km >= prevKm || m2 >= prevM {
				t.Fatalf("res %d: area not decreasing (km=%v prevKm=%v)", res, km, prevKm)
			}

			assertRelClose(t, m2, km*1e6, "m2 vs km2*1e6")
			prevKm, prevM = km, m2
		}
	})

	t.Run("out_of_range", func(t *testing.T) {
		t.Parallel()

		for _, res := range []int{-1, MaxResolution + 1} {
			if _, err := HexagonAreaAvgKm2(res); !errors.Is(err, ErrResolutionDomain) {
				t.Fatalf("HexagonAreaAvgKm2(%d): got %v, want %v", res, err, ErrResolutionDomain)
			}

			if _, err := HexagonAreaAvgM2(res); !errors.Is(err, ErrResolutionDomain) {
				t.Fatalf("HexagonAreaAvgM2(%d): got %v, want %v", res, err, ErrResolutionDomain)
			}
		}
	})
}

// TestHexagonEdgeLengthAvg checks the average-edge-length getters: a known
// resolution-0 value, strictly decreasing length with finer resolution, and the
// resolution-domain error.
func TestHexagonEdgeLengthAvg(t *testing.T) {
	t.Parallel()

	t.Run("known_and_monotonic", func(t *testing.T) {
		t.Parallel()

		km0, err := HexagonEdgeLengthAvgKm(0)
		if err != nil {
			t.Fatalf("HexagonEdgeLengthAvgKm(0): %v", err)
		}

		assertRelClose(t, km0, 1281.256011, "edge km res0")

		prevKm, prevM := math.Inf(1), math.Inf(1)

		for res := 0; res <= MaxResolution; res++ {
			km, err := HexagonEdgeLengthAvgKm(res)
			if err != nil {
				t.Fatalf("HexagonEdgeLengthAvgKm(%d): %v", res, err)
			}

			m, err := HexagonEdgeLengthAvgM(res)
			if err != nil {
				t.Fatalf("HexagonEdgeLengthAvgM(%d): %v", res, err)
			}

			if km >= prevKm || m >= prevM {
				t.Fatalf("res %d: length not decreasing (km=%v prevKm=%v)", res, km, prevKm)
			}

			prevKm, prevM = km, m
		}
	})

	t.Run("out_of_range", func(t *testing.T) {
		t.Parallel()

		for _, res := range []int{-1, MaxResolution + 1} {
			if _, err := HexagonEdgeLengthAvgKm(res); !errors.Is(err, ErrResolutionDomain) {
				t.Fatalf("HexagonEdgeLengthAvgKm(%d): got %v, want %v", res, err, ErrResolutionDomain)
			}

			if _, err := HexagonEdgeLengthAvgM(res); !errors.Is(err, ErrResolutionDomain) {
				t.Fatalf("HexagonEdgeLengthAvgM(%d): got %v, want %v", res, err, ErrResolutionDomain)
			}
		}
	})
}

// TestNewLatLngAndCell covers the NewLatLng constructor and the LatLng.Cell
// method form of LatLngToCell.
func TestNewLatLngAndCell(t *testing.T) {
	t.Parallel()

	ll := NewLatLng(37.775, -122.418)
	if ll.Lat != 37.775 || ll.Lng != -122.418 {
		t.Fatalf("NewLatLng: got %+v", ll)
	}

	viaMethod, err := ll.Cell(9)
	if err != nil {
		t.Fatalf("LatLng.Cell: %v", err)
	}

	viaFunc, err := LatLngToCell(ll, 9)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	if viaMethod != viaFunc {
		t.Fatalf("LatLng.Cell %015x != LatLngToCell %015x", uint64(viaMethod), uint64(viaFunc))
	}

	if _, err := ll.Cell(-1); !errors.Is(err, ErrResolutionDomain) {
		t.Fatalf("LatLng.Cell(-1): got %v, want ErrResolutionDomain", err)
	}
}

// TestLatLngString covers the LatLng stringer formatting.
func TestLatLngString(t *testing.T) {
	t.Parallel()

	if got := NewLatLng(37.775, -122.418).String(); got != "(37.77500, -122.41800)" {
		t.Fatalf("LatLng.String: got %q", got)
	}
}

// TestLatLngToCellString covers the coordinate-to-cell-string helper and its
// error path.
func TestLatLngToCellString(t *testing.T) {
	t.Parallel()

	got, err := LatLngToCellString(37.775, -122.418, 9)
	if err != nil {
		t.Fatalf("LatLngToCellString: %v", err)
	}

	want, err := LatLngToCell(NewLatLng(37.775, -122.418), 9)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	if got != want.String() {
		t.Fatalf("LatLngToCellString: got %q, want %q", got, want.String())
	}

	if _, err := LatLngToCellString(0, 0, 16); !errors.Is(err, ErrResolutionDomain) {
		t.Fatalf("LatLngToCellString(res 16): got %v, want ErrResolutionDomain", err)
	}
}

// TestLatLngToCellExtremeCoordinates ports the testH3Index.c
// latLngToCellExtremeCoordinates regression: absurd but finite coordinates must
// not crash and must yield a cell without error.
func TestLatLngToCellExtremeCoordinates(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveLatLng LatLng
		giveRes    int
	}{
		"huge_lng":      {giveLatLng: LatLng{Lat: 0, Lng: 1e45}, giveRes: 14},
		"huge_lat_lng":  {giveLatLng: LatLng{Lat: 1e46, Lng: 1e45}, giveRes: 15},
		"huge_negative": {giveLatLng: LatLng{Lat: 2, Lng: -3e39}, giveRes: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := LatLngToCell(tt.giveLatLng, tt.giveRes); err != nil {
				t.Fatalf("LatLngToCell: got %v, want success", err)
			}
		})
	}
}

// TestGreatCircleDistanceWrappedLongitude ports the testLatLng.c
// distanceRads_wrappedLongitude regression: a longitude difference greater than
// 180° (here -270°) measures the short way around, π/2.
func TestGreatCircleDistanceWrappedLongitude(t *testing.T) {
	t.Parallel()

	negativeLongitude := LatLng{Lat: 0, Lng: -270}
	zero := LatLng{Lat: 0, Lng: 0}

	const wantRads = math.Pi / 2

	// Matches the C reference's EPSILON_RAD (EPSILON_DEG 1e-9, in radians).
	const epsilonRad = 1e-9 * DegsToRads

	if got := GreatCircleDistanceRads(negativeLongitude, zero); math.Abs(got-wantRads) >= epsilonRad {
		t.Fatalf("GreatCircleDistanceRads: got %v, want %v", got, wantRads)
	}

	if got := GreatCircleDistanceRads(zero, negativeLongitude); math.Abs(got-wantRads) >= epsilonRad {
		t.Fatalf("GreatCircleDistanceRads (swapped): got %v, want %v", got, wantRads)
	}
}
