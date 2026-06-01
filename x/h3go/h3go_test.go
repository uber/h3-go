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

package h3go_test

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

	if (wErr == nil) != (gErr == nil) {
		t.Fatalf("error mismatch at (%v, %v) res %d: cgo err=%v, h3go err=%v",
			lat, lng, res, wErr, gErr)
	}

	if uint64(got) != uint64(want) {
		t.Fatalf("cell mismatch at (%v, %v) res %d: cgo=%015x, h3go=%015x",
			lat, lng, res, uint64(want), uint64(got))
	}
}

// TestLatLngToCellKnownValues pins the pure-Go implementation to a couple of
// well-known H3 indexes documented elsewhere in the repository.
func TestLatLngToCellKnownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lat  float64
		lng  float64
		res  int
		want uint64
	}{
		{"san francisco res 9", 37.775938728915946, -122.41795063018799, 9, 0x8928308280fffff},
		{"chukchi sea res 5", 67.1509268640, -168.3908885810, 5, 0x850dab63fffffff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := h3go.LatLngToCell(h3go.LatLng{Lat: tt.lat, Lng: tt.lng}, tt.res)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if uint64(got) != tt.want {
				t.Fatalf("got %015x, want %015x", uint64(got), tt.want)
			}
		})
	}
}

// TestLatLngToCellMatchesCgoGrid sweeps a structured grid of coordinates,
// including poles and the antimeridian, across every resolution and asserts
// the pure-Go output matches the cgo reference exactly.
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
			t.Fatalf("base cell %d res-0 index %015x is not a pentagon; test setup is wrong",
				bc, uint64(c))
		}

		ll, err := h3.CellToLatLng(c)
		if err != nil {
			t.Fatalf("CellToLatLng(%015x): %v", uint64(c), err)
		}

		for res := 0; res <= h3.MaxResolution; res++ {
			assertSameCell(t, ll.Lat, ll.Lng, res)
		}
	}
}

// TestLatLngToCellErrorParity asserts that both implementations reject the same
// out-of-domain inputs with the corresponding error.
func TestLatLngToCellErrorParity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lat       float64
		lng       float64
		res       int
		wantErr   error // expected cgo (h3) error
		wantGoErr error // expected pure-Go (h3go) error
	}{
		{"resolution below zero", 0, 0, -1, h3.ErrResolutionDomain, h3go.ErrResolutionDomain},
		{"resolution above max", 0, 0, h3.MaxResolution + 1, h3.ErrResolutionDomain, h3go.ErrResolutionDomain},
		{"lat is NaN", math.NaN(), 0, 5, h3.ErrLatLngDomain, h3go.ErrLatLngDomain},
		{"lng is NaN", 0, math.NaN(), 5, h3.ErrLatLngDomain, h3go.ErrLatLngDomain},
		{"lat is +Inf", math.Inf(1), 0, 5, h3.ErrLatLngDomain, h3go.ErrLatLngDomain},
		{"lng is -Inf", 0, math.Inf(-1), 5, h3.ErrLatLngDomain, h3go.ErrLatLngDomain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, wErr := h3.LatLngToCell(h3.LatLng{Lat: tt.lat, Lng: tt.lng}, tt.res)
			if !errors.Is(wErr, tt.wantErr) {
				t.Fatalf("cgo error = %v, want %v", wErr, tt.wantErr)
			}

			_, gErr := h3go.LatLngToCell(h3go.LatLng{Lat: tt.lat, Lng: tt.lng}, tt.res)
			if !errors.Is(gErr, tt.wantGoErr) {
				t.Fatalf("h3go error = %v, want %v", gErr, tt.wantGoErr)
			}
		})
	}
}

// TestCellMethods verifies the methods on h3go's Cell type produce the same
// results as the cgo reference's Cell methods.
func TestCellMethods(t *testing.T) {
	t.Parallel()

	for res := 0; res <= h3.MaxResolution; res++ {
		ref, err := h3.LatLngToCell(h3.LatLng{Lat: 37.7749, Lng: -122.4194}, res)
		if err != nil {
			t.Fatalf("reference LatLngToCell res %d: %v", res, err)
		}

		c := h3go.Cell(ref)

		if got, want := c.Resolution(), ref.Resolution(); got != want {
			t.Fatalf("Resolution() res %d: got %d, want %d", res, got, want)
		}

		if got, want := c.String(), ref.String(); got != want {
			t.Fatalf("String() res %d: got %q, want %q", res, got, want)
		}
	}
}

// res0Cell builds the resolution-0 H3 index for a base cell number. The layout
// is mode=1 (cell), resolution=0, the base cell in its field, and all 15 digit
// slots set to 7 (the canonical "unused digit" pattern).
func res0Cell(baseCell int) h3.Cell {
	const (
		cellMode       = 1
		modeOffset     = 59
		baseCellOffset = 45
		numDigits      = 15
		digitBits      = 3
		allDigitsSet   = (1 << (numDigits * digitBits)) - 1
	)

	return h3.Cell(cellMode<<modeOffset | baseCell<<baseCellOffset | allDigitsSet)
}
