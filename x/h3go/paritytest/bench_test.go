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
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// ---------------------------------------------------------------------------
// Conversion: lat/lng <-> cell, string <-> index
// ---------------------------------------------------------------------------

func BenchmarkLatLngToCell(b *testing.B) {
	compare(b,
		func() { out, _ := h3.LatLngToCell(benchGeo, benchRes); blackhole = out },
		func() { out, _ := h3go.LatLngToCell(benchGoGeo, benchRes); blackhole = out },
	)
}

func BenchmarkLatLngToCellString(b *testing.B) {
	compare(b,
		func() { out, _ := h3.LatLngToCellString(benchGeo.Lat, benchGeo.Lng, benchRes); blackhole = out },
		func() { out, _ := h3go.LatLngToCellString(benchGoGeo.Lat, benchGoGeo.Lng, benchRes); blackhole = out },
	)
}

func BenchmarkLatLngCell(b *testing.B) {
	compare(b,
		func() { out, _ := benchGeo.Cell(benchRes); blackhole = out },
		func() { out, _ := benchGoGeo.Cell(benchRes); blackhole = out },
	)
}

func BenchmarkNewLatLng(b *testing.B) {
	compare(b,
		func() { blackhole = h3.NewLatLng(benchGeo.Lat, benchGeo.Lng) },
		func() { blackhole = h3go.NewLatLng(benchGoGeo.Lat, benchGoGeo.Lng) },
	)
}

func BenchmarkLatLngString(b *testing.B) {
	compare(b,
		func() { blackhole = benchGeo.String() },
		func() { blackhole = benchGoGeo.String() },
	)
}

func BenchmarkCellToLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToLatLng(benchCell); blackhole = out },
		func() { out, _ := h3go.CellToLatLng(benchGoCell); blackhole = out },
	)
}

func BenchmarkCellLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.LatLng(); blackhole = out },
		func() { out, _ := benchGoCell.LatLng(); blackhole = out },
	)
}

func BenchmarkCellToString(b *testing.B) {
	compare(b,
		func() { blackhole = h3.CellToString(benchCell) },
		func() { blackhole = h3go.CellToString(benchGoCell) },
	)
}

func BenchmarkCellString(b *testing.B) {
	compare(b,
		func() { blackhole = benchCell.String() },
		func() { blackhole = benchGoCell.String() },
	)
}

func BenchmarkCellMarshalText(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.MarshalText(); blackhole = out },
		func() { out, _ := benchGoCell.MarshalText(); blackhole = out },
	)
}

func BenchmarkCellFromString(b *testing.B) {
	compare(b,
		func() { blackhole = h3.CellFromString(benchCellStr) },
		func() { blackhole = h3go.CellFromString(benchCellStr) },
	)
}

func BenchmarkIndexFromString(b *testing.B) {
	compare(b,
		func() { blackhole = h3.IndexFromString(benchCellStr) },
		func() { blackhole = h3go.IndexFromString(benchCellStr) },
	)
}

func BenchmarkIndexToString(b *testing.B) {
	compare(b,
		func() { blackhole = h3.IndexToString(uint64(benchCell)) },
		func() { blackhole = h3go.IndexToString(uint64(benchGoCell)) },
	)
}

// ---------------------------------------------------------------------------
// Index introspection
// ---------------------------------------------------------------------------

func BenchmarkResolution(b *testing.B) {
	compare(b,
		func() { blackhole = benchCell.Resolution() },
		func() { blackhole = benchGoCell.Resolution() },
	)
}

func BenchmarkBaseCellNumberFunc(b *testing.B) {
	compare(b,
		func() { blackhole = h3.BaseCellNumber(benchCell) },
		func() { blackhole = h3go.BaseCellNumber(benchGoCell) },
	)
}

func BenchmarkBaseCellNumber(b *testing.B) {
	compare(b,
		func() { blackhole = benchCell.BaseCellNumber() },
		func() { blackhole = benchGoCell.BaseCellNumber() },
	)
}

