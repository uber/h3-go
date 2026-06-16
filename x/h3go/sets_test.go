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

// TestSetH3IndexValue checks that setH3Index assembles the expected index.
func TestSetH3IndexValue(t *testing.T) {
	t.Parallel()

	h := setH3Index(5, 12, 1)
	if h.Resolution() != 5 {
		t.Fatalf("resolution = %d, want 5", h.Resolution())
	}

	if h.BaseCellNumber() != 12 {
		t.Fatalf("base cell = %d, want 12", h.BaseCellNumber())
	}

	if h.mode() != cellMode {
		t.Fatalf("mode = %d, want %d", h.mode(), cellMode)
	}

	for r := 1; r <= 5; r++ {
		if h.indexDigit(r) != 1 {
			t.Fatalf("digit %d = %d, want 1", r, h.indexDigit(r))
		}
	}

	for r := 6; r <= maxResolution; r++ {
		if h.indexDigit(r) != invalidDigit {
			t.Fatalf("digit %d = %d, want %d", r, h.indexDigit(r), invalidDigit)
		}
	}

	if uint64(h) != 0x85184927fffffff {
		t.Fatalf("index = %015x, want %015x", uint64(h), 0x85184927fffffff)
	}
}

// TestRes0Cells checks that Res0Cells returns every base cell as a valid res-0
// index.
func TestRes0Cells(t *testing.T) {
	t.Parallel()

	cells, err := Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	if len(cells) != numBaseCells {
		t.Fatalf("Res0Cells: got %d cells, want %d", len(cells), numBaseCells)
	}

	for _, c := range cells {
		if !c.IsValid() || c.Resolution() != 0 {
			t.Fatalf("Res0Cells: %015x is not a valid res-0 cell", uint64(c))
		}
	}
}

// TestPentagons checks that Pentagons returns the 12 pentagons at each
// resolution.
func TestPentagons(t *testing.T) {
	t.Parallel()

	for res := 0; res <= maxResolution; res++ {
		pents, err := Pentagons(res)
		if err != nil {
			t.Fatalf("Pentagons(%d): %v", res, err)
		}

		if len(pents) != numPentagons {
			t.Fatalf("Pentagons(%d): got %d, want %d", res, len(pents), numPentagons)
		}

		for _, p := range pents {
			if !p.IsPentagon() || p.Resolution() != res {
				t.Fatalf("Pentagons(%d): %015x is not a valid pentagon", res, uint64(p))
			}
		}
	}
}

// TestPentagonsInvalid checks that Pentagons rejects out-of-range resolutions.
func TestPentagonsInvalid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveRes int
	}{
		"too_high_resolution": {giveRes: 16},
		"absurd_resolution":   {giveRes: 100},
		"negative_resolution": {giveRes: -1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := Pentagons(tt.giveRes); err == nil {
				t.Fatalf("Pentagons(%d): expected error", tt.giveRes)
			}
		})
	}
}
