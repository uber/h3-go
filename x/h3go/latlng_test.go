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
		"resolution_above_max":  {giveLat: 0, giveLng: 0, giveRes: maxResolution + 1, wantErr: ErrResolutionDomain},
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

	for res := 0; res <= maxResolution; res++ {
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
		resCount   = maxResolution + 1
	)

	rng := rand.New(rand.NewSource(seed))
	for range iterations {
		lat := rng.Float64()*latSpan - latOffset
		lng := rng.Float64()*lngSpan - lngOffset
		assertValidAtRes(t, lat, lng, rng.Intn(resCount))
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
