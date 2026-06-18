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

// This file holds the shared fixtures and harness for the cross-implementation
// benchmarks in bench_test.go. Every public symbol is benchmarked twice, once
// against the cgo-backed github.com/uber/h3-go/v4 reference and once against the
// pure-Go x/h3go package, under sub-benchmarks named "impl=cgo" and "impl=go".
// Running benchstat with -col /impl over the output pivots the two into a single
// delta column per method.

package paritytest

import (
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// benchRes is the working resolution for the single-cell fixtures. Resolution 9
// is fine enough to exercise the digit chain without producing unwieldy child or
// disk sets.
const benchRes = 9

// benchPolyRes is the fill resolution for the polygon fixtures, kept coarse so
// PolygonToCells returns in a reasonable time per iteration.
const benchPolyRes = 8

var (
	// Geographic inputs. The two points sit a few kilometres apart in the San
	// Francisco Bay Area so distance and path fixtures stay well-defined.
	benchGeo    = h3.LatLng{Lat: 37.7749, Lng: -122.4194}
	benchGeo2   = h3.LatLng{Lat: 37.3382, Lng: -121.8863}
	benchGoGeo  = h3go.LatLng{Lat: benchGeo.Lat, Lng: benchGeo.Lng}
	benchGoGeo2 = h3go.LatLng{Lat: benchGeo2.Lat, Lng: benchGeo2.Lng}

	// Single-cell fixtures.
	benchCell   = mustCell(h3.LatLngToCell(benchGeo, benchRes))
	benchGoCell = h3goCell(benchCell)

	// An immediate neighbour, used for directed edges, neighbour checks and
	// local IJ coordinates (which are only defined near the origin).
	benchNeighbor   = adjacentCell(benchCell)
	benchGoNeighbor = h3goCell(benchNeighbor)

	// A cell five rings out, giving grid distance and path a non-trivial span.
	benchTarget   = ringCell(benchCell, 5)
	benchGoTarget = h3goCell(benchTarget)

	// Directed edge and vertex fixtures derived from the working cell.
	benchEdge   = mustEdge(benchCell.DirectedEdge(benchNeighbor))
	benchGoEdge = h3go.DirectedEdge(benchEdge)

	benchVertex   = mustVertex(h3.CellToVertex(benchCell, 0))
	benchGoVertex = h3go.Vertex(benchVertex)

	// String fixtures for the parse paths.
	benchCellStr   = benchCell.String()
	benchEdgeStr   = benchEdge.String()
	benchVertexStr = benchVertex.String()

	// Hierarchy fixtures: a coarse parent and its full child set, which compacts
	// back to the parent and uncompacts back to the children.
	benchParent        = mustCell(benchCell.Parent(6))
	benchChildren      = mustCells(benchParent.Children(benchRes))
	benchGoChildren    = toGoCells(benchChildren)
	benchUncompactIn   = []h3.Cell{benchParent}
	benchGoUncompactIn = toGoCells(benchUncompactIn)

	// A filled grid disk, reused as the input set for batch disk traversal and
	// the contiguous region passed to CellsToMultiPolygon.
	benchDisk   = mustCells(h3.GridDisk(benchCell, 4))
	benchGoDisk = toGoCells(benchDisk)

	// Local IJ coordinates of the neighbour relative to the working cell.
	benchLocalIJ   = mustIJ(h3.CellToLocalIJ(benchCell, benchNeighbor))
	benchGoLocalIJ = h3go.CoordIJ{I: benchLocalIJ.I, J: benchLocalIJ.J}

	// A polygon with a hole, shared from the parity-test corpus.
	benchPolygon   = regionPolygons()["with_hole"]
	benchGoPolygon = toGoPolygon(benchPolygon)
)

// blackhole is the shared sink that keeps the compiler from eliminating the
// benchmarked calls. Sub-benchmarks run sequentially, so a single sink is safe.
var blackhole any

// compare runs cgoFn and goFn as the "impl=cgo" and "impl=go" sub-benchmarks so
// benchstat -col /impl can report the delta between the two implementations.
func compare(b *testing.B, cgoFn, goFn func()) {
	b.Helper()

	b.Run("impl=cgo", func(b *testing.B) {
		for b.Loop() {
			cgoFn()
		}
	})

	b.Run("impl=go", func(b *testing.B) {
		for b.Loop() {
			goFn()
		}
	})
}

// mustCell returns the cell or panics, for fixture construction only.
func mustCell(cell h3.Cell, err error) h3.Cell {
	if err != nil {
		panic(err)
	}

	return cell
}

// mustCells returns the cell slice or panics, for fixture construction only.
func mustCells(cells []h3.Cell, err error) []h3.Cell {
	if err != nil {
		panic(err)
	}

	return cells
}

// mustEdge returns the directed edge or panics, for fixture construction only.
func mustEdge(edge h3.DirectedEdge, err error) h3.DirectedEdge {
	if err != nil {
		panic(err)
	}

	return edge
}

// mustVertex returns the vertex or panics, for fixture construction only.
func mustVertex(vertex h3.Vertex, err error) h3.Vertex {
	if err != nil {
		panic(err)
	}

	return vertex
}

// mustIJ returns the IJ coordinate or panics, for fixture construction only.
func mustIJ(ij h3.CoordIJ, err error) h3.CoordIJ {
	if err != nil {
		panic(err)
	}

	return ij
}

// adjacentCell returns an immediate neighbour of origin.
func adjacentCell(origin h3.Cell) h3.Cell {
	disk, err := h3.GridDisk(origin, 1)
	if err != nil {
		panic(err)
	}

	for _, candidate := range disk {
		if candidate != 0 && candidate != origin {
			return candidate
		}
	}

	panic("no adjacent cell found")
}

// ringCell returns a cell on the k-ring around origin.
func ringCell(origin h3.Cell, k int) h3.Cell {
	ring, err := h3.GridRing(origin, k)
	if err != nil {
		panic(err)
	}

	for _, candidate := range ring {
		if candidate != 0 {
			return candidate
		}
	}

	panic("empty ring")
}
