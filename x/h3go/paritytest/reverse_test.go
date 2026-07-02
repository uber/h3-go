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

// llTolerance is the maximum allowed difference, in degrees, between the pure-Go
// and cgo geographic coordinates. Both implementations run the same arithmetic,
// so results agree to within floating-point rounding.
const llTolerance = 1e-9

// TestCellToLatLngMatchesCgo asserts the pure-Go cell center matches the
// cgo-backed reference for every cell in the shared corpus.
func TestCellToLatLngMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, ref := range referenceCorpus(t) {
		want, wErr := h3.CellToLatLng(ref)
		got, gErr := h3go.CellToLatLng(h3goCell(ref))

		if !bothErr(wErr, gErr) {
			t.Fatalf("CellToLatLng(%015x) error mismatch: cgo=%v h3go=%v", uint64(ref), wErr, gErr)
		}

		assertSameLatLng(t, got.Lat, got.Lng, want.Lat, want.Lng, uint64(ref))
	}
}

// TestCellToBoundaryMatchesCgo asserts the pure-Go cell boundary matches the
// cgo-backed reference, vertex for vertex, for every cell in the shared corpus.
func TestCellToBoundaryMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, ref := range referenceCorpus(t) {
		want, wErr := h3.CellToBoundary(ref)
		got, gErr := h3go.CellToBoundary(h3goCell(ref))

		if !bothErr(wErr, gErr) {
			t.Fatalf("CellToBoundary(%015x) error mismatch: cgo=%v h3go=%v", uint64(ref), wErr, gErr)
		}

		if len(got) != len(want) {
			t.Fatalf("CellToBoundary(%015x): %d vertices, want %d", uint64(ref), len(got), len(want))
		}

		for i := range want {
			assertSameLatLng(t, got[i].Lat, got[i].Lng, want[i].Lat, want[i].Lng, uint64(ref))
		}
	}
}

// assertSameLatLng fails if two coordinates differ by more than llTolerance.
func assertSameLatLng(t *testing.T, gotLat, gotLng, wantLat, wantLng float64, cell uint64) {
	t.Helper()

	if math.Abs(gotLat-wantLat) > llTolerance || math.Abs(gotLng-wantLng) > llTolerance {
		t.Fatalf("cell %015x: got {%v %v}, want {%v %v}", cell, gotLat, gotLng, wantLat, wantLng)
	}
}
