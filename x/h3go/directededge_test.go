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

import (
	"errors"
	"testing"
)

// TestDirectedEdgeRoundTrip checks edge construction and accessors across the
// corpus: each cell's edges resolve to neighbor cells, reverse, and round trip.
func TestDirectedEdgeRoundTrip(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		edges, err := origin.DirectedEdges()
		if err != nil {
			t.Fatalf("DirectedEdges(%015x): %v", uint64(origin), err)
		}

		expected := numCellEdges
		if origin.IsPentagon() {
			expected = numCellEdges - 1
		}

		if len(edges) != expected {
			t.Fatalf("DirectedEdges(%015x): got %d, want %d", uint64(origin), len(edges), expected)
		}

		for _, edge := range edges {
			if !edge.IsValid() {
				t.Fatalf("edge %015x is not valid", uint64(edge))
			}

			gotOrigin, err := edge.Origin()
			if err != nil || gotOrigin != origin {
				t.Fatalf("Origin(%015x): got %015x (%v), want %015x", uint64(edge), uint64(gotOrigin), err, uint64(origin))
			}

			dest, err := edge.Destination()
			if err != nil {
				t.Fatalf("Destination(%015x): %v", uint64(edge), err)
			}

			rebuilt, err := origin.DirectedEdge(dest)
			if err != nil || rebuilt != edge {
				t.Fatalf("DirectedEdge round trip: got %015x (%v), want %015x", uint64(rebuilt), err, uint64(edge))
			}

			cells, err := edge.Cells()
			if err != nil || cells[0] != origin || cells[1] != dest {
				t.Fatalf("Cells(%015x): got %v (%v)", uint64(edge), cells, err)
			}

			reverse, err := edge.Reverse()
			if err != nil {
				t.Fatalf("Reverse(%015x): %v", uint64(edge), err)
			}

			if revOrigin, _ := reverse.Origin(); revOrigin != dest {
				t.Fatalf("Reverse(%015x) origin: got %015x, want %015x", uint64(edge), uint64(revOrigin), uint64(dest))
			}

			if edge.Resolution() != origin.Resolution() {
				t.Fatalf("edge resolution mismatch")
			}
		}
	}
}

// TestDirectedEdgeNotNeighbors covers the not-neighbors error path.
func TestDirectedEdgeNotNeighbors(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")
	far := CellFromString("85283473fffffff")

	if _, err := origin.DirectedEdge(far); !errors.Is(err, ErrNotNeighbors) {
		t.Fatalf("DirectedEdge(non-neighbor): got %v, want ErrNotNeighbors", err)
	}

	// Self is not a neighbor of itself.
	if _, err := origin.DirectedEdge(origin); !errors.Is(err, ErrNotNeighbors) {
		t.Fatalf("DirectedEdge(self): got %v, want ErrNotNeighbors", err)
	}
}

// TestReverseDirectedEdgeFuzzFail ports the testDirectedEdge.c fuzz_fail
// regression: a malformed index whose decoded origin and destination are not
// neighbors must report ErrNotNeighbors rather than producing an edge.
func TestReverseDirectedEdgeFuzzFail(t *testing.T) {
	t.Parallel()

	if _, err := DirectedEdge(0x1001fff7ff2fbfff).Reverse(); !errors.Is(err, ErrNotNeighbors) {
		t.Fatalf("Reverse(fuzz): got %v, want ErrNotNeighbors", err)
	}
}

// TestDirectedEdgeBoundary checks the edge boundary against the cell boundary it
// is derived from over the corpus.
func TestDirectedEdgeBoundary(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		edges, err := origin.DirectedEdges()
		if err != nil {
			t.Fatalf("DirectedEdges: %v", err)
		}

		for _, edge := range edges {
			boundary, err := edge.Boundary()
			if err != nil {
				t.Fatalf("Boundary(%015x): %v", uint64(edge), err)
			}

			if len(boundary) < numEdgeCells {
				t.Fatalf("Boundary(%015x): got %d verts, want >= %d", uint64(edge), len(boundary), numEdgeCells)
			}

			length, err := EdgeLengthRads(edge)
			if err != nil || length <= 0 {
				t.Fatalf("EdgeLengthRads(%015x): got %v (%v), want > 0", uint64(edge), length, err)
			}
		}
	}
}

