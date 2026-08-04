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
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// TestRes0CellsMatchesCgo checks Res0Cells against the cgo reference (as a set).
func TestRes0CellsMatchesCgo(t *testing.T) {
	t.Parallel()

	got, err := h3go.Res0Cells()
	if err != nil {
		t.Fatalf("Res0Cells: %v", err)
	}

	want, err := h3.Res0Cells()
	if err != nil {
		t.Fatalf("h3.Res0Cells: %v", err)
	}

	if len(got) != 122 {
		t.Fatalf("Res0Cells: got %d cells, want 122", len(got))
	}

	assertSameCellSet(t, got, want, "Res0Cells")

	for _, c := range got {
		if !c.IsValid() {
			t.Fatalf("Res0Cells: %015x is not valid", uint64(c))
		}

		if c.Resolution() != 0 {
			t.Fatalf("Res0Cells: %015x has resolution %d, want 0", uint64(c), c.Resolution())
		}
	}
}

// TestPentagonsMatchesCgo checks Pentagons across all resolutions against the
// cgo reference, and verifies validity/pentagon/resolution properties of the
// results.
func TestPentagonsMatchesCgo(t *testing.T) {
	t.Parallel()

	for res := 0; res <= h3.MaxResolution; res++ {
		got, err := h3go.Pentagons(res)
		if err != nil {
			t.Fatalf("Pentagons(%d): %v", res, err)
		}

		want, err := h3.Pentagons(res)
		if err != nil {
			t.Fatalf("h3.Pentagons(%d): %v", res, err)
		}

		assertSameCellSet(t, got, want, "Pentagons")

		if len(got) != 12 {
			t.Fatalf("Pentagons(%d): got %d, want 12", res, len(got))
		}
		seen := make(map[h3go.Cell]bool)

		for _, c := range got {
			if !c.IsValid() {
				t.Fatalf("Pentagons(%d): %015x is not valid", res, uint64(c))
			}

			if !c.IsPentagon() {
				t.Fatalf("Pentagons(%d): %015x is not a pentagon", res, uint64(c))
			}

			if c.Resolution() != res {
				t.Fatalf("Pentagons(%d): %015x has resolution %d", res, uint64(c), c.Resolution())
			}

			if seen[c] {
				t.Fatalf("Pentagons(%d): %015x seen twice", res, uint64(c))
			}
			seen[c] = true
		}
	}
}