func BenchmarkIsValid(b *testing.B) {
	compare(b,
		func() { blackhole = benchCell.IsValid() },
		func() { blackhole = benchGoCell.IsValid() },
	)
}

func BenchmarkIsPentagon(b *testing.B) {
	compare(b,
		func() { blackhole = benchCell.IsPentagon() },
		func() { blackhole = benchGoCell.IsPentagon() },
	)
}

func BenchmarkIsResClassIII(b *testing.B) {
	compare(b,
		func() { blackhole = benchCell.IsResClassIII() },
		func() { blackhole = benchGoCell.IsResClassIII() },
	)
}

func BenchmarkCellIndexDigit(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.IndexDigit(benchRes); blackhole = out },
		func() { out, _ := benchGoCell.IndexDigit(benchRes); blackhole = out },
	)
}

func BenchmarkNumCells(b *testing.B) {
	compare(b,
		func() { blackhole = h3.NumCells(benchRes) },
		func() { blackhole = h3go.NumCells(benchRes) },
	)
}

func BenchmarkRes0Cells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.Res0Cells(); blackhole = out },
		func() { out, _ := h3go.Res0Cells(); blackhole = out },
	)
}

func BenchmarkPentagons(b *testing.B) {
	compare(b,
		func() { out, _ := h3.Pentagons(benchRes); blackhole = out },
		func() { out, _ := h3go.Pentagons(benchRes); blackhole = out },
	)
}

// ---------------------------------------------------------------------------
// Hierarchy
// ---------------------------------------------------------------------------

func BenchmarkParent(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Parent(6); blackhole = out },
		func() { out, _ := benchGoCell.Parent(6); blackhole = out },
	)
}

func BenchmarkImmediateParent(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.ImmediateParent(); blackhole = out },
		func() { out, _ := benchGoCell.ImmediateParent(); blackhole = out },
	)
}

func BenchmarkChildren(b *testing.B) {
	compare(b,
		func() { out, _ := benchParent.Children(benchRes); blackhole = out },
		func() { out, _ := h3goCell(benchParent).Children(benchRes); blackhole = out },
	)
}

func BenchmarkImmediateChildren(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.ImmediateChildren(); blackhole = out },
		func() { out, _ := benchGoCell.ImmediateChildren(); blackhole = out },
	)
}

func BenchmarkCenterChild(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.CenterChild(benchRes + 2); blackhole = out },
		func() { out, _ := benchGoCell.CenterChild(benchRes + 2); blackhole = out },
	)
}

func BenchmarkCellToChildPos(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToChildPos(benchCell, 6); blackhole = out },
		func() { out, _ := h3go.CellToChildPos(benchGoCell, 6); blackhole = out },
	)
}

func BenchmarkChildPos(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.ChildPos(6); blackhole = out },
		func() { out, _ := benchGoCell.ChildPos(6); blackhole = out },
	)
}

func BenchmarkChildPosToCellFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.ChildPosToCell(0, benchParent, benchRes); blackhole = out },
		func() { out, _ := h3go.ChildPosToCell(0, h3goCell(benchParent), benchRes); blackhole = out },
	)
}

func BenchmarkChildPosToCell(b *testing.B) {
	compare(b,
		func() { out, _ := benchParent.ChildPosToCell(0, benchRes); blackhole = out },
		func() { out, _ := h3goCell(benchParent).ChildPosToCell(0, benchRes); blackhole = out },
	)
}

func BenchmarkCompactCells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CompactCells(benchChildren); blackhole = out },
		func() { out, _ := h3go.CompactCells(benchGoChildren); blackhole = out },
	)
}

func BenchmarkUncompactCells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.UncompactCells(benchUncompactIn, benchRes); blackhole = out },
		func() { out, _ := h3go.UncompactCells(benchGoUncompactIn, benchRes); blackhole = out },
	)
}

