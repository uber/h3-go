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
	"errors"
	"math"
	"math/rand"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// assertSameCell verifies that the pure-Go implementation produces the same
// result (and the same error outcome) as the cgo-backed reference for a given
// coordinate and resolution.
func assertSameCell(t *testing.T, lat, lng float64, res int) {
	t.Helper()

	want, wErr := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, res)
	got, gErr := h3go.LatLngToCell(h3go.LatLng{Lat: lat, Lng: lng}, res)

	if !bothErr(wErr, gErr) {
		t.Fatalf("error mismatch at (%v, %v) res %d: cgo err=%v, h3go err=%v", lat, lng, res, wErr, gErr)
	}

	if uint64(got) != uint64(want) {
		t.Fatalf("cell mismatch at (%v, %v) res %d: cgo=%015x, h3go=%015x", lat, lng, res, uint64(want), uint64(got))
	}
}

// TestLatLngToCellMatchesCgoGrid sweeps a structured grid of coordinates,
// including poles and the antimeridian, across every resolution.
func TestLatLngToCellMatchesCgoGrid(t *testing.T) {
	t.Parallel()

	lats := []float64{-90, -89.9, -67.5, -45, -23.43, 0, 11.7, 37.7749, 45, 67.1509, 89.9, 90}
	lngs := []float64{-180, -179.9, -122.4194, -73.9857, -45, 0, 13.4, 100.5, 151.2093, 179.9, 180}

	for res := 0; res <= h3.MaxResolution; res++ {
		for _, lat := range lats {
			for _, lng := range lngs {
				assertSameCell(t, lat, lng, res)
			}
		}
	}
}

// TestLatLngToCellMatchesCgoRandom fuzzes a large, deterministic set of random
// coordinates and resolutions against the cgo reference.
func TestLatLngToCellMatchesCgoRandom(t *testing.T) {
	t.Parallel()

	const (
		seed       = 1
		iterations = 100000
		latSpan    = 180.0
		latOffset  = 90.0
		lngSpan    = 360.0
		lngOffset  = 180.0
		resCount   = h3.MaxResolution + 1
	)

	rng := rand.New(rand.NewSource(seed))

	for range iterations {
		lat := rng.Float64()*latSpan - latOffset
		lng := rng.Float64()*lngSpan - lngOffset
		res := rng.Intn(resCount)
		assertSameCell(t, lat, lng, res)
	}
}

// TestLatLngToCellMatchesCgoPentagons exercises the pentagon-specific rotation
// path for every one of the 12 pentagon base cells, across all resolutions.
func TestLatLngToCellMatchesCgoPentagons(t *testing.T) {
	t.Parallel()

	// The 12 base cells that are pentagons, one per icosahedron vertex.
	pentagonBaseCells := []int{4, 14, 24, 38, 49, 58, 63, 72, 83, 97, 107, 117}

	for _, bc := range pentagonBaseCells {
		c := res0Cell(bc)
		if !c.IsPentagon() {
			t.Fatalf("base cell %d res-0 index %015x is not a pentagon; test setup is wrong", bc, uint64(c))
		}

		ll, err := h3.CellToLatLng(h3.Cell(c))
		if err != nil {
			t.Fatalf("CellToLatLng(%015x): %v", uint64(c), err)
		}

		for res := 0; res <= h3.MaxResolution; res++ {
			assertSameCell(t, ll.Lat, ll.Lng, res)
		}
	}
}

// TestLatLngToCellErrorParity asserts both implementations reject the same
// out-of-domain inputs with the corresponding error.
func TestLatLngToCellErrorParity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveLat   float64
		giveLng   float64
		giveRes   int
		wantErr   error // expected cgo (h3) error
		wantGoErr error // expected pure-Go (h3go) error
	}{
		"resolution_below_zero": {giveLat: 0, giveLng: 0, giveRes: -1, wantErr: h3.ErrResolutionDomain, wantGoErr: h3go.ErrResolutionDomain},
		"resolution_above_max":  {giveLat: 0, giveLng: 0, giveRes: h3.MaxResolution + 1, wantErr: h3.ErrResolutionDomain, wantGoErr: h3go.ErrResolutionDomain},
		"lat_is_nan":            {giveLat: math.NaN(), giveLng: 0, giveRes: 5, wantErr: h3.ErrLatLngDomain, wantGoErr: h3go.ErrLatLngDomain},
		"lng_is_nan":            {giveLat: 0, giveLng: math.NaN(), giveRes: 5, wantErr: h3.ErrLatLngDomain, wantGoErr: h3go.ErrLatLngDomain},
		"lat_is_pos_inf":        {giveLat: math.Inf(1), giveLng: 0, giveRes: 5, wantErr: h3.ErrLatLngDomain, wantGoErr: h3go.ErrLatLngDomain},
		"lng_is_neg_inf":        {giveLat: 0, giveLng: math.Inf(-1), giveRes: 5, wantErr: h3.ErrLatLngDomain, wantGoErr: h3go.ErrLatLngDomain},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, wErr := h3.LatLngToCell(h3.LatLng{Lat: tt.giveLat, Lng: tt.giveLng}, tt.giveRes)
			if !errors.Is(wErr, tt.wantErr) {
				t.Fatalf("cgo error = %v, want %v", wErr, tt.wantErr)
			}

			_, gErr := h3go.LatLngToCell(h3go.LatLng{Lat: tt.giveLat, Lng: tt.giveLng}, tt.giveRes)
			if !errors.Is(gErr, tt.wantGoErr) {
				t.Fatalf("h3go error = %v, want %v", gErr, tt.wantGoErr)
			}
		})
	}
}
