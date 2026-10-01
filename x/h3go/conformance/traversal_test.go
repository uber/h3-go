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

// TestTraversal runs every traversal record in the suite against h3go.
func TestTraversal(t *testing.T) {
	t.Parallel()

	runGroup(t, "traversal/", checkTraversal)
}

// checkTraversal asserts every field of one traversal record against h3go.
func checkTraversal(t *testing.T, record TraversalRecord) {
	t.Helper()

	cell := h3go.CellFromString(record.Index)
	target := h3go.CellFromString(record.Target)

	disks := [][]string{record.Disk1, record.Disk2, record.Disk3}
	rings := [][]string{record.Ring1, record.Ring2, record.Ring3}

	for k := 1; k <= len(disks); k++ {
		disk, err := cell.GridDisk(k)
		if got := cellsResult(disk, err); !equalResults(got, Result[[]string]{Value: disks[k-1]}, slices.Equal) {
			t.Errorf("GridDisk(%d) = %s, want %v", k, formatResult(got), disks[k-1])
		}

		ring, err := cell.GridRing(k)
		if got := cellsResult(ring, err); !equalResults(got, Result[[]string]{Value: rings[k-1]}, slices.Equal) {
			t.Errorf("GridRing(%d) = %s, want %v", k, formatResult(got), rings[k-1])
		}
	}

	distances, err := cell.GridDiskDistances(2)
	if err != nil {
		t.Errorf("GridDiskDistances(2): %v", err)
	}

	if len(distances) != len(record.DiskDistances2) {
		t.Fatalf("GridDiskDistances(2) has %d rings, want %d", len(distances), len(record.DiskDistances2))
	}

	for distance, want := range record.DiskDistances2 {
		if got := cellsResult(distances[distance], nil); !slices.Equal(got.Value, want) {
			t.Errorf("GridDiskDistances(2)[%d] = %v, want %v", distance, got.Value, want)
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
func cellSequenceResult(cells []h3go.Cell, err error) Result[[]string] {
	if err != nil {
		return Result[[]string]{Err: ErrorName(err)}
	}

	out := make([]string, len(cells))
	for i, cell := range cells {
		out[i] = cell.String()
	}

	return Result[[]string]{Value: out}
}
