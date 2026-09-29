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

// Package paritytest's benchmarks compare the cgo reference against the pure-Go
// x/h3go package for every publicly exposed function and method. Each benchmark
// runs both implementations as "impl=cgo" and "impl=go" sub-benchmarks; pivot
// with `benchstat -col /impl` to read the per-method delta.
package paritytest

import (
	"fmt"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// ---------------------------------------------------------------------------
// Conversion: lat/lng <-> cell, string <-> index
// ---------------------------------------------------------------------------

func BenchmarkLatLngToCell(b *testing.B) {
	compare(b,
		func() { out, _ := h3.LatLngToCell(benchGeo, benchRes); sink(out) },
		func() { out, _ := h3go.LatLngToCell(benchGoGeo, benchRes); sink(out) },
	)
}

func BenchmarkLatLngToCellString(b *testing.B) {
	compare(b,
		func() { out, _ := h3.LatLngToCellString(benchGeo.Lat, benchGeo.Lng, benchRes); sink(out) },
		func() { out, _ := h3go.LatLngToCellString(benchGoGeo.Lat, benchGoGeo.Lng, benchRes); sink(out) },
	)
}

func BenchmarkLatLngCell(b *testing.B) {
	compare(b,
		func() { out, _ := benchGeo.Cell(benchRes); sink(out) },
		func() { out, _ := benchGoGeo.Cell(benchRes); sink(out) },
	)
}

func BenchmarkNewLatLng(b *testing.B) {
	compare(b,
		func() { sink(h3.NewLatLng(benchGeo.Lat, benchGeo.Lng)) },
		func() { sink(h3go.NewLatLng(benchGoGeo.Lat, benchGoGeo.Lng)) },
	)
}

func BenchmarkLatLngString(b *testing.B) {
	compare(b,
		func() { sink(benchGeo.String()) },
		func() { sink(benchGoGeo.String()) },
	)
}

func BenchmarkCellToLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToLatLng(benchCell); sink(out) },
		func() { out, _ := h3go.CellToLatLng(benchGoCell); sink(out) },
	)
}

func BenchmarkCellLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.LatLng(); sink(out) },
		func() { out, _ := benchGoCell.LatLng(); sink(out) },
	)
}

func BenchmarkCellToString(b *testing.B) {
	compare(b,
		func() { sink(h3.CellToString(benchCell)) },
		func() { sink(h3go.CellToString(benchGoCell)) },
	)
}

func BenchmarkCellString(b *testing.B) {
	compare(b,
		func() { sink(benchCell.String()) },
		func() { sink(benchGoCell.String()) },
	)
}

func BenchmarkCellMarshalText(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.MarshalText(); sink(out) },
		func() { out, _ := benchGoCell.MarshalText(); sink(out) },
	)
}

func BenchmarkCellFromString(b *testing.B) {
	compare(b,
		func() { sink(h3.CellFromString(benchCellStr)) },
		func() { sink(h3go.CellFromString(benchCellStr)) },
	)
}

func BenchmarkIndexFromString(b *testing.B) {
	compare(b,
		func() { sink(h3.IndexFromString(benchCellStr)) },
		func() { sink(h3go.IndexFromString(benchCellStr)) },
	)
}

func BenchmarkIndexToString(b *testing.B) {
	compare(b,
		func() { sink(h3.IndexToString(uint64(benchCell))) },
		func() { sink(h3go.IndexToString(uint64(benchGoCell))) },
	)
}

// ---------------------------------------------------------------------------
// Index introspection
// ---------------------------------------------------------------------------

func BenchmarkResolution(b *testing.B) {
	compare(b,
		func() { sink(benchCell.Resolution()) },
		func() { sink(benchGoCell.Resolution()) },
	)
}

func BenchmarkBaseCellNumberFunc(b *testing.B) {
	compare(b,
		func() { sink(h3.BaseCellNumber(benchCell)) },
		func() { sink(h3go.BaseCellNumber(benchGoCell)) },
	)
}

