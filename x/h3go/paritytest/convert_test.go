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

// TestCellStringMatchesCgo verifies CellToString / CellFromString / String
// round-trip against the cgo reference over the corpus.
func TestCellStringMatchesCgo(t *testing.T) {
	t.Parallel()
	corpus := referenceCorpus(t)

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).String(), ref.String(); got != want {
				t.Fatalf("String(%015x): got %q, want %q", uint64(ref), got, want)
			}
		}
	})

	t.Run("CellToString", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3go.CellToString(h3goCell(ref)), h3.CellToString(ref); got != want {
				t.Fatalf("CellToString(%015x): got %q, want %q", uint64(ref), got, want)
			}
		}
	})

	t.Run("CellFromString", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got := h3go.CellFromString(ref.String()); uint64(got) != uint64(ref) {
				t.Fatalf("CellFromString round-trip: got %015x, want %015x", uint64(got), uint64(ref))
			}
		}
	})
}
