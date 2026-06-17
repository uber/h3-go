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

package paritytest

import (
	"math"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// vertexNumRange is the range of vertex numbers tried against every cell. It
// covers the six hexagon vertexes plus an out-of-range value for error parity.
const vertexNumRange = 7

// TestCellToVertexMatchesCgo asserts CellToVertex matches the cgo reference for
// every vertex number across the corpus, including error parity.
func TestCellToVertexMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		for vertexNum := range vertexNumRange {
			want, wantErr := h3.CellToVertex(origin, vertexNum)
			got, gotErr := h3go.CellToVertex(h3goCell(origin), vertexNum)

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("CellToVertex(%015x, %d) error mismatch: cgo=%v h3go=%v", uint64(origin), vertexNum, wantErr, gotErr)
			}

			if wantErr == nil && uint64(got) != uint64(want) {
				t.Fatalf("CellToVertex(%015x, %d): got %015x, want %015x", uint64(origin), vertexNum, uint64(got), uint64(want))
			}
		}
	}
}

// TestCellToVertexesMatchesCgo asserts the full set of cell vertexes matches the
// cgo reference, comparing as sets of non-zero indexes (the cgo reference pads a
// pentagon's missing sixth vertex with H3_NULL).
func TestCellToVertexesMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		want, wantErr := h3.CellToVertexes(origin)
		got, gotErr := h3go.CellToVertexes(h3goCell(origin))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("CellToVertexes(%015x) error mismatch: cgo=%v h3go=%v", uint64(origin), wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		assertSameVertexSet(t, got, want, uint64(origin))
	}
}

// TestVertexToLatLngMatchesCgo asserts the geographic coordinates of each vertex
// match the cgo reference.
func TestVertexToLatLngMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		vertexes, err := h3.CellToVertexes(origin)
		if err != nil {
			continue
		}

		for _, vertex := range vertexes {
			if vertex == 0 {
				continue
			}

			want, wantErr := h3.VertexToLatLng(vertex)
			got, gotErr := h3go.VertexToLatLng(h3go.Vertex(vertex))

			if !bothErr(wantErr, gotErr) {
				t.Fatalf("VertexToLatLng(%015x) error mismatch: cgo=%v h3go=%v", uint64(vertex), wantErr, gotErr)
			}

			if wantErr != nil {
				continue
			}

			if math.Abs(got.Lat-want.Lat) > 1e-9 || math.Abs(got.Lng-want.Lng) > 1e-9 {
				t.Fatalf("VertexToLatLng(%015x): got {%v,%v}, want {%v,%v}", uint64(vertex), got.Lat, got.Lng, want.Lat, want.Lng)
			}
		}
	}
}

// TestIsValidVertexMatchesCgo asserts vertex validity matches the cgo reference
// for real vertexes and for cells reinterpreted as vertexes.
func TestIsValidVertexMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		if h3.IsValidVertex(h3.Vertex(origin)) != h3go.IsValidVertex(h3go.Vertex(origin)) {
			t.Fatalf("IsValidVertex(cell %015x) mismatch", uint64(origin))
		}

		vertexes, err := h3.CellToVertexes(origin)
		if err != nil {
			continue
		}

		for _, vertex := range vertexes {
			if h3.IsValidVertex(vertex) != h3go.IsValidVertex(h3go.Vertex(vertex)) {
				t.Fatalf("IsValidVertex(%015x) mismatch", uint64(vertex))
			}
		}
	}
}

// assertSameVertexSet fails if the two vertex slices differ as sets of non-zero
// indexes.
func assertSameVertexSet(t *testing.T, got []h3go.Vertex, want []h3.Vertex, origin uint64) {
	t.Helper()

	set := make(map[uint64]bool)

	for _, vertex := range want {
		if vertex != 0 {
			set[uint64(vertex)] = true
		}
	}

	if len(got) != len(set) {
		t.Fatalf("CellToVertexes(%015x): len got=%d want=%d", origin, len(got), len(set))
	}

	for _, vertex := range got {
		if !set[uint64(vertex)] {
			t.Fatalf("CellToVertexes(%015x): unexpected vertex %015x", origin, uint64(vertex))
		}
	}
}