func BenchmarkBaseCellNumber(b *testing.B) {
	compare(b,
		func() { sink(benchCell.BaseCellNumber()) },
		func() { sink(benchGoCell.BaseCellNumber()) },
	)
}

func BenchmarkIsValid(b *testing.B) {
	compare(b,
		func() { sink(benchCell.IsValid()) },
		func() { sink(benchGoCell.IsValid()) },
	)
}

func BenchmarkIsPentagon(b *testing.B) {
	compare(b,
		func() { sink(benchCell.IsPentagon()) },
		func() { sink(benchGoCell.IsPentagon()) },
	)
}

func BenchmarkIsResClassIII(b *testing.B) {
	compare(b,
		func() { sink(benchCell.IsResClassIII()) },
		func() { sink(benchGoCell.IsResClassIII()) },
	)
}

func BenchmarkCellIndexDigit(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.IndexDigit(benchRes); sink(out) },
		func() { out, _ := benchGoCell.IndexDigit(benchRes); sink(out) },
	)
}

func BenchmarkNumCells(b *testing.B) {
	compare(b,
		func() { sink(h3.NumCells(benchRes)) },
		func() { sink(h3go.NumCells(benchRes)) },
	)
}

func BenchmarkRes0Cells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.Res0Cells(); sink(out) },
		func() { out, _ := h3go.Res0Cells(); sink(out) },
	)
}

func BenchmarkPentagons(b *testing.B) {
	compare(b,
		func() { out, _ := h3.Pentagons(benchRes); sink(out) },
		func() { out, _ := h3go.Pentagons(benchRes); sink(out) },
	)
}

// ---------------------------------------------------------------------------
// Hierarchy
// ---------------------------------------------------------------------------

func BenchmarkParent(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Parent(6); sink(out) },
		func() { out, _ := benchGoCell.Parent(6); sink(out) },
	)
}

func BenchmarkImmediateParent(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.ImmediateParent(); sink(out) },
		func() { out, _ := benchGoCell.ImmediateParent(); sink(out) },
	)
}

func BenchmarkChildren(b *testing.B) {
	compare(b,
		func() { out, _ := benchParent.Children(benchRes); sink(out) },
		func() { out, _ := h3goCell(benchParent).Children(benchRes); sink(out) },
	)
}

func BenchmarkImmediateChildren(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.ImmediateChildren(); sink(out) },
		func() { out, _ := benchGoCell.ImmediateChildren(); sink(out) },
	)
}

func BenchmarkCenterChild(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.CenterChild(benchRes + 2); sink(out) },
		func() { out, _ := benchGoCell.CenterChild(benchRes + 2); sink(out) },
	)
}

func BenchmarkCellToChildPos(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToChildPos(benchCell, 6); sink(out) },
		func() { out, _ := h3go.CellToChildPos(benchGoCell, 6); sink(out) },
	)
}

func BenchmarkChildPos(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.ChildPos(6); sink(out) },
		func() { out, _ := benchGoCell.ChildPos(6); sink(out) },
	)
}

func BenchmarkChildPosToCellFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.ChildPosToCell(0, benchParent, benchRes); sink(out) },
		func() { out, _ := h3go.ChildPosToCell(0, h3goCell(benchParent), benchRes); sink(out) },
	)
}

func BenchmarkChildPosToCell(b *testing.B) {
	compare(b,
		func() { out, _ := benchParent.ChildPosToCell(0, benchRes); sink(out) },
		func() { out, _ := h3goCell(benchParent).ChildPosToCell(0, benchRes); sink(out) },
	)
}

func BenchmarkCompactCells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CompactCells(benchChildren); sink(out) },
		func() { out, _ := h3go.CompactCells(benchGoChildren); sink(out) },
	)
}

func BenchmarkUncompactCells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.UncompactCells(benchUncompactIn, benchRes); sink(out) },
		func() { out, _ := h3go.UncompactCells(benchGoUncompactIn, benchRes); sink(out) },
	)
}