// ---------------------------------------------------------------------------
// Boundary
// ---------------------------------------------------------------------------

func BenchmarkCellToBoundary(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToBoundary(benchCell); blackhole = out },
		func() { out, _ := h3go.CellToBoundary(benchGoCell); blackhole = out },
	)
}

func BenchmarkCellBoundary(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Boundary(); blackhole = out },
		func() { out, _ := benchGoCell.Boundary(); blackhole = out },
	)
}

// ---------------------------------------------------------------------------
// Measures: area, edge length, great-circle distance
// ---------------------------------------------------------------------------

func BenchmarkCellAreaRads2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellAreaRads2(benchCell); blackhole = out },
		func() { out, _ := h3go.CellAreaRads2(benchGoCell); blackhole = out },
	)
}

func BenchmarkCellAreaKm2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellAreaKm2(benchCell); blackhole = out },
		func() { out, _ := h3go.CellAreaKm2(benchGoCell); blackhole = out },
	)
}

func BenchmarkCellAreaM2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellAreaM2(benchCell); blackhole = out },
		func() { out, _ := h3go.CellAreaM2(benchGoCell); blackhole = out },
	)
}

func BenchmarkHexagonAreaAvgKm2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonAreaAvgKm2(benchRes); blackhole = out },
		func() { out, _ := h3go.HexagonAreaAvgKm2(benchRes); blackhole = out },
	)
}

func BenchmarkHexagonAreaAvgM2(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonAreaAvgM2(benchRes); blackhole = out },
		func() { out, _ := h3go.HexagonAreaAvgM2(benchRes); blackhole = out },
	)
}

func BenchmarkEdgeLengthRads(b *testing.B) {
	compare(b,
		func() { out, _ := h3.EdgeLengthRads(benchEdge); blackhole = out },
		func() { out, _ := h3go.EdgeLengthRads(benchGoEdge); blackhole = out },
	)
}

func BenchmarkEdgeLengthKm(b *testing.B) {
	compare(b,
		func() { out, _ := h3.EdgeLengthKm(benchEdge); blackhole = out },
		func() { out, _ := h3go.EdgeLengthKm(benchGoEdge); blackhole = out },
	)
}

func BenchmarkEdgeLengthM(b *testing.B) {
	compare(b,
		func() { out, _ := h3.EdgeLengthM(benchEdge); blackhole = out },
		func() { out, _ := h3go.EdgeLengthM(benchGoEdge); blackhole = out },
	)
}

func BenchmarkHexagonEdgeLengthAvgKm(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonEdgeLengthAvgKm(benchRes); blackhole = out },
		func() { out, _ := h3go.HexagonEdgeLengthAvgKm(benchRes); blackhole = out },
	)
}

func BenchmarkHexagonEdgeLengthAvgM(b *testing.B) {
	compare(b,
		func() { out, _ := h3.HexagonEdgeLengthAvgM(benchRes); blackhole = out },
		func() { out, _ := h3go.HexagonEdgeLengthAvgM(benchRes); blackhole = out },
	)
}

func BenchmarkGreatCircleDistanceRads(b *testing.B) {
	compare(b,
		func() { blackhole = h3.GreatCircleDistanceRads(benchGeo, benchGeo2) },
		func() { blackhole = h3go.GreatCircleDistanceRads(benchGoGeo, benchGoGeo2) },
	)
}

func BenchmarkGreatCircleDistanceKm(b *testing.B) {
	compare(b,
		func() { blackhole = h3.GreatCircleDistanceKm(benchGeo, benchGeo2) },
		func() { blackhole = h3go.GreatCircleDistanceKm(benchGoGeo, benchGoGeo2) },
	)
}

func BenchmarkGreatCircleDistanceM(b *testing.B) {
	compare(b,
		func() { blackhole = h3.GreatCircleDistanceM(benchGeo, benchGeo2) },
		func() { blackhole = h3go.GreatCircleDistanceM(benchGoGeo, benchGoGeo2) },
	)
}

