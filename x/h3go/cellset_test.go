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

package h3go

import "testing"

// TestCellSetZeroValue confirms a zero cellSet answers membership without
// allocating and allocates on first insert.
func TestCellSetZeroValue(t *testing.T) {
	t.Parallel()

	var set cellSet

	if set.contains(cellSetOrigin(t)) {
		t.Fatal("zero cellSet contains a cell")
	}

	if !set.insert(cellSetOrigin(t)) {
		t.Fatal("first insert into zero cellSet reported not new")
	}

	if len(set.slots) != cellSetMinSize {
		t.Fatalf("slots after first insert: got %d, want %d", len(set.slots), cellSetMinSize)
	}

	if !set.contains(cellSetOrigin(t)) {
		t.Fatal("cellSet missing the cell just inserted")
	}
}

// TestCellSetInsert checks insert reports new versus duplicate cells and that
// the set grows past its initial capacity without losing members.
func TestCellSetInsert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveExpected int
		giveCount    int
	}{
		{name: "fits without growing", giveExpected: 100, giveCount: 100},
		{name: "grows past estimate", giveExpected: 4, giveCount: 500},
		{name: "grows from minimum", giveExpected: 0, giveCount: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cells, err := cellSetOrigin(t).GridDisk(20)
			if err != nil {
				t.Fatalf("GridDisk: %v", err)
			}

			cells = cells[:tt.giveCount]

			set := newCellSet(tt.giveExpected)
			for _, cell := range cells {
				if !set.insert(cell) {
					t.Fatalf("insert(%v) reported duplicate on first insert", cell)
				}
			}

			for _, cell := range cells {
				if set.insert(cell) {
					t.Fatalf("insert(%v) reported new on second insert", cell)
				}

				if !set.contains(cell) {
					t.Fatalf("contains(%v) false after insert", cell)
				}
			}

			if set.count != tt.giveCount {
				t.Fatalf("count: got %d, want %d", set.count, tt.giveCount)
			}

			if set.count*cellSetLoadFactor > len(set.slots) {
				t.Fatalf("load factor exceeded: %d cells in %d slots", set.count, len(set.slots))
			}

			other, err := cellSetOrigin(t).GridRing(25)
			if err != nil {
				t.Fatalf("GridRing: %v", err)
			}

			for _, cell := range other {
				if set.contains(cell) {
					t.Fatalf("contains(%v) true for a cell never inserted", cell)
				}
			}
		})
	}
}

// cellSetOrigin returns a hexagon in San Francisco at resolution 9, used as
// the origin for the cellSet tests.
func cellSetOrigin(t *testing.T) Cell {
	t.Helper()

	cell, err := LatLngToCell(LatLng{Lat: 37.7749, Lng: -122.4194}, 9)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	return cell
}
