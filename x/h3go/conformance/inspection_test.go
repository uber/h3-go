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
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// TestInspection runs every inspection record in the suite against h3go.
func TestInspection(t *testing.T) {
	t.Parallel()

	manifest := loadSuite(t)

	for _, name := range slices.Sorted(manifestFiles(manifest, "inspection/")) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, err := os.ReadFile(filepath.Join(suiteDir, filepath.FromSlash(name)))
			if err != nil {
				t.Fatal(err)
			}

			for record, err := range Records[InspectionRecord](bytes.NewReader(data)) {
				if err != nil {
					t.Fatal(err)
				}

				t.Run(record.Index, func(t *testing.T) {
					t.Parallel()
					checkInspection(t, record)
				})
			}
		})
	}
}

// checkInspection asserts every field of one inspection record against h3go.
func checkInspection(t *testing.T, record InspectionRecord) {
	t.Helper()

	index := h3go.IndexFromString(record.Index)
	if got := h3go.IndexToString(index); got != record.Index {
		t.Errorf("IndexToString(IndexFromString(%q)) = %q", record.Index, got)
	}

	cell := h3go.Cell(index)

	if got := cell.Resolution(); got != record.Res {
		t.Errorf("Resolution() = %d, want %d", got, record.Res)
	}

	if got := cell.BaseCellNumber(); got != record.BaseCell {
		t.Errorf("BaseCellNumber() = %d, want %d", got, record.BaseCell)
	}

	if got := cell.IsValid(); got != record.ValidCell {
		t.Errorf("IsValid() = %v, want %v", got, record.ValidCell)
	}

	if got := h3go.IsValidIndex(cell); got != record.ValidIndex {
		t.Errorf("IsValidIndex() = %v, want %v", got, record.ValidIndex)
	}

	if got := cell.IsResClassIII(); got != record.ResClassIII {
		t.Errorf("IsResClassIII() = %v, want %v", got, record.ResClassIII)
	}

	if got := cell.IsPentagon(); got != record.Pentagon {
		t.Errorf("IsPentagon() = %v, want %v", got, record.Pentagon)
	}

	if len(record.Digits) != h3go.MaxResolution {
		t.Fatalf("record has %d digits, want %d", len(record.Digits), h3go.MaxResolution)
	}

	for r := 1; r <= h3go.MaxResolution; r++ {
		got, err := cell.IndexDigit(r)
		if err != nil {
			t.Errorf("IndexDigit(%d): %v", r, err)
		}

		if got != record.Digits[r-1] {
			t.Errorf("IndexDigit(%d) = %d, want %d", r, got, record.Digits[r-1])
		}
	}

	faces, err := cell.IcosahedronFaces()
	slices.Sort(faces)

	if got := (Result[[]int]{Value: faces, Err: ErrorName(err)}); !equalResults(got, record.Faces, slices.Equal) {
		t.Errorf("IcosahedronFaces() = %s, want %s", formatResult(got), formatResult(record.Faces))
	}

	constructed, err := h3go.ConstructCell(record.Res, record.BaseCell, record.Digits[:record.Res])

	got := Result[string]{Err: ErrorName(err)}
	if err == nil {
		got.Value = h3go.IndexToString(uint64(constructed))
	}

	if !equalResults(got, record.Construct, func(a, b string) bool { return a == b }) {
		t.Errorf("ConstructCell(%d, %d, %v) = %s, want %s", record.Res, record.BaseCell, record.Digits[:record.Res], formatResult(got), formatResult(record.Construct))
	}
}

// equalResults reports whether two results agree: same error name, or both
// successful with equal values.
func equalResults[T any](got, want Result[T], equal func(T, T) bool) bool {
	if got.Err != want.Err {
		return false
	}

	return got.Err != "" || equal(got.Value, want.Value)
}

// formatResult renders a result as it appears in a record file.
func formatResult[T any](result Result[T]) string {
	data, err := result.MarshalJSON()
	if err != nil {
		return err.Error()
	}

	return strings.TrimSpace(string(data))
}
