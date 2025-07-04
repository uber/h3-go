// Package h3 is the go binding for Uber's H3 Geo Index system.
// It uses cgo to link with a statically compiled h3 library
package h3

/*
 * Copyright 2018 Uber Technologies, Inc.
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

/*
#cgo CFLAGS: -std=c99
#cgo CFLAGS: -DH3_HAVE_VLA=1
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include <h3_h3api.h>
#include <h3_h3Index.h>
#include <h3_polygon.h>
#include <h3_polyfill.h>
*/
import "C"

import (
	"encoding"
	"errors"
	"math"
	"strconv"
	"strings"
	"unsafe"
)

const (
	// MaxCellBndryVerts is the maximum number of vertices that can be used
	// to represent the shape of a cell.
	MaxCellBndryVerts = C.MAX_CELL_BNDRY_VERTS

	// MaxResolution is the maximum H3 resolution a LatLng can be indexed to.
	MaxResolution = C.MAX_H3_RES

	// NumIcosaFaces is the number of faces on an icosahedron.
	NumIcosaFaces = C.NUM_ICOSA_FACES

	// NumBaseCells is the number of H3 base cells.
	NumBaseCells = C.NUM_BASE_CELLS

	// NumPentagons is the number of H3 pentagon cells (same at every resolution).
	NumPentagons = C.NUM_PENTAGONS

	// InvalidH3Index is a sentinel value for an invalid H3 index.
	InvalidH3Index = C.H3_NULL

	base16  = 16
	bitSize = 64

	numCellEdges    = 6
	numEdgeCells    = 2
	numCellVertexes = 6

	// DegsToRads converts degrees to radians by multiplying degrees by this constant.
	DegsToRads = math.Pi / 180.0
	// RadsToDegs converts radians to degrees by multiplying radians by this constant.
	RadsToDegs = 180.0 / math.Pi
)

const (
	latLngFloatPrecision = 5
	// latLngStringSize is the size to pre-allocate the buffer for.
	// Given latLngFloatPrecision, a typical string is "(DD.DDDDD, -DDD.DDDDD)"
	// which is ~25-30 bytes. 32 is a safe and efficient capacity to start with
	// to avoid re-allocation.
	latLngStringSize = 32
)

// PolygonToCells containment modes
const (
	ContainmentCenter          ContainmentMode = C.CONTAINMENT_CENTER           // Cell center is contained in the shape
	ContainmentFull            ContainmentMode = C.CONTAINMENT_FULL             // Cell is fully contained in the shape
	ContainmentOverlapping     ContainmentMode = C.CONTAINMENT_OVERLAPPING      // Cell overlaps the shape at any point
	ContainmentOverlappingBbox ContainmentMode = C.CONTAINMENT_OVERLAPPING_BBOX // Cell bounding box overlaps shape
	ContainmentInvalid         ContainmentMode = C.CONTAINMENT_INVALID          // This mode is invalid and should not be used
)

// Error codes.
var (
	ErrFailed                = errors.New("the operation failed")
	ErrDomain                = errors.New("argument was outside of acceptable range")
	ErrLatLngDomain          = errors.New("latitude or longitude arguments were outside of acceptable range")
	ErrResolutionDomain      = errors.New("resolution argument was outside of acceptable range")
	ErrCellInvalid           = errors.New("H3Index cell argument was not valid")
	ErrDirectedEdgeInvalid   = errors.New("H3Index directed edge argument was not valid")
	ErrUndirectedEdgeInvalid = errors.New("H3Index undirected edge argument was not valid")
	ErrVertexInvalid         = errors.New("H3Index vertex argument was not valid")
	ErrPentagon              = errors.New("pentagon distortion was encountered")
	ErrDuplicateInput        = errors.New("duplicate input was encountered in the arguments")
	ErrNotNeighbors          = errors.New("H3Index cell arguments were not neighbors")
	ErrRsolutionMismatch     = errors.New("H3Index cell arguments had incompatible resolutions")
	ErrMemoryAlloc           = errors.New("necessary memory allocation failed")
	ErrMemoryBounds          = errors.New("bounds of provided memory were not large enough")
	ErrOptionInvalid         = errors.New("mode or flags argument was not valid")

	ErrUnknown = errors.New("unknown error code returned by H3")

	errMap = map[C.uint32_t]error{
		0:  nil, // Success error code.
		1:  ErrFailed,
		2:  ErrDomain,
		3:  ErrLatLngDomain,
		4:  ErrResolutionDomain,
		5:  ErrCellInvalid,
		6:  ErrDirectedEdgeInvalid,
		7:  ErrUndirectedEdgeInvalid,
		8:  ErrVertexInvalid,
		9:  ErrPentagon,
		10: ErrDuplicateInput,
		11: ErrNotNeighbors,
		12: ErrRsolutionMismatch,
		13: ErrMemoryAlloc,
		14: ErrMemoryBounds,
		15: ErrOptionInvalid,
	}
)

type (
	// Cell is an Index that identifies a single hexagon cell at a resolution.
	Cell int64

	// DirectedEdge is an Index that identifies a directed edge between two cells.
	DirectedEdge int64

	// Vertex is an Index that identifies a single topological vertex, shared by three cells.
	// A vertex is arbitrarily assigned one of the three neighboring cells as its "owner", which is used to calculate
	// the canonical index and geographic coordinates for the vertex.
	Vertex int64

	// CoordIJ IJ hexagon coordinates
	//
	// Each axis is spaced 120 degrees apart.
	CoordIJ struct {
		I, J int
	}

	// CellBoundary is a slice of LatLng.  Note, len(CellBoundary) will never
	// exceed MaxCellBndryVerts.
	CellBoundary []LatLng

	// GeoLoop is a slice of LatLng points that make up a loop.
	GeoLoop []LatLng

	// LatLng is a struct for geographic coordinates in degrees.
	LatLng struct {
		Lat, Lng float64
	}

	// GeoPolygon is a GeoLoop with 0 or more GeoLoop holes.
	GeoPolygon struct {
		GeoLoop GeoLoop
		Holes   []GeoLoop
	}

	// ContainmentMode is an int for specifying PolygonToCell containment behavior.
	ContainmentMode C.uint32_t
)