// ---------------------------------------------------------------------------
// Grid traversal
// ---------------------------------------------------------------------------

func BenchmarkGridDiskFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDisk(benchCell, 5); blackhole = out },
		func() { out, _ := h3go.GridDisk(benchGoCell, 5); blackhole = out },
	)
}

func BenchmarkGridDisk(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDisk(5); blackhole = out },
		func() { out, _ := benchGoCell.GridDisk(5); blackhole = out },
	)
}

func BenchmarkGridDiskDistancesFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDiskDistances(benchCell, 5); blackhole = out },
		func() { out, _ := h3go.GridDiskDistances(benchGoCell, 5); blackhole = out },
	)
}

func BenchmarkGridDiskDistances(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDiskDistances(5); blackhole = out },
		func() { out, _ := benchGoCell.GridDiskDistances(5); blackhole = out },
	)
}

func BenchmarkGridDiskDistancesSafeFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDiskDistancesSafe(benchCell, 5); blackhole = out },
		func() { out, _ := h3go.GridDiskDistancesSafe(benchGoCell, 5); blackhole = out },
	)
}

func BenchmarkGridDiskDistancesSafe(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDiskDistancesSafe(5); blackhole = out },
		func() { out, _ := benchGoCell.GridDiskDistancesSafe(5); blackhole = out },
	)
}

func BenchmarkGridDiskDistancesUnsafeFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDiskDistancesUnsafe(benchCell, 5); blackhole = out },
		func() { out, _ := h3go.GridDiskDistancesUnsafe(benchGoCell, 5); blackhole = out },
	)
}

func BenchmarkGridDiskDistancesUnsafe(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDiskDistancesUnsafe(5); blackhole = out },
		func() { out, _ := benchGoCell.GridDiskDistancesUnsafe(5); blackhole = out },
	)
}

func BenchmarkGridRingFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridRing(benchCell, 5); blackhole = out },
		func() { out, _ := h3go.GridRing(benchGoCell, 5); blackhole = out },
	)
}

func BenchmarkGridRing(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridRing(5); blackhole = out },
		func() { out, _ := benchGoCell.GridRing(5); blackhole = out },
	)
}

func BenchmarkGridRingUnsafeFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridRingUnsafe(benchCell, 5); blackhole = out },
		func() { out, _ := h3go.GridRingUnsafe(benchGoCell, 5); blackhole = out },
	)
}

func BenchmarkGridRingUnsafe(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridRingUnsafe(5); blackhole = out },
		func() { out, _ := benchGoCell.GridRingUnsafe(5); blackhole = out },
	)
}

func BenchmarkGridDisksUnsafe(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDisksUnsafe(benchDisk, 3); blackhole = out },
		func() { out, _ := h3go.GridDisksUnsafe(benchGoDisk, 3); blackhole = out },
	)
}

func BenchmarkGridDistanceFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridDistance(benchCell, benchTarget); blackhole = out },
		func() { out, _ := h3go.GridDistance(benchGoCell, benchGoTarget); blackhole = out },
	)
}

func BenchmarkGridDistance(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridDistance(benchTarget); blackhole = out },
		func() { out, _ := benchGoCell.GridDistance(benchGoTarget); blackhole = out },
	)
}

func BenchmarkGridPathFunc(b *testing.B) {
	compare(b,
		func() { out, _ := h3.GridPath(benchCell, benchTarget); blackhole = out },
		func() { out, _ := h3go.GridPath(benchGoCell, benchGoTarget); blackhole = out },
	)
}

func BenchmarkGridPath(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.GridPath(benchTarget); blackhole = out },
		func() { out, _ := benchGoCell.GridPath(benchGoTarget); blackhole = out },
	)
}

func BenchmarkIsNeighbor(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.IsNeighbor(benchNeighbor); blackhole = out },
		func() { out, _ := benchGoCell.IsNeighbor(benchGoNeighbor); blackhole = out },
	)
}