// TestEdgeLengthUnits checks the unit conversions of edge length relative to one
// another.
func TestEdgeLengthUnits(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	edges, err := origin.DirectedEdges()
	if err != nil {
		t.Fatalf("DirectedEdges: %v", err)
	}

	edge := edges[0]

	rads, err := EdgeLengthRads(edge)
	if err != nil {
		t.Fatalf("EdgeLengthRads: %v", err)
	}

	km, err := EdgeLengthKm(edge)
	if err != nil {
		t.Fatalf("EdgeLengthKm: %v", err)
	}

	meters, err := EdgeLengthM(edge)
	if err != nil {
		t.Fatalf("EdgeLengthM: %v", err)
	}

	if km <= rads || meters <= km {
		t.Fatalf("edge length units not ordered: rads=%v km=%v m=%v", rads, km, meters)
	}
}

// TestDirectedEdgeInvalid covers the failure paths of the edge accessors and
// validity check for malformed edges.
func TestDirectedEdgeInvalid(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	// An edge with reserved bits of center (0) is invalid.
	centerEdge := DirectedEdge(origin.setMode(directedEdgeMode).setReservedBits(centerDigit))
	if centerEdge.IsValid() {
		t.Fatal("edge with center direction should be invalid")
	}

	// A cell-mode index is not a valid edge, and its accessors fail.
	cellModeEdge := DirectedEdge(origin)
	if cellModeEdge.IsValid() {
		t.Fatal("cell-mode index should not be a valid edge")
	}

	if _, err := cellModeEdge.Origin(); !errors.Is(err, ErrDirectedEdgeInvalid) {
		t.Fatalf("Origin(cell mode): got %v, want ErrDirectedEdgeInvalid", err)
	}

	if _, err := cellModeEdge.Destination(); !errors.Is(err, ErrDirectedEdgeInvalid) {
		t.Fatalf("Destination(cell mode): got %v, want ErrDirectedEdgeInvalid", err)
	}

	if _, err := cellModeEdge.Reverse(); !errors.Is(err, ErrDirectedEdgeInvalid) {
		t.Fatalf("Reverse(cell mode): got %v, want ErrDirectedEdgeInvalid", err)
	}

	if _, err := cellModeEdge.Cells(); !errors.Is(err, ErrDirectedEdgeInvalid) {
		t.Fatalf("Cells(cell mode): got %v, want ErrDirectedEdgeInvalid", err)
	}

	if _, err := cellModeEdge.Boundary(); !errors.Is(err, ErrDirectedEdgeInvalid) {
		t.Fatalf("Boundary(cell mode): got %v, want ErrDirectedEdgeInvalid", err)
	}

	// An edge over a pentagon in the deleted k direction is invalid.
	pents, err := Pentagons(2)
	if err != nil {
		t.Fatalf("Pentagons: %v", err)
	}

	kEdge := DirectedEdge(pents[0].setMode(directedEdgeMode).setReservedBits(kAxesDigit))
	if kEdge.IsValid() {
		t.Fatal("pentagon k-axis edge should be invalid")
	}

	// Its boundary cannot be built (no valid start vertex).
	if _, err := kEdge.Boundary(); !errors.Is(err, ErrDirectedEdgeInvalid) {
		t.Fatalf("Boundary(pentagon k edge): got %v, want ErrDirectedEdgeInvalid", err)
	}
}