// compile time checks that ensure interface implementation
var (
	_ encoding.TextMarshaler   = (*Cell)(nil)
	_ encoding.TextUnmarshaler = (*Cell)(nil)
	_ encoding.TextMarshaler   = (*Vertex)(nil)
	_ encoding.TextUnmarshaler = (*Vertex)(nil)
)

// NewLatLng is a helper function to create a LatLng.
func NewLatLng(lat, lng float64) LatLng {
	return LatLng{lat, lng}
}

// LatLngToCell returns the Cell at resolution for a geographic coordinate.
func LatLngToCell(latLng LatLng, resolution int) (Cell, error) {
	var i C.H3Index

	errC := C.latLngToCell(latLng.toCPtr(), C.int(resolution), &i)

	return Cell(i), toErr(errC)
}

// Cell returns the Cell at resolution for a geographic coordinate.
func (g LatLng) Cell(resolution int) (Cell, error) {
	return LatLngToCell(g, resolution)
}

// CellToLatLng returns the geographic centerpoint of a Cell.
func CellToLatLng(c Cell) (LatLng, error) {
	var g C.LatLng

	errC := C.cellToLatLng(C.H3Index(c), &g)

	return latLngFromC(g), toErr(errC)
}

// LatLng returns the Cell at resolution for a geographic coordinate.
func (c Cell) LatLng() (LatLng, error) {
	return CellToLatLng(c)
}

// CellToBoundary returns a CellBoundary of the Cell.
func CellToBoundary(c Cell) (CellBoundary, error) {
	var cb C.CellBoundary

	errC := C.cellToBoundary(C.H3Index(c), &cb)

	return cellBndryFromC(&cb), toErr(errC)
}

// Boundary returns a CellBoundary of the Cell.
func (c Cell) Boundary() (CellBoundary, error) {
	return CellToBoundary(c)
}

// GridDisk produces cells within grid distance k of the origin cell.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Output is placed in an array in no particular order. Elements of the output
// array may be left zero, as can happen when crossing a pentagon.
func GridDisk(origin Cell, k int) ([]Cell, error) {
	out := make([]C.H3Index, maxGridDiskSize(k))
	errC := C.gridDisk(C.H3Index(origin), C.int(k), &out[0])
	// QUESTION: should we prune zeroes from the output?
	return cellsFromC(out, true, false), toErr(errC)
}

// GridDisk produces cells within grid distance k of the origin cell.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Output is placed in an array in no particular order. Elements of the output
// array may be left zero, as can happen when crossing a pentagon.
func (c Cell) GridDisk(k int) ([]Cell, error) {
	return GridDisk(c, k)
}

// GridDisksUnsafe produces cells within grid distance k of all provided origin
// cells.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered in the same order origins were passed in. Inner slices
// are in no particular order.
//
// This does not call through to the underlying C.gridDisksUnsafe implementation
// as it is slightly easier to do so to avoid unnecessary type conversions.
func GridDisksUnsafe(origins []Cell, k int) ([][]Cell, error) {
	out := make([][]Cell, len(origins))
	gridDiskSize := maxGridDiskSize(k)
	for i := range origins {
		inner := make([]C.H3Index, gridDiskSize)
		errC := C.gridDiskUnsafe(C.H3Index(origins[i]), C.int(k), &inner[0])
		if err := toErr(errC); err != nil {
			return nil, err
		}
		out[i] = cellsFromC(inner, true, false)
	}
	return out, nil
}

// GridDiskDistances produces cells within grid distance k of the origin cell.
// This method optimistically tries the faster GridDiskDistancesUnsafe first.
// If a cell was a pentagon or was in the pentagon distortion area, it falls
// back to GridDiskDistancesSafe.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func GridDiskDistances(origin Cell, k int) ([][]Cell, error) {
	rsz := maxGridDiskSize(k)
	outHexes := make([]C.H3Index, rsz)
	outDists := make([]C.int, rsz)

	if err := toErr(C.gridDiskDistances(C.H3Index(origin), C.int(k), &outHexes[0], &outDists[0])); err != nil {
		return nil, err
	}

	ret := make([][]Cell, k+1)
	for i := 0; i <= k; i++ {
		ret[i] = make([]Cell, 0, ringSize(i))
	}

	for i, d := range outDists {
		ret[d] = append(ret[d], Cell(outHexes[i]))
	}

	return ret, nil
}

// GridDiskDistances produces cells within grid distance k of the origin cell.
// This method optimistically tries the faster GridDiskDistancesUnsafe first.
// If a cell was a pentagon or was in the pentagon distortion area, it falls
// back to GridDiskDistancesSafe.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func (c Cell) GridDiskDistances(k int) ([][]Cell, error) {
	return GridDiskDistances(c, k)
}

// GridDiskDistancesUnsafe produces cells within grid distance k of the origin cell.
// Output behavior is undefined when one of the cells returned by this
// function is a pentagon or is in the pentagon distortion area.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func GridDiskDistancesUnsafe(origin Cell, k int) ([][]Cell, error) {
	rsz := maxGridDiskSize(k)
	outHexes := make([]C.H3Index, rsz)
	outDists := make([]C.int, rsz)

	if err := toErr(C.gridDiskDistancesUnsafe(C.H3Index(origin), C.int(k), &outHexes[0], &outDists[0])); err != nil {
		return nil, err
	}

	ret := make([][]Cell, k+1)
	for i := 0; i <= k; i++ {
		ret[i] = make([]Cell, 0, ringSize(i))
	}

	for i, d := range outDists {
		ret[d] = append(ret[d], Cell(outHexes[i]))
	}

	return ret, nil
}

