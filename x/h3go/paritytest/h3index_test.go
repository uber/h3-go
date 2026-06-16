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

// TestIntrospectionMatchesCgo sweeps the introspection helpers over the
// reference corpus and asserts the pure-Go output matches the cgo reference.
func TestIntrospectionMatchesCgo(t *testing.T) {
	t.Parallel()
	corpus := referenceCorpus(t)

	t.Run("Resolution", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).Resolution(), ref.Resolution(); got != want {
				t.Fatalf("Resolution(%015x): got %d, want %d", uint64(ref), got, want)
			}
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).IsValid(), ref.IsValid(); got != want {
				t.Fatalf("IsValid(%015x): got %v, want %v", uint64(ref), got, want)
			}
		}
	})

	t.Run("IsPentagon", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).IsPentagon(), ref.IsPentagon(); got != want {
				t.Fatalf("IsPentagon(%015x): got %v, want %v", uint64(ref), got, want)
			}
		}
	})

	t.Run("IsResClassIII", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).IsResClassIII(), ref.IsResClassIII(); got != want {
				t.Fatalf("IsResClassIII(%015x): got %v, want %v", uint64(ref), got, want)
			}
		}
	})

	t.Run("BaseCellNumber", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			c := h3goCell(ref)
			if got, want := c.BaseCellNumber(), ref.BaseCellNumber(); got != want {
				t.Fatalf("BaseCellNumber(%015x): got %d, want %d", uint64(ref), got, want)
			}

			if got, want := h3go.BaseCellNumber(c), h3.BaseCellNumber(ref); got != want {
				t.Fatalf("BaseCellNumber func(%015x): got %d, want %d", uint64(ref), got, want)
			}
		}
	})

	t.Run("IndexDigit", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			assertIndexDigitsMatch(t, h3goCell(ref), ref)
		}
	})
}

func assertIndexDigitsMatch(t *testing.T, c h3go.Cell, ref h3.Cell) {
	t.Helper()

	for res := 1; res <= h3.MaxResolution; res++ {
		got, gErr := c.IndexDigit(res)
		want, wErr := ref.IndexDigit(res)

		if !bothErr(gErr, wErr) {
			t.Fatalf("IndexDigit(%015x, %d): err got=%v want=%v", uint64(ref), res, gErr, wErr)
		}

		if wErr == nil && got != want {
			t.Fatalf("IndexDigit(%015x, %d): got %d, want %d", uint64(ref), res, got, want)
		}
	}
}

// TestNumCellsMatchesCgo checks NumCells across all resolutions against the cgo
// reference.
func TestNumCellsMatchesCgo(t *testing.T) {
	t.Parallel()

	for res := 0; res <= h3.MaxResolution; res++ {
		if got, want := h3go.NumCells(res), h3.NumCells(res); got != want {
			t.Fatalf("NumCells(%d): got %d, want %d", res, got, want)
		}
	}
}
