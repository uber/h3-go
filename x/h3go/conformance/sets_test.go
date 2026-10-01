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

package conformance

import (
	"slices"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// TestSets runs every sets record in the suite against h3go.
func TestSets(t *testing.T) {
	t.Parallel()

	runGroup(t, "sets/", checkSets)
}

// checkSets asserts every field of one sets record against h3go, plus the
// round trip the group definition requires.
func checkSets(t *testing.T, record SetsRecord) {
	t.Helper()

	input := make([]h3go.Cell, len(record.Input))
	allValid := true

	for i, index := range record.Input {
		input[i] = h3go.CellFromString(index)
		allValid = allValid && input[i].IsValid()
	}

	compacted, err := h3go.CompactCells(input)
	if got := cellsResult(compacted, err); !equalResults(got, record.Compact, slices.Equal) {
		t.Errorf("CompactCells() = %s, want %s", formatResult(got), formatResult(record.Compact))
	}

	uncompacted, err := h3go.UncompactCells(input, record.UncompactRes)
	if got := cellsResult(uncompacted, err); !equalResults(got, record.Uncompact, slices.Equal) {
		t.Errorf("UncompactCells(%d) = %s, want %s", record.UncompactRes, formatResult(got), formatResult(record.Uncompact))
	}

	if record.Compact.Err != "" || !allValid {
		return
	}

	finest := 0
	for _, cell := range input {
		finest = max(finest, cell.Resolution())
	}

	roundTrip, err := h3go.UncompactCells(compacted, finest)
	if got, want := cellsResult(roundTrip, err), cellsResult(input, nil); !equalResults(got, want, slices.Equal) {
		t.Errorf("UncompactCells(compact, %d) = %s, want %s", finest, formatResult(got), formatResult(want))
	}
}