// GridDiskDistancesUnsafe produces cells within grid distance k of the origin cell.
// Output behavior is undefined when one of the cells returned by this
// function is a pentagon or is in the pentagon distortion area.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func (c Cell) GridDiskDistancesUnsafe(k int) ([][]Cell, error) {
	return GridDiskDistancesUnsafe(c, k)
}

// GridDiskDistancesSafe produces cells within grid distance k of the origin cell.
// This is the safe, but slow version of GridDiskDistances.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func GridDiskDistancesSafe(origin Cell, k int) ([][]Cell, error) {
	rsz := maxGridDiskSize(k)
	outHexes := make([]C.H3Index, rsz)
	outDists := make([]C.int, rsz)

	if err := toErr(C.gridDiskDistancesSafe(C.H3Index(origin), C.int(k), &outHexes[0], &outDists[0])); err != nil {
		return nil, err
	}

	ret := make([][]Cell, k+1)
	for i := 0; i <= k; i++ {
		ret[i] = make([]Cell, 0, ringSize(i))
	}

	for i, d := range outDists {
		ret[d] = append(ret[d], Cell(outHexes[i]))
	}

	return ret, nil
}

// GridDiskDistancesSafe produces cells within grid distance k of the origin cell.
// This is the safe, but slow version of GridDiskDistances.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func (c Cell) GridDiskDistancesSafe(k int) ([][]Cell, error) {
	return GridDiskDistancesSafe(c, k)
}

// GridRing produces the "hollow" ring of cells at exactly grid distance k from the origin cell.
//
// k-ring 0 returns just the origin hexagon.
//
// Elements of the output array may be left zero, as can happen when crossing a pentagon.
func GridRing(origin Cell, k int) ([]Cell, error) {
	if k < 0 {
		return nil, ErrDomain
	}
	out := make([]C.H3Index, ringSize(k))
	errC := C.gridRing(C.H3Index(origin), C.int(k), &out[0])
	return cellsFromC(out, true, false), toErr(errC)
}

// GridRing produces the "hollow" ring of cells at exactly grid distance k from the origin cell.
//
// k-ring 0 returns just the origin hexagon.
//
// Elements of the output array may be left zero, as can happen when crossing a pentagon.
func (c Cell) GridRing(k int) ([]Cell, error) {
	return GridRing(c, k)
}

// GridRingUnsafe produces the "hollow" ring of cells at exactly grid distance k from the origin cell.
//
// k-ring 0 returns just the origin hexagon.
func GridRingUnsafe(origin Cell, k int) ([]Cell, error) {
	if k < 0 {
		return nil, ErrDomain
	}
	out := make([]C.H3Index, ringSize(k))
	errC := C.gridRingUnsafe(C.H3Index(origin), C.int(k), &out[0])
	return cellsFromC(out, true, false), toErr(errC)
}

// GridRingUnsafe produces the "hollow" ring of cells at exactly grid distance k from the origin cell.
//
// k-ring 0 returns just the origin hexagon.
func (c Cell) GridRingUnsafe(k int) ([]Cell, error) {
	return GridRingUnsafe(c, k)
}

// PolygonToCells takes a given GeoJSON-like data structure fills it with the
// hexagon cells that are contained by the GeoJSON-like data structure.
//
// This implementation traces the GeoJSON geoloop(s) in cartesian space with
// hexagons, tests them and their neighbors to be contained by the geoloop(s),
// and then any newly found hexagons are used to test again until no new
// hexagons are found.
func PolygonToCells(polygon GeoPolygon, resolution int) ([]Cell, error) {
	if len(polygon.GeoLoop) == 0 {
		return nil, nil
	}
	cpoly := allocCGeoPolygon(polygon)

	defer freeCGeoPolygon(&cpoly)

	maxLen := new(C.int64_t)
	if err := toErr(C.maxPolygonToCellsSize(&cpoly, C.int(resolution), 0, maxLen)); err != nil {
		return nil, err
	}

	out := make([]C.H3Index, *maxLen)
	errC := C.polygonToCells(&cpoly, C.int(resolution), 0, &out[0])

	return cellsFromC(out, true, false), toErr(errC)
}

// PolygonToCellsExperimental takes a given GeoJSON-like data structure fills it with the
// hexagon cells that are contained by the GeoJSON-like data structure.
//
// This implementation traces the GeoJSON geoloop(s) in cartesian space with
// hexagons, tests them and their neighbors to be contained by the geoloop(s),
// and then any newly found hexagons are used to test again until no new
// hexagons are found.
func PolygonToCellsExperimental(polygon GeoPolygon, resolution int, mode ContainmentMode, maxNumCellsReturn ...int64) ([]Cell, error) {
	maxNumCells := int64(math.MaxInt64)
	if len(maxNumCellsReturn) > 0 {
		maxNumCells = maxNumCellsReturn[0]
	}
	if len(polygon.GeoLoop) == 0 {
		return nil, nil
	}
	cpoly := allocCGeoPolygon(polygon)

	defer freeCGeoPolygon(&cpoly)

	maxLen := new(C.int64_t)
	if err := toErr(C.maxPolygonToCellsSizeExperimental(&cpoly, C.int(resolution), C.uint32_t(mode), maxLen)); err != nil {
		return nil, err
	}

	out := make([]C.H3Index, *maxLen)
	errC := C.polygonToCellsExperimental(&cpoly, C.int(resolution), C.uint32_t(mode), C.int64_t(maxNumCells), &out[0])

	return cellsFromC(out, true, false), toErr(errC)
}

