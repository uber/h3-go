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

	"github.com/uber/h3-go/v4"
)

// TestTraversal runs every traversal record in the suite against h3.
func TestTraversal(t *testing.T) {
	t.Parallel()

	runGroup(t, "traversal/", checkTraversal)
}

// checkTraversal asserts every field of one traversal record against h3.
// The record carries the rings; the k-disk it implies is the cell and the
// rings 1 to k, and the distance-grouped disk is those rings in order.
func checkTraversal(t *testing.T, record TraversalRecord) {
	t.Helper()

	cell := h3.CellFromString(record.Index)
	target := h3.CellFromString(record.Target)

	rings := [][]string{{record.Index}, record.Ring1, record.Ring2, record.Ring3}

	var disk []string

	for k := range rings {
		disk = slices.Sorted(slices.Values(slices.Concat(disk, rings[k])))

		if k == 0 {
			continue
		}

		ring, err := cell.GridRing(k)
		if got := cellsResult(ring, err); !equalResults(got, Result[[]string]{Value: rings[k]}, slices.Equal) {
			t.Errorf("GridRing(%d) = %s, want %v", k, formatResult(got), rings[k])
		}

		cells, err := cell.GridDisk(k)
		if got := cellsResult(cells, err); !equalResults(got, Result[[]string]{Value: disk}, slices.Equal) {
			t.Errorf("GridDisk(%d) = %s, want %v", k, formatResult(got), disk)
		}

		distances, err := cell.GridDiskDistances(k)
		if err != nil {
			t.Errorf("GridDiskDistances(%d): %v", k, err)

			continue
		}

		if len(distances) != k+1 {
			t.Errorf("GridDiskDistances(%d) has %d rings, want %d", k, len(distances), k+1)

			continue
		}

		for distance, want := range rings[:k+1] {
			if got := cellsResult(distances[distance], nil); !slices.Equal(got.Value, want) {
				t.Errorf("GridDiskDistances(%d)[%d] = %v, want %v", k, distance, got.Value, want)
			}
		}
	}

	distance, err := cell.GridDistance(target)
	if got := (Result[int64]{Value: int64(distance), Err: ErrorName(err)}); !equalResults(got, record.Distance, equalValues) {
		t.Errorf("GridDistance(%s) = %s, want %s", target, formatResult(got), formatResult(record.Distance))
	}

	path, err := cell.GridPath(target)
	if got := cellSequenceResult(path, err); !equalResults(got, record.Path, slices.Equal) {
		t.Errorf("GridPath(%s) = %s, want %s", target, formatResult(got), formatResult(record.Path))
	}

	neighbor, err := cell.IsNeighbor(target)
	if got := (Result[bool]{Value: neighbor, Err: ErrorName(err)}); !equalResults(got, record.Neighbor, equalValues) {
		t.Errorf("IsNeighbor(%s) = %s, want %s", target, formatResult(got), formatResult(record.Neighbor))
	}
}

// cellSequenceResult packs an ordered cell-returning call into a Result of
// string forms, preserving order.
func cellSequenceResult(cells []h3.Cell, err error) Result[[]string] {
	if err != nil {
		return Result[[]string]{Err: ErrorName(err)}
	}

	out := make([]string, len(cells))
	for i, cell := range cells {
		out[i] = cell.String()
	}

	return Result[[]string]{Value: out}
}