// ---------------------------------------------------------------------------
// Boundary
// ---------------------------------------------------------------------------

func BenchmarkCellToBoundary(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToBoundary(benchCell); sink(out) },
		func() { out, _ := h3go.CellToBoundary(benchGoCell); sink(out) },
	)
}

func BenchmarkCellBoundary(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Boundary(); sink(out) },
		func() { out, _ := benchGoCell.Boundary(); sink(out) },
	)
}

// ---------------------------------------------------------------------------
// Measures: area, edge length, great-circle distance
// ---------------------------------------------------------------------------

func BenchmarkCellAreaRads2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellAreaRads2(benchCell); sink(out) },
		func() { out, _ := h3go.CellAreaRads2(benchGoCell); sink(out) },
	)
}

func BenchmarkCellAreaKm2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellAreaKm2(benchCell); sink(out) },
		func() { out, _ := h3go.CellAreaKm2(benchGoCell); sink(out) },
	)
}

func BenchmarkCellAreaM2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellAreaM2(benchCell); sink(out) },
		func() { out, _ := h3go.CellAreaM2(benchGoCell); sink(out) },
	)
}

func BenchmarkHexagonAreaAvgKm2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonAreaAvgKm2(benchRes); sink(out) },
		func() { out, _ := h3go.HexagonAreaAvgKm2(benchRes); sink(out) },
	)
}

func BenchmarkHexagonAreaAvgM2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonAreaAvgM2(benchRes); sink(out) },
		func() { out, _ := h3go.HexagonAreaAvgM2(benchRes); sink(out) },
	)
}

func BenchmarkEdgeLengthRads(b *testing.B) {
	compare(b,
		func() { out, _ := h3.EdgeLengthRads(benchEdge); sink(out) },
		func() { out, _ := h3go.EdgeLengthRads(benchGoEdge); sink(out) },
	)
}

func BenchmarkEdgeLengthKm(b *testing.B) {
	compare(b,
		func() { out, _ := h3.EdgeLengthKm(benchEdge); sink(out) },
		func() { out, _ := h3go.EdgeLengthKm(benchGoEdge); sink(out) },
	)
}

func BenchmarkEdgeLengthM(b *testing.B) {
	compare(b,
		func() { out, _ := h3.EdgeLengthM(benchEdge); sink(out) },
		func() { out, _ := h3go.EdgeLengthM(benchGoEdge); sink(out) },
	)
}

func BenchmarkHexagonEdgeLengthAvgKm(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonEdgeLengthAvgKm(benchRes); sink(out) },
		func() { out, _ := h3go.HexagonEdgeLengthAvgKm(benchRes); sink(out) },
	)
}

func BenchmarkHexagonEdgeLengthAvgM(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonEdgeLengthAvgM(benchRes); sink(out) },
		func() { out, _ := h3go.HexagonEdgeLengthAvgM(benchRes); sink(out) },
	)
}

func BenchmarkGreatCircleDistanceRads(b *testing.B) {
	compare(b,
		func() { sink(h3.GreatCircleDistanceRads(benchGeo, benchGeo2)) },
		func() { sink(h3go.GreatCircleDistanceRads(benchGoGeo, benchGoGeo2)) },
	)
}

func BenchmarkGreatCircleDistanceKm(b *testing.B) {
	compare(b,
		func() { sink(h3.GreatCircleDistanceKm(benchGeo, benchGeo2)) },
		func() { sink(h3go.GreatCircleDistanceKm(benchGoGeo, benchGoGeo2)) },
	)
}

func BenchmarkGreatCircleDistanceM(b *testing.B) {
	compare(b,
		func() { sink(h3.GreatCircleDistanceM(benchGeo, benchGeo2)) },
		func() { sink(h3go.GreatCircleDistanceM(benchGoGeo, benchGoGeo2)) },
	)
}

// ---------------------------------------------------------------------------
// Grid traversal
// ---------------------------------------------------------------------------

func BenchmarkGridDiskFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDisk(benchCell, 5); sink(out) },
		func() { out, _ := h3go.GridDisk(benchGoCell, 5); sink(out) },
	)
}

func BenchmarkGridDisk(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDisk(5); sink(out) },
		func() { out, _ := benchGoCell.GridDisk(5); sink(out) },
	)
}

func BenchmarkGridDiskDistancesFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDiskDistances(benchCell, 5); sink(out) },
		func() { out, _ := h3go.GridDiskDistances(benchGoCell, 5); sink(out) },
	)
}

func BenchmarkGridDiskDistances(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDiskDistances(5); sink(out) },
		func() { out, _ := benchGoCell.GridDiskDistances(5); sink(out) },
	)
}

func BenchmarkGridDiskDistancesSafeFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDiskDistancesSafe(benchCell, 5); sink(out) },
		func() { out, _ := h3go.GridDiskDistancesSafe(benchGoCell, 5); sink(out) },
	)
}

func BenchmarkGridDiskDistancesSafe(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDiskDistancesSafe(5); sink(out) },
		func() { out, _ := benchGoCell.GridDiskDistancesSafe(5); sink(out) },
	)
}

func BenchmarkGridDiskDistancesUnsafeFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDiskDistancesUnsafe(benchCell, 5); sink(out) },
		func() { out, _ := h3go.GridDiskDistancesUnsafe(benchGoCell, 5); sink(out) },
	)
}

func BenchmarkGridDiskDistancesUnsafe(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDiskDistancesUnsafe(5); sink(out) },
		func() { out, _ := benchGoCell.GridDiskDistancesUnsafe(5); sink(out) },
	)
}

func BenchmarkGridRingFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridRing(benchCell, 5); sink(out) },
		func() { out, _ := h3go.GridRing(benchGoCell, 5); sink(out) },
	)
}

func BenchmarkGridRing(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridRing(5); sink(out) },
		func() { out, _ := benchGoCell.GridRing(5); sink(out) },
	)
}

func BenchmarkGridRingUnsafeFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridRingUnsafe(benchCell, 5); sink(out) },
		func() { out, _ := h3go.GridRingUnsafe(benchGoCell, 5); sink(out) },
	)
}

func BenchmarkGridRingUnsafe(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridRingUnsafe(5); sink(out) },
		func() { out, _ := benchGoCell.GridRingUnsafe(5); sink(out) },
	)
}

// BenchmarkGridDiskPentagon measures the safe breadth-first fallback that
// GridDisk takes when the origin is a pentagon, across increasing radii.
func BenchmarkGridDiskPentagon(b *testing.B) {
	for _, k := range []int{10, 20, 30, 40} {
		b.Run(fmt.Sprintf("k=%d", k), func(b *testing.B) {
			compare(b,
				func() { out, _ := h3.GridDisk(benchPentagon, k); sink(out) },
				func() { out, _ := h3go.GridDisk(benchGoPentagon, k); sink(out) },
			)
		})
	}
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDisksUnsafe(benchDisk, 3); sink(out) },
		func() { out, _ := h3go.GridDisksUnsafe(benchGoDisk, 3); sink(out) },
	)
}

func BenchmarkGridDistanceFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDistance(benchCell, benchTarget); sink(out) },
		func() { out, _ := h3go.GridDistance(benchGoCell, benchGoTarget); sink(out) },
	)
}

func BenchmarkGridDistance(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDistance(benchTarget); sink(out) },
		func() { out, _ := benchGoCell.GridDistance(benchGoTarget); sink(out) },
	)
}

func BenchmarkGridPathFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridPath(benchCell, benchTarget); sink(out) },
		func() { out, _ := h3go.GridPath(benchGoCell, benchGoTarget); sink(out) },
	)
}

func BenchmarkGridPath(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridPath(benchTarget); sink(out) },
		func() { out, _ := benchGoCell.GridPath(benchGoTarget); sink(out) },
	)
}