// Cells takes a given GeoJSON-like data structure fills it with the
// hexagon cells that are contained by the GeoJSON-like data structure.
//
// This implementation traces the GeoJSON geoloop(s) in cartesian space with
// hexagons, tests them and their neighbors to be contained by the geoloop(s),
// and then any newly found hexagons are used to test again until no new
// hexagons are found.
func (p GeoPolygon) Cells(resolution int) ([]Cell, error) {
	return PolygonToCells(p, resolution)
}

// CellsToMultiPolygon takes a set of cells and creates GeoPolygon(s)
// describing the outline(s) of a set of hexagons. Polygon outlines will follow
// GeoJSON MultiPolygon order: Each polygon will have one outer loop, which is first in
// the list, followed by any holes.
//
// It is expected that all hexagons in the set have the same resolution and that the set
// contains no duplicates. Behavior is undefined if duplicates or multiple resolutions are
// present, and the algorithm may produce unexpected or invalid output.
func CellsToMultiPolygon(cells []Cell) ([]GeoPolygon, error) {
	if len(cells) == 0 {
		return nil, nil
	}
	h3Indexes := cellsToC(cells)
	cLinkedGeoPolygon := new(C.LinkedGeoPolygon)
	if err := toErr(C.cellsToLinkedMultiPolygon(&h3Indexes[0], C.int(len(h3Indexes)), cLinkedGeoPolygon)); err != nil {
		return nil, err
	}

	currPoly := cLinkedGeoPolygon
	var countPoly int
	for currPoly != nil {
		countPoly++
		currPoly = currPoly.next
	}

	ret := make([]GeoPolygon, countPoly)

	// traverse polygons for linked list of polygons
	currPoly = cLinkedGeoPolygon
	countPoly = 0
	for currPoly != nil {
		currLoop := currPoly.first
		var countLoop int
		for currLoop != nil {
			countLoop++
			currLoop = currLoop.next
		}
		loops := make([]GeoLoop, countLoop)

		// traverse loops for a polygon
		currLoop = currPoly.first
		countLoop = 0
		for currLoop != nil {
			currPt := currLoop.first
			var countPt int
			for currPt != nil {
				countPt++
				currPt = currPt.next
			}
			loop := make([]LatLng, countPt)

			// traverse points for a loop
			currPt = currLoop.first
			countPt = 0
			for currPt != nil {
				loop[countPt] = latLngFromC(currPt.vertex)
				countPt++
				currPt = currPt.next
			}

			loops[countLoop] = loop
			countLoop++
			currLoop = currLoop.next
		}

		ret[countPoly] = GeoPolygon{GeoLoop: loops[0], Holes: loops[1:]}
		countPoly++
		currPoly = currPoly.next
	}

	C.destroyLinkedMultiPolygon(cLinkedGeoPolygon)
	return ret, nil
}

// GreatCircleDistanceRads returns the "great circle" or "haversine" distance between
// pairs of LatLng points (lat/lng pairs) in radians.
func GreatCircleDistanceRads(a, b LatLng) float64 {
	return float64(C.greatCircleDistanceRads(a.toCPtr(), b.toCPtr()))
}

// GreatCircleDistanceKm returns the "great circle" or "haversine" distance between pairs
// of LatLng points (lat/lng pairs) in kilometers.
func GreatCircleDistanceKm(a, b LatLng) float64 {
	return float64(C.greatCircleDistanceKm(a.toCPtr(), b.toCPtr()))
}

// GreatCircleDistanceM returns the "great circle" or "haversine" distance between pairs
// of LatLng points (lat/lng pairs) in meters.
func GreatCircleDistanceM(a, b LatLng) float64 {
	return float64(C.greatCircleDistanceM(a.toCPtr(), b.toCPtr()))
}

// HexagonAreaAvgKm2 returns the average hexagon area in square kilometers at the given
// resolution.
func HexagonAreaAvgKm2(resolution int) (float64, error) {
	var out C.double

	errC := C.getHexagonAreaAvgKm2(C.int(resolution), &out)

	return float64(out), toErr(errC)
}

// HexagonAreaAvgM2 returns the average hexagon area in square meters at the given
// resolution.
func HexagonAreaAvgM2(resolution int) (float64, error) {
	var out C.double

	errC := C.getHexagonAreaAvgM2(C.int(resolution), &out)

	return float64(out), toErr(errC)
}

// CellAreaRads2 returns the exact area of specific cell in square radians.
func CellAreaRads2(c Cell) (float64, error) {
	var out C.double

	errC := C.cellAreaRads2(C.H3Index(c), &out)

	return float64(out), toErr(errC)
}

// CellAreaKm2 returns the exact area of specific cell in square kilometers.
func CellAreaKm2(c Cell) (float64, error) {
	var out C.double

	errC := C.cellAreaKm2(C.H3Index(c), &out)

	return float64(out), toErr(errC)
}

// CellAreaM2 returns the exact area of specific cell in square meters.
func CellAreaM2(c Cell) (float64, error) {
	var out C.double

	errC := C.cellAreaM2(C.H3Index(c), &out)

	return float64(out), toErr(errC)
}

// HexagonEdgeLengthAvgKm returns the average hexagon edge length in kilometers
// at the given resolution.
func HexagonEdgeLengthAvgKm(resolution int) (float64, error) {
	var out C.double

	errC := C.getHexagonEdgeLengthAvgKm(C.int(resolution), &out)

	return float64(out), toErr(errC)
}

