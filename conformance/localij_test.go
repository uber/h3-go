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
	"testing"

	"github.com/uber/h3-go/v4"
)

// TestLocalIJ runs every local IJ record in the suite against h3.
func TestLocalIJ(t *testing.T) {
	t.Parallel()

	runGroup(t, "localij/", checkLocalIJ)
}

// checkLocalIJ asserts every field of one local IJ record against h3.
func checkLocalIJ(t *testing.T, record LocalIJRecord) {
	t.Helper()

	origin := h3.CellFromString(record.Index)
	target := h3.CellFromString(record.Target)

	ij, err := h3.CellToLocalIJ(origin, target)

	got := Result[CoordIJ]{Value: CoordIJ{I: ij.I, J: ij.J}, Err: ErrorName(err)}
	if !equalResults(got, record.IJ, equalValues) {
		t.Errorf("CellToLocalIJ(%s) = %s, want %s", target, formatResult(got), formatResult(record.IJ))
	}

	if record.IJ.Err != "" {
		if record.Cell.Err != record.IJ.Err {
			t.Errorf("record cell error %q differs from ij error %q", record.Cell.Err, record.IJ.Err)
		}

		return
	}

	cell, err := h3.LocalIJToCell(origin, h3.CoordIJ{I: record.IJ.Value.I, J: record.IJ.Value.J})
	if got := cellResult(cell, err); !equalResults(got, record.Cell, equalValues) {
		t.Errorf("LocalIJToCell(%+v) = %s, want %s", record.IJ.Value, formatResult(got), formatResult(record.Cell))
	}
}