func BenchmarkIsNeighbor(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.IsNeighbor(benchNeighbor); sink(out) },
		func() { out, _ := benchGoCell.IsNeighbor(benchGoNeighbor); sink(out) },
	)
}

// ---------------------------------------------------------------------------
// Local IJ coordinates
// ---------------------------------------------------------------------------

func BenchmarkCellToLocalIJ(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToLocalIJ(benchCell, benchNeighbor); sink(out) },
		func() { out, _ := h3go.CellToLocalIJ(benchGoCell, benchGoNeighbor); sink(out) },
	)
}

func BenchmarkLocalIJToCell(b *testing.B) {
	compare(b,
		func() { out, _ := h3.LocalIJToCell(benchCell, benchLocalIJ); sink(out) },
		func() { out, _ := h3go.LocalIJToCell(benchGoCell, benchGoLocalIJ); sink(out) },
	)
}

// ---------------------------------------------------------------------------
// Icosahedron faces
// ---------------------------------------------------------------------------

func BenchmarkIcosahedronFaces(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.IcosahedronFaces(); sink(out) },
		func() { out, _ := benchGoCell.IcosahedronFaces(); sink(out) },
	)
}

// ---------------------------------------------------------------------------
// Vertexes
// ---------------------------------------------------------------------------

func BenchmarkCellToVertex(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToVertex(benchCell, 0); sink(out) },
		func() { out, _ := h3go.CellToVertex(benchGoCell, 0); sink(out) },
	)
}

func BenchmarkCellVertex(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Vertex(0); sink(out) },
		func() { out, _ := benchGoCell.Vertex(0); sink(out) },
	)
}

func BenchmarkCellToVertexes(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToVertexes(benchCell); sink(out) },
		func() { out, _ := h3go.CellToVertexes(benchGoCell); sink(out) },
	)
}

func BenchmarkCellVertexes(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Vertexes(); sink(out) },
		func() { out, _ := benchGoCell.Vertexes(); sink(out) },
	)
}

func BenchmarkVertexToLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := h3.VertexToLatLng(benchVertex); sink(out) },
		func() { out, _ := h3go.VertexToLatLng(benchGoVertex); sink(out) },
	)
}

func BenchmarkVertexLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := benchVertex.LatLng(); sink(out) },
		func() { out, _ := benchGoVertex.LatLng(); sink(out) },
	)
}

func BenchmarkIsValidVertexFunc(b *testing.B) {
	compare(b,
		func() { sink(h3.IsValidVertex(benchVertex)) },
		func() { sink(h3go.IsValidVertex(benchGoVertex)) },
	)
}

func BenchmarkVertexIsValid(b *testing.B) {
	compare(b,
		func() { sink(benchVertex.IsValid()) },
		func() { sink(benchGoVertex.IsValid()) },
	)
}

func BenchmarkVertexResolution(b *testing.B) {
	compare(b,
		func() { sink(benchVertex.Resolution()) },
		func() { sink(benchGoVertex.Resolution()) },
	)
}

func BenchmarkVertexIndexDigit(b *testing.B) {
	compare(b,
		func() { out, _ := benchVertex.IndexDigit(benchRes); sink(out) },
		func() { out, _ := benchGoVertex.IndexDigit(benchRes); sink(out) },
	)
}

func BenchmarkVertexString(b *testing.B) {
	compare(b,
		func() { sink(benchVertex.String()) },
		func() { sink(benchGoVertex.String()) },
	)
}

func BenchmarkVertexMarshalText(b *testing.B) {
	compare(b,
		func() { out, _ := benchVertex.MarshalText(); sink(out) },
		func() { out, _ := benchGoVertex.MarshalText(); sink(out) },
	)
}

func BenchmarkVertexFromString(b *testing.B) {
	compare(b,
		func() { sink(h3.VertexFromString(benchVertexStr)) },
		func() { sink(h3go.VertexFromString(benchVertexStr)) },
	)
}