// HexagonEdgeLengthAvgM returns the average hexagon edge length in meters at
// the given resolution.
func HexagonEdgeLengthAvgM(resolution int) (float64, error) {
	var out C.double

	errC := C.getHexagonEdgeLengthAvgM(C.int(resolution), &out)

	return float64(out), toErr(errC)
}

// EdgeLengthRads returns the exact edge length of specific unidirectional edge
// in radians.
func EdgeLengthRads(e DirectedEdge) (float64, error) {
	var out C.double

	errC := C.edgeLengthRads(C.H3Index(e), &out)

	return float64(out), toErr(errC)
}

// EdgeLengthKm returns the exact edge length of specific unidirectional
// edge in kilometers.
func EdgeLengthKm(e DirectedEdge) (float64, error) {
	var out C.double

	errC := C.edgeLengthKm(C.H3Index(e), &out)

	return float64(out), toErr(errC)
}

// EdgeLengthM returns the exact edge length of specific unidirectional
// edge in meters.
func EdgeLengthM(e DirectedEdge) (float64, error) {
	var out C.double

	errC := C.edgeLengthM(C.H3Index(e), &out)

	return float64(out), toErr(errC)
}

// NumCells returns the number of cells at the given resolution.
func NumCells(resolution int) int {
	// NOTE: this is a mathematical operation, no need to call into H3 library.
	// See h3api.h for formula derivation.
	return 2 + 120*intPow(7, resolution) //nolint:mnd // math formula
}

// Res0Cells returns all the cells at resolution 0.
func Res0Cells() ([]Cell, error) {
	out := make([]C.H3Index, C.res0CellCount())
	errC := C.getRes0Cells(&out[0])

	return cellsFromC(out, false, false), toErr(errC)
}

// Pentagons returns all the pentagons at resolution.
func Pentagons(resolution int) ([]Cell, error) {
	out := make([]C.H3Index, NumPentagons)
	errC := C.getPentagons(C.int(resolution), &out[0])

	return cellsFromC(out, false, false), toErr(errC)
}

// Resolution returns the resolution of the cell.
func (c Cell) Resolution() int {
	return int(C.getResolution(C.H3Index(c)))
}

// Resolution returns the resolution of the edge.
func (e DirectedEdge) Resolution() int {
	return int(C.getResolution(C.H3Index(e)))
}

// BaseCellNumber returns the integer ID (0-121) of the base cell the H3Index h
// belongs to.
func BaseCellNumber(h Cell) int {
	return int(C.getBaseCellNumber(C.H3Index(h)))
}

// BaseCellNumber returns the integer ID (0-121) of the base cell the H3Index h
// belongs to.
func (c Cell) BaseCellNumber() int {
	return BaseCellNumber(c)
}

// IndexFromString returns an uint64 from a string. Should call c.IsValid() to check
// if the Cell is valid before using it.
func IndexFromString(s string) uint64 {
	if len(s) > 2 && strings.ToLower(s[:2]) == "0x" {
		s = s[2:]
	}
	c, _ := strconv.ParseUint(s, base16, bitSize)

	return c
}

// IndexToString returns a string from a Cell.
func IndexToString(i uint64) string {
	return strconv.FormatUint(i, base16)
}

// CellFromString returns a Cell from a string. Should call c.IsValid() to check
// if the Cell is valid before using it.
func CellFromString(s string) Cell {
	return Cell(IndexFromString(s))
}

// CellToString returns a string from a Cell.
func CellToString(c Cell) string {
	return IndexToString(uint64(c))
}

// VertexFromString returns a Vertex from a string. Should call v.IsValid() to check
// if the Vertex is valid before using it.
func VertexFromString(s string) Vertex {
	return Vertex(IndexFromString(s))
}