// ---------------------------------------------------------------------------
// Local IJ coordinates
// ---------------------------------------------------------------------------

func BenchmarkCellToLocalIJ(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToLocalIJ(benchCell, benchNeighbor); blackhole = out },
		func() { out, _ := h3go.CellToLocalIJ(benchGoCell, benchGoNeighbor); blackhole = out },
	)
}

func BenchmarkLocalIJToCell(b *testing.B) {
	compare(b,
		func() { out, _ := h3.LocalIJToCell(benchCell, benchLocalIJ); blackhole = out },
		func() { out, _ := h3go.LocalIJToCell(benchGoCell, benchGoLocalIJ); blackhole = out },
	)
}

// ---------------------------------------------------------------------------
// Icosahedron faces
// ---------------------------------------------------------------------------

func BenchmarkIcosahedronFaces(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.IcosahedronFaces(); blackhole = out },
		func() { out, _ := benchGoCell.IcosahedronFaces(); blackhole = out },
	)
}

// ---------------------------------------------------------------------------
// Vertexes
// ---------------------------------------------------------------------------

func BenchmarkCellToVertex(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToVertex(benchCell, 0); blackhole = out },
		func() { out, _ := h3go.CellToVertex(benchGoCell, 0); blackhole = out },
	)
}

func BenchmarkCellVertex(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Vertex(0); blackhole = out },
		func() { out, _ := benchGoCell.Vertex(0); blackhole = out },
	)
}

func BenchmarkCellToVertexes(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellToVertexes(benchCell); blackhole = out },
		func() { out, _ := h3go.CellToVertexes(benchGoCell); blackhole = out },
	)
}

func BenchmarkCellVertexes(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.Vertexes(); blackhole = out },
		func() { out, _ := benchGoCell.Vertexes(); blackhole = out },
	)
}

func BenchmarkVertexToLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := h3.VertexToLatLng(benchVertex); blackhole = out },
		func() { out, _ := h3go.VertexToLatLng(benchGoVertex); blackhole = out },
	)
}

func BenchmarkVertexLatLng(b *testing.B) {
	compare(b,
		func() { out, _ := benchVertex.LatLng(); blackhole = out },
		func() { out, _ := benchGoVertex.LatLng(); blackhole = out },
	)
}

func BenchmarkIsValidVertexFunc(b *testing.B) {
	compare(b,
		func() { blackhole = h3.IsValidVertex(benchVertex) },
		func() { blackhole = h3go.IsValidVertex(benchGoVertex) },
	)
}

func BenchmarkVertexIsValid(b *testing.B) {
	compare(b,
		func() { blackhole = benchVertex.IsValid() },
		func() { blackhole = benchGoVertex.IsValid() },
	)
}

func BenchmarkVertexResolution(b *testing.B) {
	compare(b,
		func() { blackhole = benchVertex.Resolution() },
		func() { blackhole = benchGoVertex.Resolution() },
	)
}

func BenchmarkVertexIndexDigit(b *testing.B) {
	compare(b,
		func() { out, _ := benchVertex.IndexDigit(benchRes); blackhole = out },
		func() { out, _ := benchGoVertex.IndexDigit(benchRes); blackhole = out },
	)
}

func BenchmarkVertexString(b *testing.B) {
	compare(b,
		func() { blackhole = benchVertex.String() },
		func() { blackhole = benchGoVertex.String() },
	)
}

func BenchmarkVertexMarshalText(b *testing.B) {
	compare(b,
		func() { out, _ := benchVertex.MarshalText(); blackhole = out },
		func() { out, _ := benchGoVertex.MarshalText(); blackhole = out },
	)
}

func BenchmarkVertexFromString(b *testing.B) {
	compare(b,
		func() { blackhole = h3.VertexFromString(benchVertexStr) },
		func() { blackhole = h3go.VertexFromString(benchVertexStr) },
	)
}

