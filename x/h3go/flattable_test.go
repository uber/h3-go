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

// TestFlatTableZeroValue confirms a zero flatTable answers membership and
// lookups without allocating and allocates on first insert.
func TestFlatTableZeroValue(t *testing.T) {
	t.Parallel()

	var table flatTable[DirectedEdge, int]

	edges := flatTableEdges(t)

	if table.contains(edges[0]) {
		t.Fatal("zero flatTable contains a key")
	}

	if value, ok := table.lookup(edges[0]); ok || value != 0 {
		t.Fatalf("zero flatTable lookup: got (%d, %t), want (0, false)", value, ok)
	}

	if !table.insert(edges[0], 7) {
		t.Fatal("first insert into zero flatTable reported not new")
	}

	if len(table.entries) != flatTableMinSize {
		t.Fatalf("entries after first insert: got %d, want %d", len(table.entries), flatTableMinSize)
	}

	if value, ok := table.lookup(edges[0]); !ok || value != 7 {
		t.Fatalf("lookup after insert: got (%d, %t), want (7, true)", value, ok)
	}
}

// TestFlatTableInsert checks insert reports new versus duplicate keys, keeps
// the first value for a key, and grows past its initial capacity without losing
// entries.
func TestFlatTableInsert(t *testing.T) {
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

			edges := flatTableEdges(t)[:tt.giveCount]

			table := newFlatTable[DirectedEdge, int](tt.giveExpected)
			for i, edge := range edges {
				if !table.insert(edge, i) {
					t.Fatalf("insert(%v) reported duplicate on first insert", edge)
				}
			}

			for i, edge := range edges {
				if table.insert(edge, -1) {
					t.Fatalf("insert(%v) reported new on second insert", edge)
				}

				if value, ok := table.lookup(edge); !ok || value != i {
					t.Fatalf("lookup(%v): got (%d, %t), want (%d, true)", edge, value, ok, i)
				}
			}

			if table.count != tt.giveCount {
				t.Fatalf("count: got %d, want %d", table.count, tt.giveCount)
			}

			if table.count*flatTableLoadFactor > len(table.entries) {
				t.Fatalf("load factor exceeded: %d entries in %d slots", table.count, len(table.entries))
			}

			other, err := flatTableOrigin(t).GridRing(25)
			if err != nil {
				t.Fatalf("GridRing: %v", err)
			}

			for _, cell := range other {
				edge, err := cell.DirectedEdges()
				if err != nil {
					t.Fatalf("DirectedEdges: %v", err)
				}

				if table.contains(edge[0]) {
					t.Fatalf("contains(%v) true for a key never inserted", edge[0])
				}
			}
		})
	}
}

// TestCellSet covers the set wrapper: a zero set is empty, insert reports new
// versus duplicate cells, and contains answers for members and non-members.
func TestCellSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveExpected int
		giveCount    int
	}{
		{name: "zero value", giveExpected: -1, giveCount: 10},
		{name: "fits without growing", giveExpected: 100, giveCount: 100},
		{name: "grows past estimate", giveExpected: 4, giveCount: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cells, err := flatTableOrigin(t).GridDisk(20)
			if err != nil {
				t.Fatalf("GridDisk: %v", err)
			}

			cells = cells[:tt.giveCount]

			var set cellSet
			if tt.giveExpected >= 0 {
				set = newCellSet(tt.giveExpected)
			}

			if set.contains(cells[0]) {
				t.Fatal("empty cellSet contains a cell")
			}

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

			if set.table.count != tt.giveCount {
				t.Fatalf("count: got %d, want %d", set.table.count, tt.giveCount)
			}

			other, err := flatTableOrigin(t).GridRing(25)
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

// flatTableOrigin returns a hexagon in San Francisco at resolution 9, used as
// the origin for the flatTable tests.
func flatTableOrigin(t *testing.T) Cell {
	t.Helper()

	cell, err := LatLngToCell(LatLng{Lat: 37.7749, Lng: -122.4194}, 9)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	return cell
}

// flatTableEdges returns at least 1000 distinct directed edges: the first edge
// of every cell in a grid disk around flatTableOrigin.
func flatTableEdges(t *testing.T) []DirectedEdge {
	t.Helper()

	cells, err := flatTableOrigin(t).GridDisk(20)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	edges := make([]DirectedEdge, 0, len(cells))

	for _, cell := range cells {
		cellEdges, err := cell.DirectedEdges()
		if err != nil {
			t.Fatalf("DirectedEdges: %v", err)
		}

		edges = append(edges, cellEdges[0])
	}

	return edges
}