// TestDirectedEdgeDestinationError covers the destination failure path when the
// origin cannot step to a neighbor (corrupt base cell).
func TestDirectedEdgeDestinationError(t *testing.T) {
	t.Parallel()

	corrupt := CellFromString("8928308280fffff").setBaseCell(NumBaseCells)
	edge := DirectedEdge(corrupt.setMode(directedEdgeMode).setReservedBits(jAxesDigit))

	if _, err := edge.Destination(); err == nil {
		t.Fatal("Destination(corrupt origin): got nil error, want failure")
	}

	if _, err := edge.Cells(); err == nil {
		t.Fatal("Cells(corrupt origin): got nil error, want failure")
	}

	if _, err := edge.Reverse(); err == nil {
		t.Fatal("Reverse(corrupt origin): got nil error, want failure")
	}

	if _, err := EdgeLengthKm(edge); err == nil {
		t.Fatal("EdgeLengthKm(corrupt origin): got nil error, want failure")
	}

	if _, err := EdgeLengthM(edge); err == nil {
		t.Fatal("EdgeLengthM(corrupt origin): got nil error, want failure")
	}

	if _, err := edge.Boundary(); err == nil {
		t.Fatal("Boundary(corrupt origin): got nil error, want failure")
	}
}

// TestDirectedEdgeIsValidOriginError covers the IsValid branch where the index
// has a valid direction but is not in directed-edge mode, so Origin fails.
func TestDirectedEdgeIsValidOriginError(t *testing.T) {
	t.Parallel()

	// Cell mode with a valid neighbor direction in the reserved bits: the
	// direction check passes but Origin rejects the non-edge mode.
	cellWithDir := DirectedEdge(CellFromString("8928308280fffff").setReservedBits(jAxesDigit))

	if cellWithDir.IsValid() {
		t.Fatal("cell-mode index with direction bits should not be a valid edge")
	}
}

// TestDirectedEdgeStringRoundTrip covers the string, marshal and unmarshal
// helpers.
func TestDirectedEdgeStringRoundTrip(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	edges, err := origin.DirectedEdges()
	if err != nil {
		t.Fatalf("DirectedEdges: %v", err)
	}

	edge := edges[0]

	parsed := DirectedEdgeFromString(edge.String())
	if parsed != edge {
		t.Fatalf("DirectedEdgeFromString round trip: got %015x, want %015x", uint64(parsed), uint64(edge))
	}

	text, err := edge.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}

	var unmarshaled DirectedEdge
	if err := unmarshaled.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}

	if unmarshaled != edge {
		t.Fatalf("UnmarshalText: got %015x, want %015x", uint64(unmarshaled), uint64(edge))
	}

	if err := unmarshaled.UnmarshalText([]byte("not-an-edge")); err == nil {
		t.Fatal("UnmarshalText(invalid): got nil error, want failure")
	}

	if _, err := edge.IndexDigit(1); err != nil {
		t.Fatalf("IndexDigit(1): %v", err)
	}

	if _, err := edge.IndexDigit(0); !errors.Is(err, ErrResolutionDomain) {
		t.Fatalf("IndexDigit(0): want ErrResolutionDomain")
	}
}

// TestIsValidIndexDispatch covers the mode dispatch of the generic validity
// check, including the default (unknown mode) branch.
func TestIsValidIndexDispatch(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")
	if !IsValidIndex(origin) {
		t.Fatal("valid cell should pass IsValidIndex")
	}

	edge, err := origin.DirectedEdge(mustNeighbor(t, origin))
	if err != nil {
		t.Fatalf("DirectedEdge: %v", err)
	}

	if !IsValidIndex(edge) {
		t.Fatal("valid edge should pass IsValidIndex")
	}

	vertex, err := origin.Vertex(0)
	if err != nil {
		t.Fatalf("Vertex: %v", err)
	}

	if !IsValidIndex(vertex) {
		t.Fatal("valid vertex should pass IsValidIndex")
	}

	// Mode 0 (reserved) is not a valid index of any kind.
	unknown := origin.setMode(0)
	if IsValidIndex(unknown) {
		t.Fatal("unknown-mode index should fail IsValidIndex")
	}
}

// mustNeighbor returns a neighbor of origin from its grid disk.
func mustNeighbor(t *testing.T, origin Cell) Cell {
	t.Helper()

	disk, err := origin.GridDisk(1)
	if err != nil {
		t.Fatalf("GridDisk: %v", err)
	}

	for _, cell := range disk {
		if cell != origin {
			return cell
		}
	}

	t.Fatal("no neighbor found")

	return 0
}
