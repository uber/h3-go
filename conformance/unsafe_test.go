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

// TestUnsafe runs every unsafe-traversal record in the suite against h3.
func TestUnsafe(t *testing.T) {
	t.Parallel()

	runGroup(t, "unsafe/", checkUnsafe)
}

// checkUnsafe asserts that each unsafe traversal of the record's cell refuses
// exactly where the reference does, and that when it succeeds it returns the
// safe traversal's cells. GridDiskDistancesUnsafe must agree with
// GridDiskUnsafe, as it does in the reference, where one implements the other.
func checkUnsafe(t *testing.T, record UnsafeRecord) {
	t.Helper()

	cell := h3.CellFromString(record.Index)

	disks := record.DiskUnsafe
	rings := record.RingUnsafe

	if len(disks) != len(rings) {
		t.Fatalf("record has %d disk outcomes and %d ring outcomes", len(disks), len(rings))
	}

	for k := 1; k <= len(disks); k++ {
		want := disks[k-1]

		disk, err := cell.GridDiskUnsafe(k)
		checkUnsafeOutcome(t, "GridDiskUnsafe", k, cellsResult(disk, err), want, func() []h3.Cell {
			safe, safeErr := cell.GridDisk(k)
			if safeErr != nil {
				t.Fatalf("GridDisk(%d): %v", k, safeErr)
			}

			return safe
		})

		distances, err := cell.GridDiskDistancesUnsafe(k)
		checkUnsafeOutcome(t, "GridDiskDistancesUnsafe", k, cellsResult(slices.Concat(distances...), err), want, func() []h3.Cell {
			return disk
		})

		ring, err := cell.GridRingUnsafe(k)
		checkUnsafeOutcome(t, "GridRingUnsafe", k, cellsResult(ring, err), rings[k-1], func() []h3.Cell {
			safe, safeErr := cell.GridRing(k)
			if safeErr != nil {
				t.Fatalf("GridRing(%d): %v", k, safeErr)
			}

			return safe
		})
	}
}

// checkUnsafeOutcome compares one unsafe call's outcome with the record. On a
// recorded success the call must succeed and return the cells that safe
// yields; on a recorded error it must fail with the same error.
func checkUnsafeOutcome(t *testing.T, name string, k int, got Result[[]string], want Result[bool], safe func() []h3.Cell) {
	t.Helper()

	if want.Err != "" {
		if got.Err != want.Err {
			t.Errorf("%s(%d) = %s, want error %s", name, k, formatResult(got), want.Err)
		}

		return
	}

	if got.Err != "" {
		t.Errorf("%s(%d) = %s, want success", name, k, formatResult(got))

		return
	}

	if wantCells := cellsResult(safe(), nil); !slices.Equal(got.Value, wantCells.Value) {
		t.Errorf("%s(%d) = %v, want the safe traversal's %v", name, k, got.Value, wantCells.Value)
	}
}
