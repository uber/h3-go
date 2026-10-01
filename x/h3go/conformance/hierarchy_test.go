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

// TestHierarchy runs every hierarchy record in the suite against h3go.
func TestHierarchy(t *testing.T) {
	t.Parallel()

	runGroup(t, "hierarchy/", checkHierarchy)
}

// checkHierarchy asserts every field of one hierarchy record against h3go,
// plus the childPosToCell round trip the group definition requires.
func checkHierarchy(t *testing.T, record HierarchyRecord) {
	t.Helper()

	cell := h3go.CellFromString(record.Index)
	res := cell.Resolution()

	if len(record.Parents) != res {
		t.Fatalf("record has %d parents, want %d", len(record.Parents), res)
	}

	for parentRes, want := range record.Parents {
		got, err := cell.Parent(parentRes)
		if err != nil {
			t.Errorf("Parent(%d): %v", parentRes, err)
		}

		if got.String() != want {
			t.Errorf("Parent(%d) = %s, want %s", parentRes, got, want)
		}
	}

	centerChild, err := cell.CenterChild(res + 1)
	if got := cellResult(centerChild, err); !equalResults(got, record.CenterChild, equalValues) {
		t.Errorf("CenterChild(%d) = %s, want %s", res+1, formatResult(got), formatResult(record.CenterChild))
	}

	children, err := cell.Children(res + 1)
	if got := cellsResult(children, err); !equalResults(got, record.Children, slices.Equal) {
		t.Errorf("Children(%d) = %s, want %s", res+1, formatResult(got), formatResult(record.Children))
	}

	childPos, err := cell.ChildPos(0)
	if got := (Result[int64]{Value: int64(childPos), Err: ErrorName(err)}); !equalResults(got, record.ChildPos, equalValues) {
		t.Errorf("ChildPos(0) = %s, want %s", formatResult(got), formatResult(record.ChildPos))
	}

	if record.ChildPos.Err != "" {
		return
	}

	ancestor := cell
	if res > 0 {
		ancestor = h3go.CellFromString(record.Parents[0])
	}

	roundTrip, err := ancestor.ChildPosToCell(int(record.ChildPos.Value), res)
	if err != nil || roundTrip != cell {
		t.Errorf("ChildPosToCell(%d, %s, %d) = %s (%v), want %s", record.ChildPos.Value, ancestor, res, roundTrip, err, cell)
	}
}

// cellResult packs a cell-returning call into a Result of its string form.
func cellResult(cell h3go.Cell, err error) Result[string] {
	if err != nil {
		return Result[string]{Err: ErrorName(err)}
	}

	return Result[string]{Value: cell.String()}
}

// cellsResult packs a cell-set-returning call into a Result of sorted string
// forms.
func cellsResult(cells []h3go.Cell, err error) Result[[]string] {
	if err != nil {
		return Result[[]string]{Err: ErrorName(err)}
	}

	slices.Sort(cells)

	out := make([]string, len(cells))
	for i, cell := range cells {
		out[i] = cell.String()
	}

	return Result[[]string]{Value: out}
}

// equalValues compares two comparable values.
func equalValues[T comparable](a, b T) bool {
	return a == b
}