// String returns the string representation of the H3Index h.
func (c Cell) String() string {
	return CellToString(c)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (c Cell) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (c *Cell) UnmarshalText(text []byte) error {
	*c = Cell(IndexFromString(string(text)))
	if !c.IsValid() {
		return errors.New("invalid cell index")
	}

	return nil
}

// IsValid returns if a Cell is a valid cell (hexagon or pentagon).
func (c Cell) IsValid() bool {
	return c != 0 && C.isValidCell(C.H3Index(c)) == 1
}

// Parent returns the parent or grandparent Cell of this Cell.
func (c Cell) Parent(resolution int) (Cell, error) {
	var out C.H3Index

	errC := C.cellToParent(C.H3Index(c), C.int(resolution), &out)

	return Cell(out), toErr(errC)
}

// ImmediateParent returns the immediate parent of the cell.
func (c Cell) ImmediateParent() (Cell, error) {
	return c.Parent(c.Resolution() - 1)
}

// Children returns the children or grandchildren cells of this Cell.
func (c Cell) Children(resolution int) ([]Cell, error) {
	var outsz C.int64_t

	if err := toErr(C.cellToChildrenSize(C.H3Index(c), C.int(resolution), &outsz)); err != nil {
		return nil, err
	}
	out := make([]C.H3Index, outsz)

	// Seems like this function always returns E_SUCCESS.
	errC := C.cellToChildren(C.H3Index(c), C.int(resolution), &out[0])

	return cellsFromC(out, false, false), toErr(errC)
}

// ImmediateChildren returns the children or grandchildren cells of this Cell.
func (c Cell) ImmediateChildren() ([]Cell, error) {
	return c.Children(c.Resolution() + 1)
}

// CenterChild returns the center child Cell of this Cell.
func (c Cell) CenterChild(resolution int) (Cell, error) {
	var out C.H3Index

	errC := C.cellToCenterChild(C.H3Index(c), C.int(resolution), &out)

	return Cell(out), toErr(errC)
}

// IsResClassIII returns true if this is a class III index. If false, this is a
// class II index.
func (c Cell) IsResClassIII() bool {
	return C.isResClassIII(C.H3Index(c)) == 1
}

// IsPentagon returns true if this is a pentagon.
func (c Cell) IsPentagon() bool {
	return C.isPentagon(C.H3Index(c)) == 1
}

// IcosahedronFaces finds all icosahedron faces (0-19) intersected by this Cell.
func (c Cell) IcosahedronFaces() ([]int, error) {
	var outsz C.int

	// Seems like this function always returns E_SUCCESS.
	C.maxFaceCount(C.H3Index(c), &outsz)

	out := make([]C.int, outsz)
	errC := C.getIcosahedronFaces(C.H3Index(c), &out[0])

	return intsFromC(out), toErr(errC)
}

// IsNeighbor returns true if this Cell is a neighbor of the other Cell.
func (c Cell) IsNeighbor(other Cell) (bool, error) {
	var out C.int
	errC := C.areNeighborCells(C.H3Index(c), C.H3Index(other), &out)

	return out == 1, toErr(errC)
}

// DirectedEdge returns a DirectedEdge from this Cell to other.
func (c Cell) DirectedEdge(other Cell) (DirectedEdge, error) {
	var out C.H3Index
	errC := C.cellsToDirectedEdge(C.H3Index(c), C.H3Index(other), &out)

	return DirectedEdge(out), toErr(errC)
}

// DirectedEdges returns 6 directed edges with h as the origin.
func (c Cell) DirectedEdges() ([]DirectedEdge, error) {
	out := make([]C.H3Index, numCellEdges) // always 6 directed edges

	// Seems like this function always returns E_SUCCESS.
	errC := C.originToDirectedEdges(C.H3Index(c), &out[0])

	return edgesFromC(out), toErr(errC)
}

// IsValid determines if the directed edge is valid.
func (e DirectedEdge) IsValid() bool {
	return C.isValidDirectedEdge(C.H3Index(e)) == 1
}

// Origin returns the origin cell of this directed edge.
func (e DirectedEdge) Origin() (Cell, error) {
	var out C.H3Index
	errC := C.getDirectedEdgeOrigin(C.H3Index(e), &out)

	return Cell(out), toErr(errC)
}

// Destination returns the destination cell of this directed edge.
func (e DirectedEdge) Destination() (Cell, error) {
	var out C.H3Index
	errC := C.getDirectedEdgeDestination(C.H3Index(e), &out)

	return Cell(out), toErr(errC)
}

// Cells returns the origin and destination cells in that order.
func (e DirectedEdge) Cells() ([]Cell, error) {
	out := make([]C.H3Index, numEdgeCells)
	if err := toErr(C.directedEdgeToCells(C.H3Index(e), &out[0])); err != nil {
		return nil, err
	}

	return cellsFromC(out, false, false), nil
}

// Boundary provides the coordinates of the boundary of the directed edge. Note,
// the type returned is CellBoundary, but the coordinates will be from the
// center of the origin to the center of the destination. There may be more than
// 2 coordinates to account for crossing faces.
func (e DirectedEdge) Boundary() (CellBoundary, error) {
	var out C.CellBoundary
	if err := toErr(C.directedEdgeToBoundary(C.H3Index(e), &out)); err != nil {
		return nil, err
	}

	return cellBndryFromC(&out), nil
}

// CompactCells merges full sets of children into their parent H3Index
// recursively, until no more merges are possible.
func CompactCells(in []Cell) ([]Cell, error) {
	cin := cellsToC(in)
	csz := C.int64_t(len(in))
	// worst case no compaction so we need a set **at least** as large as the
	// input
	cout := make([]C.H3Index, csz)
	errC := C.compactCells(&cin[0], &cout[0], csz)

	return cellsFromC(cout, false, true), toErr(errC)
}

// UncompactCells splits every H3Index in in if its resolution is greater
// than resolution recursively. Returns all the H3Indexes at resolution resolution.
func UncompactCells(in []Cell, resolution int) ([]Cell, error) {
	cin := cellsToC(in)
	var csz C.int64_t
	if err := toErr(C.uncompactCellsSize(&cin[0], C.int64_t(len(cin)), C.int(resolution), &csz)); err != nil {
		return nil, err
	}

	cout := make([]C.H3Index, csz)
	errC := C.uncompactCells(
		&cin[0], C.int64_t(len(in)),
		&cout[0], csz,
		C.int(resolution))

	return cellsFromC(cout, false, true), toErr(errC)
}

// ChildPosToCell returns the child of cell a at a given position within an ordered list of all
// children at the specified resolution.
func ChildPosToCell(position int, a Cell, resolution int) (Cell, error) {
	var out C.H3Index

	errC := C.childPosToCell(C.int64_t(position), C.H3Index(a), C.int(resolution), &out)

	return Cell(out), toErr(errC)
}

// ChildPosToCell returns the child cell at a given position within an ordered list of all
// children at the specified resolution.
func (c Cell) ChildPosToCell(position int, resolution int) (Cell, error) {
	return ChildPosToCell(position, c, resolution)
}

// CellToChildPos returns the position of the cell a within an ordered list of all children of the cell's parent
// at the specified resolution.
func CellToChildPos(a Cell, resolution int) (int, error) {
	var out C.int64_t

	errC := C.cellToChildPos(C.H3Index(a), C.int(resolution), &out)

	return int(out), toErr(errC)
}

// ChildPos returns the position of the cell within an ordered list of all children of the cell's parent
// at the specified resolution.
func (c Cell) ChildPos(resolution int) (int, error) {
	return CellToChildPos(c, resolution)
}

// GridDistance returns grid distance between two cells.
//
// This function may fail to find the distance between two indexes, for example if they are very far apart. It may also
// fail when finding distances for indexes on opposite sides of a pentagon.
func GridDistance(a, b Cell) (int, error) {
	var out C.int64_t
	errC := C.gridDistance(C.H3Index(a), C.H3Index(b), &out)

	return int(out), toErr(errC)
}

// GridDistance returns grid distance between two cells.
//
// This function may fail to find the distance between two indexes, for example if they are very far apart. It may also
// fail when finding distances for indexes on opposite sides of a pentagon.
func (c Cell) GridDistance(other Cell) (int, error) {
	return GridDistance(c, other)
}

// GridPath returns the line of cells between the two cells (inclusive).
//
// This function may fail to find the line between two indexes, for example if they are very far apart. It may also fail
// when finding distances for indexes on opposite sides of a pentagon.
func GridPath(a, b Cell) ([]Cell, error) {
	var outsz C.int64_t
	if err := toErr(C.gridPathCellsSize(C.H3Index(a), C.H3Index(b), &outsz)); err != nil {
		return nil, err
	}

	out := make([]C.H3Index, outsz)
	if err := toErr(C.gridPathCells(C.H3Index(a), C.H3Index(b), &out[0])); err != nil {
		return nil, err
	}

	return cellsFromC(out, false, false), nil
}

// GridPath returns the line of cells between the two cells (inclusive).
//
// This function may fail to find the line between two indexes, for example if they are very far apart. It may also fail
// when finding distances for indexes on opposite sides of a pentagon.
func (c Cell) GridPath(other Cell) ([]Cell, error) {
	return GridPath(c, other)
}

// CellToLocalIJ produces ij coordinates for cell anchored by an origin.
//
// The coordinate space used by this function may have deleted regions or warping due to pentagonal distortion.
//
// Coordinates are only comparable if they come from the same origin index.
//
// Failure may occur if the index is too far away from the origin or if the index is on the other side of a pentagon.
func CellToLocalIJ(origin, cell Cell) (CoordIJ, error) {
	var out C.CoordIJ
	errC := C.cellToLocalIj(C.H3Index(origin), C.H3Index(cell), 0, &out)

	return CoordIJ{int(out.i), int(out.j)}, toErr(errC)
}

// LocalIJToCell produces a cell for ij coordinates anchored by an origin.
//
// The coordinate space used by this function may have deleted regions or warping due to pentagonal distortion.
//
// Failure may occur if the index is too far away from the origin or if the index is on the other side of a pentagon.
func LocalIJToCell(origin Cell, ij CoordIJ) (Cell, error) {
	var out C.H3Index
	errC := C.localIjToCell(C.H3Index(origin), ij.toCPtr(), 0, &out)

	return Cell(out), toErr(errC)
}

// Vertex returns a single vertex for a given cell, or InvalidH3Index if the vertex is invalid.
func (c Cell) Vertex(vertexNum int) (Vertex, error) {
	return CellToVertex(c, vertexNum)
}

// CellToVertex returns a single vertex for a given cell, or InvalidH3Index if the vertex is invalid.
func CellToVertex(c Cell, vertexNum int) (Vertex, error) {
	var out C.H3Index
	errC := C.cellToVertex(C.H3Index(c), C.int(vertexNum), &out)

	return Vertex(out), toErr(errC)
}

// Vertexes returns all vertexes for the given cell.
func (c Cell) Vertexes() ([]Vertex, error) {
	return CellToVertexes(c)
}

// CellToVertexes returns all vertexes for the given cell.
func CellToVertexes(c Cell) ([]Vertex, error) {
	out := make([]C.H3Index, numCellVertexes)
	if err := toErr(C.cellToVertexes(C.H3Index(c), &out[0])); err != nil {
		return nil, err
	}
	return vertexesFromC(out), nil
}

// LatLng returns the geographic coordinates of the vertex.
func (v Vertex) LatLng() (LatLng, error) {
	return VertexToLatLng(v)
}

// VertexToLatLng returns the geographic coordinates of the vertex.
func VertexToLatLng(vertex Vertex) (LatLng, error) {
	var out C.LatLng
	errC := C.vertexToLatLng(C.H3Index(vertex), &out)
	return latLngFromC(out), toErr(errC)
}

// IsValid returns whether the cell is a valid vertex.
func (v Vertex) IsValid() bool {
	return IsValidVertex(v)
}

// IsValidVertex returns whether the cell is a valid vertex.
func IsValidVertex(v Vertex) bool {
	return C.isValidVertex(C.H3Index(v)) == 1
}

// String returns a string from a Vertex.
func (v Vertex) String() string {
	return IndexToString(uint64(v))
}

// MarshalText implements the encoding.TextMarshaler interface.
func (v Vertex) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (v *Vertex) UnmarshalText(text []byte) error {
	*v = VertexFromString(string(text))
	if !v.IsValid() {
		return errors.New("invalid cell index")
	}

	return nil
}

func maxGridDiskSize(k int) int {
	return 3*k*(k+1) + 1
}

func latLngFromC(cg C.LatLng) LatLng {
	return LatLng{
		Lat: RadsToDegs * float64(cg.lat),
		Lng: RadsToDegs * float64(cg.lng),
	}
}

func cellBndryFromC(cb *C.CellBoundary) CellBoundary {
	g := make(CellBoundary, 0, MaxCellBndryVerts)
	for i := C.int(0); i < cb.numVerts; i++ {
		g = append(g, latLngFromC(cb.verts[i]))
	}

	return g
}

func ringSize(k int) int {
	if k == 0 {
		return 1
	}

	return 6 * k //nolint:mnd // math formula
}

// Convert slice of LatLngs to an array of C LatLngs (represented in C-style as
// a pointer to the first item in the array). The caller must free the returned
// pointer when finished with it.
func latLngsToC(coords []LatLng) *C.LatLng {
	if len(coords) == 0 {
		return nil
	}

	// Use malloc to construct a C-style struct array for the output
	cverts := C.malloc(C.size_t(C.sizeof_LatLng * len(coords)))
	pv := cverts

	for _, gc := range coords {
		*((*C.LatLng)(pv)) = *gc.toCPtr()
		pv = unsafe.Pointer(uintptr(pv) + C.sizeof_LatLng)
	}

	return (*C.LatLng)(cverts)
}

// Convert geofences (slices of slices of LatLnginates) to C geofences (represented in C-style as
// a pointer to the first item in the array). The caller must free the returned pointer and any
// pointer on the verts field when finished using it.
func geoLoopsToC(geofences []GeoLoop) *C.GeoLoop {
	if len(geofences) == 0 {
		return nil
	}

	// Use malloc to construct a C-style struct array for the output
	cgeofences := C.malloc(C.size_t(C.sizeof_GeoLoop * len(geofences)))

	pcgeofences := cgeofences

	for _, coords := range geofences {
		cverts := latLngsToC(coords)

		*((*C.GeoLoop)(pcgeofences)) = C.GeoLoop{
			verts:    cverts,
			numVerts: C.int(len(coords)),
		}
		pcgeofences = unsafe.Pointer(uintptr(pcgeofences) + C.sizeof_GeoLoop)
	}

	return (*C.GeoLoop)(cgeofences)
}

// Convert GeoPolygon struct to C equivalent struct.
func allocCGeoPolygon(gp GeoPolygon) C.GeoPolygon {
	cverts := latLngsToC(gp.GeoLoop)
	choles := geoLoopsToC(gp.Holes)

	return C.GeoPolygon{
		geoloop: C.GeoLoop{
			numVerts: C.int(len(gp.GeoLoop)),
			verts:    cverts,
		},
		numHoles: C.int(len(gp.Holes)),
		holes:    choles,
	}
}

// Free pointer values on a C GeoPolygon struct
func freeCGeoPolygon(cgp *C.GeoPolygon) {
	C.free(unsafe.Pointer(cgp.geoloop.verts))
	cgp.geoloop.verts = nil

	ph := unsafe.Pointer(cgp.holes)

	for i := C.int(0); i < cgp.numHoles; i++ {
		C.free(unsafe.Pointer((*C.GeoLoop)(ph).verts))
		(*C.GeoLoop)(ph).verts = nil
		ph = unsafe.Pointer(uintptr(ph) + uintptr(C.sizeof_GeoLoop))
	}

	C.free(unsafe.Pointer(cgp.holes))
	cgp.holes = nil
}

// https://stackoverflow.com/questions/64108933/how-to-use-math-pow-with-integers-in-golang
func intPow(n, m int) int {
	if m == 0 {
		return 1
	}
	result := n

	for i := 2; i <= m; i++ {
		result *= n
	}

	return result
}

func cellsFromC(chs []C.H3Index, prune, refit bool) []Cell {
	in := unsafe.Slice((*Cell)(unsafe.Pointer(&chs[0])), len(chs))
	out := in[:0]
	for i := range in {
		if prune && in[i] <= 0 {
			continue
		}
		out = append(out, in[i])
	}
	if refit {
		// Some algorithms require a maximum sized array, but only use a subset
		// of the memory.  refit sizes the slice to the last non-empty element.
		for i := len(out) - 1; i >= 0; i-- {
			if out[i] == 0 {
				out = out[:i]
			}
		}
	}
	return out
}

func vertexesFromC(chs []C.H3Index) []Vertex {
	in := unsafe.Slice((*Vertex)(unsafe.Pointer(&chs[0])), len(chs))
	out := in[:0]
	for i := range in {
		if in[i] <= 0 {
			continue
		}
		out = append(out, in[i])
	}
	return out
}

func edgesFromC(chs []C.H3Index) []DirectedEdge {
	out := make([]DirectedEdge, 0, len(chs))

	for i := range chs {
		if chs[i] <= 0 {
			continue
		}

		out = append(out, DirectedEdge(chs[i]))
	}

	return out
}

func cellsToC(chs []Cell) []C.H3Index {
	return unsafe.Slice((*C.H3Index)(unsafe.Pointer(&chs[0])), len(chs))
}

func intsFromC(chs []C.int) []int {
	out := make([]int, 0, len(chs))

	for i := range chs {
		// C API returns a sparse array of indexes in the event pentagons and
		// deleted sequences are encountered.
		if chs[i] != -1 {
			out = append(out, int(chs[i]))
		}
	}

	return out
}

func (g LatLng) String() string {
	buf := make([]byte, 0, latLngStringSize)
	buf = append(buf, '(')
	buf = strconv.AppendFloat(buf, g.Lat, 'f', latLngFloatPrecision, 64) //nolint:mnd // float bit size
	buf = append(buf, ',', ' ')
	buf = strconv.AppendFloat(buf, g.Lng, 'f', latLngFloatPrecision, 64) //nolint:mnd // float bit size
	buf = append(buf, ')')
	return string(buf)
}

func (g LatLng) toCPtr() *C.LatLng {
	return &C.LatLng{
		lat: C.double(DegsToRads * g.Lat),
		lng: C.double(DegsToRads * g.Lng),
	}
}

func (ij CoordIJ) toCPtr() *C.CoordIJ {
	return &C.CoordIJ{
		i: C.int(ij.I),
		j: C.int(ij.J),
	}
}

func toErr(errC C.uint32_t) error {
	err, ok := errMap[errC]
	if ok {
		return err
	}

	return ErrUnknown
}