// edgePairs returns origin/neighbor cell pairs from the corpus by pairing each
// corpus cell with the cells in its grid disk.
func edgePairs(t *testing.T) [][2]h3.Cell {
	t.Helper()

	var pairs [][2]h3.Cell

	for _, origin := range referenceCorpus(t) {
		disk, err := h3.GridDisk(origin, 1)
		if err != nil {
			continue
		}

		for _, neighbor := range disk {
			if neighbor == 0 || neighbor == origin {
				continue
			}

			pairs = append(pairs, [2]h3.Cell{origin, neighbor})
		}
	}

	return pairs
}

// TestCellsToDirectedEdgeMatchesCgo asserts edge construction matches the cgo
// reference, including the not-neighbors error path.
func TestCellsToDirectedEdgeMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, pair := range edgePairs(t) {
		origin, neighbor := pair[0], pair[1]

		want, wantErr := origin.DirectedEdge(neighbor)
		got, gotErr := h3goCell(origin).DirectedEdge(h3goCell(neighbor))

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("DirectedEdge(%015x, %015x) error mismatch: cgo=%v h3go=%v", uint64(origin), uint64(neighbor), wantErr, gotErr)
		}

		if wantErr == nil && uint64(got) != uint64(want) {
			t.Fatalf("DirectedEdge(%015x, %015x): got %015x, want %015x", uint64(origin), uint64(neighbor), uint64(got), uint64(want))
		}
	}

	// Non-neighbor cells return the not-neighbors error.
	far := h3.CellFromString("85283473fffffff")
	farSibling := h3.CellFromString("85f29263fffffff")

	_, wantErr := far.DirectedEdge(farSibling)
	_, gotErr := h3goCell(far).DirectedEdge(h3goCell(farSibling))

	if !bothErr(wantErr, gotErr) {
		t.Fatalf("DirectedEdge(non-neighbors) error mismatch: cgo=%v h3go=%v", wantErr, gotErr)
	}
}

// TestOriginToDirectedEdgesMatchesCgo asserts the set of edges from each cell
// matches the cgo reference, compared as sets of non-zero indexes.
func TestOriginToDirectedEdgesMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		want, wantErr := origin.DirectedEdges()
		got, gotErr := h3goCell(origin).DirectedEdges()

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("DirectedEdges(%015x) error mismatch: cgo=%v h3go=%v", uint64(origin), wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		set := make(map[uint64]bool)

		for _, edge := range want {
			if edge != 0 {
				set[uint64(edge)] = true
			}
		}

		if len(got) != len(set) {
			t.Fatalf("DirectedEdges(%015x): len got=%d want=%d", uint64(origin), len(got), len(set))
		}

		for _, edge := range got {
			if !set[uint64(edge)] {
				t.Fatalf("DirectedEdges(%015x): unexpected edge %015x", uint64(origin), uint64(edge))
			}
		}
	}
}

// TestDirectedEdgeAccessorsMatchCgo asserts the edge accessors and edge length
// match the cgo reference over all edges derived from the corpus.
func TestDirectedEdgeAccessorsMatchCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		edges, err := origin.DirectedEdges()
		if err != nil {
			continue
		}

		for _, edge := range edges {
			if edge == 0 {
				continue
			}

			goEdge := h3go.DirectedEdge(edge)

			if edge.IsValid() != goEdge.IsValid() {
				t.Fatalf("IsValid(%015x) mismatch", uint64(edge))
			}

			assertEdgeCellsMatch(t, edge, goEdge)
			assertEdgeReverseMatches(t, edge, goEdge)
			assertEdgeBoundaryMatches(t, edge, goEdge)
			assertEdgeLengthMatches(t, edge, goEdge)
		}
	}
}

// assertEdgeCellsMatch checks Origin, Destination and Cells against the cgo
// reference.
func assertEdgeCellsMatch(t *testing.T, edge h3.DirectedEdge, goEdge h3go.DirectedEdge) {
	t.Helper()

	wantOrigin, _ := edge.Origin()

	gotOrigin, err := goEdge.Origin()
	if err != nil || uint64(gotOrigin) != uint64(wantOrigin) {
		t.Fatalf("Origin(%015x): got %015x (%v), want %015x", uint64(edge), uint64(gotOrigin), err, uint64(wantOrigin))
	}

	wantDest, _ := edge.Destination()

	gotDest, err := goEdge.Destination()
	if err != nil || uint64(gotDest) != uint64(wantDest) {
		t.Fatalf("Destination(%015x): got %015x (%v), want %015x", uint64(edge), uint64(gotDest), err, uint64(wantDest))
	}

	wantCells, _ := edge.Cells()

	gotCells, err := goEdge.Cells()
	if err != nil || len(gotCells) != len(wantCells) {
		t.Fatalf("Cells(%015x): got %v (%v), want %v", uint64(edge), gotCells, err, wantCells)
	}

	for i := range wantCells {
		if uint64(gotCells[i]) != uint64(wantCells[i]) {
			t.Fatalf("Cells(%015x)[%d]: got %015x, want %015x", uint64(edge), i, uint64(gotCells[i]), uint64(wantCells[i]))
		}
	}
}

