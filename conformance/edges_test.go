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

// TestEdges runs every edges record in the suite against h3.
func TestEdges(t *testing.T) {
	t.Parallel()

	runGroup(t, "edges/", checkEdges)
}

// checkEdges asserts every field of one edges record against h3, plus the
// origin check the group definition requires.
func checkEdges(t *testing.T, record EdgesRecord) {
	t.Helper()

	cell := h3.CellFromString(record.Index)
	target := h3.CellFromString(record.Target)

	edges, err := cell.DirectedEdges()
	if err != nil {
		t.Fatalf("DirectedEdges(): %v", err)
	}

	slices.Sort(edges)

	got := make([]string, len(edges))
	for i, edge := range edges {
		got[i] = edge.String()
	}

	if !slices.Equal(got, record.Edges) {
		t.Fatalf("DirectedEdges() = %v, want %v", got, record.Edges)
	}

	if len(record.Destinations) != len(edges) || len(record.Reverses) != len(edges) || len(record.Vertexes) != len(edges) {
		t.Fatalf("record has %d destinations, %d reverses and %d vertex pairs for %d edges",
			len(record.Destinations), len(record.Reverses), len(record.Vertexes), len(edges))
	}

	for i, edge := range edges {
		origin, err := edge.Origin()
		if err != nil || origin != cell {
			t.Errorf("Origin(%s) = %s (%v), want %s", edge, origin, err, cell)
		}

		destination, err := edge.Destination()
		if err != nil || destination.String() != record.Destinations[i] {
			t.Errorf("Destination(%s) = %s (%v), want %s", edge, destination, err, record.Destinations[i])
		}

		cellsOrigin, cellsDestination, err := edge.Cells()
		if err != nil || cellsOrigin != cell || cellsDestination.String() != record.Destinations[i] {
			t.Errorf("Cells(%s) = %s, %s (%v), want %s, %s", edge, cellsOrigin, cellsDestination, err, cell, record.Destinations[i])
		}

		reverse, err := edge.Reverse()
		if err != nil || reverse.String() != record.Reverses[i] {
			t.Errorf("Reverse(%s) = %s (%v), want %s", edge, reverse, err, record.Reverses[i])
		}

		start, end, err := edge.Vertexes()
		if want := record.Vertexes[i]; err != nil || len(want) != 2 || start.String() != want[0] || end.String() != want[1] {
			t.Errorf("Vertexes(%s) = %s, %s (%v), want %v", edge, start, end, err, want)
		}

		if !edge.IsValid() {
			t.Errorf("IsValid(%s) = false", edge)
		}
	}

	edge, err := cell.DirectedEdge(target)

	result := Result[string]{Err: ErrorName(err)}
	if err == nil {
		result.Value = edge.String()
	}

	if !equalResults(result, record.Edge, equalValues) {
		t.Errorf("DirectedEdge(%s) = %s, want %s", target, formatResult(result), formatResult(record.Edge))
	}
}