// ---------------------------------------------------------------------------
// Directed edges
// ---------------------------------------------------------------------------

func BenchmarkDirectedEdge(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.DirectedEdge(benchNeighbor); blackhole = out },
		func() { out, _ := benchGoCell.DirectedEdge(benchGoNeighbor); blackhole = out },
	)
}

func BenchmarkDirectedEdges(b *testing.B) {
	compare(b,
		func() { out, _ := benchCell.DirectedEdges(); blackhole = out },
		func() { out, _ := benchGoCell.DirectedEdges(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeCells(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Cells(); blackhole = out },
		func() { out, _ := benchGoEdge.Cells(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeOrigin(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Origin(); blackhole = out },
		func() { out, _ := benchGoEdge.Origin(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeDestination(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Destination(); blackhole = out },
		func() { out, _ := benchGoEdge.Destination(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeReverse(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Reverse(); blackhole = out },
		func() { out, _ := benchGoEdge.Reverse(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeIsValid(b *testing.B) {
	compare(b,
		func() { blackhole = benchEdge.IsValid() },
		func() { blackhole = benchGoEdge.IsValid() },
	)
}

func BenchmarkDirectedEdgeResolution(b *testing.B) {
	compare(b,
		func() { blackhole = benchEdge.Resolution() },
		func() { blackhole = benchGoEdge.Resolution() },
	)
}

func BenchmarkDirectedEdgeIndexDigit(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.IndexDigit(benchRes); blackhole = out },
		func() { out, _ := benchGoEdge.IndexDigit(benchRes); blackhole = out },
	)
}

func BenchmarkDirectedEdgeBoundary(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.Boundary(); blackhole = out },
		func() { out, _ := benchGoEdge.Boundary(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeString(b *testing.B) {
	compare(b,
		func() { blackhole = benchEdge.String() },
		func() { blackhole = benchGoEdge.String() },
	)
}

func BenchmarkDirectedEdgeMarshalText(b *testing.B) {
	compare(b,
		func() { out, _ := benchEdge.MarshalText(); blackhole = out },
		func() { out, _ := benchGoEdge.MarshalText(); blackhole = out },
	)
}

func BenchmarkDirectedEdgeFromString(b *testing.B) {
	compare(b,
		func() { blackhole = h3.DirectedEdgeFromString(benchEdgeStr) },
		func() { blackhole = h3go.DirectedEdgeFromString(benchEdgeStr) },
	)
}

// ---------------------------------------------------------------------------
// Regions: polygon <-> cells
// ---------------------------------------------------------------------------

func BenchmarkPolygonToCells(b *testing.B) {
	compare(b,
		func() { out, _ := h3.PolygonToCells(benchPolygon, benchPolyRes); blackhole = out },
		func() { out, _ := h3go.PolygonToCells(benchGoPolygon, benchPolyRes); blackhole = out },
	)
}

func BenchmarkGeoPolygonCells(b *testing.B) {
	compare(b,
		func() { out, _ := benchPolygon.Cells(benchPolyRes); blackhole = out },
		func() { out, _ := benchGoPolygon.Cells(benchPolyRes); blackhole = out },
	)
}

func BenchmarkPolygonToCellsExperimental(b *testing.B) {
	compare(b,
		func() {
			out, _ := h3.PolygonToCellsExperimental(benchPolygon, benchPolyRes, h3.ContainmentCenter)
			blackhole = out
		},
		func() {
			out, _ := h3go.PolygonToCellsExperimental(benchGoPolygon, benchPolyRes, h3go.ContainmentCenter)
			blackhole = out
		},
	)
}

func BenchmarkCellsToMultiPolygon(b *testing.B) {
	compare(b,
		func() { out, _ := h3.CellsToMultiPolygon(benchDisk); blackhole = out },
		func() { out, _ := h3go.CellsToMultiPolygon(benchGoDisk); blackhole = out },
	)
}