// assertEdgeReverseMatches checks Reverse against the cgo reference.
func assertEdgeReverseMatches(t *testing.T, edge h3.DirectedEdge, goEdge h3go.DirectedEdge) {
	t.Helper()

	want, _ := edge.Reverse()

	got, err := goEdge.Reverse()
	if err != nil || uint64(got) != uint64(want) {
		t.Fatalf("Reverse(%015x): got %015x (%v), want %015x", uint64(edge), uint64(got), err, uint64(want))
	}
}

// assertEdgeBoundaryMatches checks Boundary against the cgo reference.
func assertEdgeBoundaryMatches(t *testing.T, edge h3.DirectedEdge, goEdge h3go.DirectedEdge) {
	t.Helper()

	want, _ := edge.Boundary()

	got, err := goEdge.Boundary()
	if err != nil || len(got) != len(want) {
		t.Fatalf("Boundary(%015x): got len %d (%v), want len %d", uint64(edge), len(got), err, len(want))
	}

	for i := range want {
		if math.Abs(got[i].Lat-want[i].Lat) > 1e-9 || math.Abs(got[i].Lng-want[i].Lng) > 1e-9 {
			t.Fatalf("Boundary(%015x)[%d]: got {%v,%v}, want {%v,%v}", uint64(edge), i, got[i].Lat, got[i].Lng, want[i].Lat, want[i].Lng)
		}
	}
}

// assertEdgeLengthMatches checks the three edge length units against the cgo
// reference.
func assertEdgeLengthMatches(t *testing.T, edge h3.DirectedEdge, goEdge h3go.DirectedEdge) {
	t.Helper()

	wantRads, _ := h3.EdgeLengthRads(edge)

	gotRads, err := h3go.EdgeLengthRads(goEdge)
	if err != nil || math.Abs(gotRads-wantRads) > 1e-12 {
		t.Fatalf("EdgeLengthRads(%015x): got %v (%v), want %v", uint64(edge), gotRads, err, wantRads)
	}

	wantKm, _ := h3.EdgeLengthKm(edge)

	gotKm, _ := h3go.EdgeLengthKm(goEdge)
	if math.Abs(gotKm-wantKm) > 1e-9 {
		t.Fatalf("EdgeLengthKm(%015x): got %v, want %v", uint64(edge), gotKm, wantKm)
	}

	wantM, _ := h3.EdgeLengthM(edge)

	gotM, _ := h3go.EdgeLengthM(goEdge)
	if math.Abs(gotM-wantM) > 1e-6 {
		t.Fatalf("EdgeLengthM(%015x): got %v, want %v", uint64(edge), gotM, wantM)
	}
}

// TestIsValidIndexMatchesCgo asserts the generic index validity check matches the
// cgo reference for cells, edges and vertexes.
func TestIsValidIndexMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, origin := range referenceCorpus(t) {
		if h3.IsValidIndex(origin) != h3go.IsValidIndex(h3goCell(origin)) {
			t.Fatalf("IsValidIndex(cell %015x) mismatch", uint64(origin))
		}

		edges, err := origin.DirectedEdges()
		if err == nil {
			for _, edge := range edges {
				if edge == 0 {
					continue
				}

				if h3.IsValidIndex(edge) != h3go.IsValidIndex(h3go.DirectedEdge(edge)) {
					t.Fatalf("IsValidIndex(edge %015x) mismatch", uint64(edge))
				}
			}
		}

		vertexes, err := h3.CellToVertexes(origin)
		if err != nil {
			continue
		}

		for _, vertex := range vertexes {
			if vertex == 0 {
				continue
			}

			if h3.IsValidIndex(vertex) != h3go.IsValidIndex(h3go.Vertex(vertex)) {
				t.Fatalf("IsValidIndex(vertex %015x) mismatch", uint64(vertex))
			}
		}
	}
}