// ---------------------------------------------------------------------------
// Directed edges
// ---------------------------------------------------------------------------

func BenchmarkDirectedEdge(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.DirectedEdge(benchNeighbor); sink(out) },
		func() { out, _ := benchGoCell.DirectedEdge(benchGoNeighbor); sink(out) },
	)
}

func BenchmarkDirectedEdges(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.DirectedEdges(); sink(out) },
		func() { out, _ := benchGoCell.DirectedEdges(); sink(out) },
	)
}

func BenchmarkDirectedEdgeCells(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Cells(); sink(out) },
		func() { out, _ := benchGoEdge.Cells(); sink(out) },
	)
}

func BenchmarkDirectedEdgeOrigin(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Origin(); sink(out) },
		func() { out, _ := benchGoEdge.Origin(); sink(out) },
	)
}

func BenchmarkDirectedEdgeDestination(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Destination(); sink(out) },
		func() { out, _ := benchGoEdge.Destination(); sink(out) },
	)
}

func BenchmarkDirectedEdgeReverse(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Reverse(); sink(out) },
		func() { out, _ := benchGoEdge.Reverse(); sink(out) },
	)
}

func BenchmarkDirectedEdgeIsValid(b *testing.B) {
	compare(b,
		func() { sink(benchEdge.IsValid()) },
		func() { sink(benchGoEdge.IsValid()) },
	)
}

func BenchmarkDirectedEdgeResolution(b *testing.B) {
	compare(b,
		func() { sink(benchEdge.Resolution()) },
		func() { sink(benchGoEdge.Resolution()) },
	)
}

func BenchmarkDirectedEdgeIndexDigit(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.IndexDigit(benchRes); sink(out) },
		func() { out, _ := benchGoEdge.IndexDigit(benchRes); sink(out) },
	)
}

func BenchmarkDirectedEdgeBoundary(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Boundary(); sink(out) },
		func() { out, _ := benchGoEdge.Boundary(); sink(out) },
	)
}

func BenchmarkDirectedEdgeString(b *testing.B) {
	compare(b,
		func() { sink(benchEdge.String()) },
		func() { sink(benchGoEdge.String()) },
	)
}

func BenchmarkDirectedEdgeMarshalText(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.MarshalText(); sink(out) },
		func() { out, _ := benchGoEdge.MarshalText(); sink(out) },
	)
}

func BenchmarkDirectedEdgeFromString(b *testing.B) {
	compare(b,
		func() { sink(h3.DirectedEdgeFromString(benchEdgeStr)) },
		func() { sink(h3go.DirectedEdgeFromString(benchEdgeStr)) },
	)
}

// ---------------------------------------------------------------------------
// Regions: polygon <-> cells
// ---------------------------------------------------------------------------

func BenchmarkPolygonToCells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.PolygonToCells(benchPolygon, benchPolyRes); sink(out) },
		func() { out, _ := h3go.PolygonToCells(benchGoPolygon, benchPolyRes); sink(out) },
	)
}

func BenchmarkGeoPolygonCells(b *testing.B) {
	compare(b,
		func() { out, _ := benchPolygon.Cells(benchPolyRes); sink(out) },
		func() { out, _ := benchGoPolygon.Cells(benchPolyRes); sink(out) },
	)
}

func BenchmarkPolygonToCellsExperimental(b *testing.B) {
	compare(b,
		func() {
			out, _ := h3.PolygonToCellsExperimental(benchPolygon, benchPolyRes, h3.ContainmentCenter)
			sink(out)
		},
		func() {
			out, _ := h3go.PolygonToCellsExperimental(benchGoPolygon, benchPolyRes, h3go.ContainmentCenter)
			sink(out)
		},
	)
}

func BenchmarkCellsToMultiPolygon(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellsToMultiPolygon(benchDisk); sink(out) },
		func() { out, _ := h3go.CellsToMultiPolygon(benchGoDisk); sink(out) },
	)
}
