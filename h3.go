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
*/
import "C"

import (
	"errors"
	"fmt"
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

	// The number of faces on an icosahedron
	NumIcosaFaces = C.NUM_ICOSA_FACES

	// The number of H3 base cells
	NumBaseCells = C.NUM_BASE_CELLS

	// The number of H3 pentagon cells (same at every resolution)
	NumPentagons = C.NUM_PENTAGONS

	// InvalidH3Index is a sentinel value for an invalid H3 index.
	InvalidH3Index = C.H3_NULL

	base16  = 16
	bitSize = 64

	numCellEdges = 6
	numEdgeCells = 2

	DegsToRads = math.Pi / 180.0
	RadsToDegs = 180.0 / math.Pi
)

type (

	// Cell is an Index that identifies a single hexagon cell at a resolution.
	Cell int64

	// DirectedEdge is an Index that identifies a directed edge between two cells.
	DirectedEdge int64

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

	// LinkedGeoPolygon is a linked-list of GeoPolygons.
	// TODO: not implemented.
	LinkedGeoPolygon struct {
		Data GeoPolygon
		Next *LinkedGeoPolygon
	}
)

func NewLatLng(lat, lng float64) LatLng {
	return LatLng{lat, lng}
}

// LatLngToCell returns the Cell at resolution for a geographic coordinate.
func LatLngToCell(latLng LatLng, resolution int) Cell {
	var i C.H3Index

	C.latLngToCell(latLng.toCPtr(), C.int(resolution), &i)

	return Cell(i)
}

// Cell returns the Cell at resolution for a geographic coordinate.
func (g LatLng) Cell(resolution int) Cell {
	return LatLngToCell(g, resolution)
}

// CellToLatLng returns the geographic centerpoint of a Cell.
func CellToLatLng(c Cell) LatLng {
	var g C.LatLng

	C.cellToLatLng(C.H3Index(c), &g)

	return latLngFromC(g)
}

// LatLng returns the Cell at resolution for a geographic coordinate.
func (c Cell) LatLng() LatLng {
	return CellToLatLng(c)
}

// CellToBoundary returns a CellBoundary of the Cell.
func CellToBoundary(c Cell) CellBoundary {
	var cb C.CellBoundary

	C.cellToBoundary(C.H3Index(c), &cb)

	return cellBndryFromC(&cb)
}

// Boundary returns a CellBoundary of the Cell.
func (c Cell) Boundary() CellBoundary {
	return CellToBoundary(c)
}

// GridDisk produces cells within grid distance k of the origin cell.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Output is placed in an array in no particular order. Elements of the output
// array may be left zero, as can happen when crossing a pentagon.
func GridDisk(origin Cell, k int) []Cell {
	out := make([]C.H3Index, maxGridDiskSize(k))
	C.gridDisk(C.H3Index(origin), C.int(k), &out[0])
	// QUESTION: should we prune zeroes from the output?
	return cellsFromC(out, true, false)
}

// GridDisk produces cells within grid distance k of the origin cell.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Output is placed in an array in no particular order. Elements of the output
// array may be left zero, as can happen when crossing a pentagon.
func (c Cell) GridDisk(k int) []Cell {
	return GridDisk(c, k)
}

// GridDiskDistances produces cells within grid distance k of the origin cell.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func GridDiskDistances(origin Cell, k int) [][]Cell {
	rsz := maxGridDiskSize(k)
	outHexes := make([]C.H3Index, rsz)
	outDists := make([]C.int, rsz)
	C.gridDiskDistances(C.H3Index(origin), C.int(k), &outHexes[0], &outDists[0])

	ret := make([][]Cell, k+1)
	for i := 0; i <= k; i++ {
		ret[i] = make([]Cell, 0, ringSize(i))
	}

	for i, d := range outDists {
		ret[d] = append(ret[d], Cell(outHexes[i]))
	}

	return ret
}

// GridDiskDistances produces cells within grid distance k of the origin cell.
//
// k-ring 0 is defined as the origin cell, k-ring 1 is defined as k-ring 0 and
// all neighboring cells, and so on.
//
// Outer slice is ordered from origin outwards. Inner slices are in no
// particular order. Elements of the output array may be left zero, as can
// happen when crossing a pentagon.
func (c Cell) GridDiskDistances(k int) [][]Cell {
	return GridDiskDistances(c, k)
}

// PolygonToCells takes a given GeoJSON-like data structure fills it with the
// hexagon cells that are contained by the GeoJSON-like data structure.
//
// This implementation traces the GeoJSON geoloop(s) in cartesian space with
// hexagons, tests them and their neighbors to be contained by the geoloop(s),
// and then any newly found hexagons are used to test again until no new
// hexagons are found.
func PolygonToCells(polygon GeoPolygon, resolution int) []Cell {
	if len(polygon.GeoLoop) == 0 {
		return nil
	}
	cpoly := allocCGeoPolygon(polygon)

	defer freeCGeoPolygon(&cpoly)

	maxLen := new(C.int64_t)
	C.maxPolygonToCellsSize(&cpoly, C.int(resolution), 0, maxLen)

	out := make([]C.H3Index, *maxLen)
	C.polygonToCells(&cpoly, C.int(resolution), 0, &out[0])

	return cellsFromC(out, true, false)
}

// PolygonToCells takes a given GeoJSON-like data structure fills it with the
// hexagon cells that are contained by the GeoJSON-like data structure.
//
// This implementation traces the GeoJSON geoloop(s) in cartesian space with
// hexagons, tests them and their neighbors to be contained by the geoloop(s),
// and then any newly found hexagons are used to test again until no new
// hexagons are found.
func (p GeoPolygon) Cells(resolution int) []Cell {
	return PolygonToCells(p, resolution)
}

func CellToGeoPolygon(cell Cell) GeoPolygon {
	boundary := CellToBoundary(cell)
	loop := make(GeoLoop, len(boundary))

	for i, coord := range boundary {
		loop[i] = LatLng{Lat: coord.Lat, Lng: coord.Lng}
	}

	return GeoPolygon{GeoLoop: loop}
}

func CellsToMultiPolygon(cells []Cell) *LinkedGeoPolygon {
	var head, current *LinkedGeoPolygon

	for _, cell := range cells {
		geoPoly := CellToGeoPolygon(cell)

		if head == nil {
			head = &LinkedGeoPolygon{Data: geoPoly}
			current = head
		} else {
			current.Next = &LinkedGeoPolygon{Data: geoPoly}
			current = current.Next
		}
	}

	return head
}

// PointDistRads returns the "great circle" or "haversine" distance between
// pairs of LatLng points (lat/lng pairs) in radians.
func GreatCircleDistanceRads(a, b LatLng) float64 {
	return float64(C.greatCircleDistanceRads(a.toCPtr(), b.toCPtr()))
}

// PointDistKm returns the "great circle" or "haversine" distance between pairs
// of LatLng points (lat/lng pairs) in kilometers.
func GreatCircleDistanceKm(a, b LatLng) float64 {
	return float64(C.greatCircleDistanceKm(a.toCPtr(), b.toCPtr()))
}

// PointDistM returns the "great circle" or "haversine" distance between pairs
// of LatLng points (lat/lng pairs) in meters.
func GreatCircleDistanceM(a, b LatLng) float64 {
	return float64(C.greatCircleDistanceM(a.toCPtr(), b.toCPtr()))
}

// HexAreaKm2 returns the average hexagon area in square kilometers at the given
// resolution.
func HexagonAreaAvgKm2(resolution int) float64 {
	var out C.double

	C.getHexagonAreaAvgKm2(C.int(resolution), &out)

	return float64(out)
}

// HexAreaM2 returns the average hexagon area in square meters at the given
// resolution.
func HexagonAreaAvgM2(resolution int) float64 {
	var out C.double

	C.getHexagonAreaAvgM2(C.int(resolution), &out)

	return float64(out)
}

// CellAreaRads2 returns the exact area of specific cell in square radians.
func CellAreaRads2(c Cell) float64 {
	var out C.double

	C.cellAreaRads2(C.H3Index(c), &out)

	return float64(out)
}

// CellAreaKm2 returns the exact area of specific cell in square kilometers.
func CellAreaKm2(c Cell) float64 {
	var out C.double

	C.cellAreaKm2(C.H3Index(c), &out)

	return float64(out)
}

// CellAreaM2 returns the exact area of specific cell in square meters.
func CellAreaM2(c Cell) float64 {
	var out C.double

	C.cellAreaM2(C.H3Index(c), &out)

	return float64(out)
}

// HexagonEdgeLengthAvgKm returns the average hexagon edge length in kilometers
// at the given resolution.
func HexagonEdgeLengthAvgKm(resolution int) float64 {
	var out C.double

	C.getHexagonEdgeLengthAvgKm(C.int(resolution), &out)

	return float64(out)
}

// HexagonEdgeLengthAvgM returns the average hexagon edge length in meters at
// the given resolution.
func HexagonEdgeLengthAvgM(resolution int) float64 {
	var out C.double

	C.getHexagonEdgeLengthAvgM(C.int(resolution), &out)

	return float64(out)
}

// EdgeLengthRads returns the exact edge length of specific unidirectional edge
// in radians.
func EdgeLengthRads(e DirectedEdge) float64 {
	var out C.double

	C.edgeLengthRads(C.H3Index(e), &out)

	return float64(out)
}

// EdgeLengthKm returns the exact edge length of specific unidirectional
// edge in kilometers.
func EdgeLengthKm(e DirectedEdge) float64 {
	var out C.double

	C.edgeLengthKm(C.H3Index(e), &out)

	return float64(out)
}

// EdgeLengthM returns the exact edge length of specific unidirectional
// edge in meters.
func EdgeLengthM(e DirectedEdge) float64 {
	var out C.double

	C.edgeLengthM(C.H3Index(e), &out)

	return float64(out)
}

// NumCells returns the number of cells at the given resolution.
func NumCells(resolution int) int {
	// NOTE: this is a mathematical operation, no need to call into H3 library.
	// See h3api.h for formula derivation.
	return 2 + 120*intPow(7, (resolution)) //nolint:gomnd // math formula
}

// Res0Cells returns all the cells at resolution 0.
func Res0Cells() []Cell {
	out := make([]C.H3Index, C.res0CellCount())
	C.getRes0Cells(&out[0])

	return cellsFromC(out, false, false)
}

// Pentagons returns all the pentagons at resolution.
func Pentagons(resolution int) []Cell {
	out := make([]C.H3Index, NumPentagons)
	C.getPentagons(C.int(resolution), &out[0])

	return cellsFromC(out, false, false)
}

func (c Cell) Resolution() int {
	return int(C.getResolution(C.H3Index(c)))
}

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

// IndexFromString returns a Cell from a string. Should call c.IsValid() to check
// if the Cell is valid before using it.
func IndexFromString(s string) uint64 {
	if len(s) > 2 && strings.ToLower(s[:2]) == "0x" {
		s = s[2:]
	}
	c, _ := strconv.ParseUint(s, base16, bitSize)

	return c
}

// IndexToString returns a Cell from a string. Should call c.IsValid() to check
// if the Cell is valid before using it.
func IndexToString(i uint64) string {
	return strconv.FormatUint(i, base16)
}

// String returns the string representation of the H3Index h.
func (c Cell) String() string {
	return IndexToString(uint64(c))
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
func (c Cell) Parent(resolution int) Cell {
	var out C.H3Index

	C.cellToParent(C.H3Index(c), C.int(resolution), &out)

	return Cell(out)
}

// Parent returns the parent or grandparent Cell of this Cell.
func (c Cell) ImmediateParent() Cell {
	return c.Parent(c.Resolution() - 1)
}

// Children returns the children or grandchildren cells of this Cell.
func (c Cell) Children(resolution int) []Cell {
	var outsz C.int64_t

	C.cellToChildrenSize(C.H3Index(c), C.int(resolution), &outsz)
	out := make([]C.H3Index, outsz)

	C.cellToChildren(C.H3Index(c), C.int(resolution), &out[0])

	return cellsFromC(out, false, false)
}

// ImmediateChildren returns the children or grandchildren cells of this Cell.
func (c Cell) ImmediateChildren() []Cell {
	return c.Children(c.Resolution() + 1)
}

// CenterChild returns the center child Cell of this Cell.
func (c Cell) CenterChild(resolution int) Cell {
	var out C.H3Index

	C.cellToCenterChild(C.H3Index(c), C.int(resolution), &out)

	return Cell(out)
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
func (c Cell) IcosahedronFaces() []int {
	var outsz C.int

	C.maxFaceCount(C.H3Index(c), &outsz)
	out := make([]C.int, outsz)

	C.getIcosahedronFaces(C.H3Index(c), &out[0])

	return intsFromC(out)
}

// IsNeighbor returns true if this Cell is a neighbor of the other Cell.
func (c Cell) IsNeighbor(other Cell) bool {
	var out C.int
	C.areNeighborCells(C.H3Index(c), C.H3Index(other), &out)

	return out == 1
}

// DirectedEdge returns a DirectedEdge from this Cell to other.
func (c Cell) DirectedEdge(other Cell) DirectedEdge {
	var out C.H3Index
	C.cellsToDirectedEdge(C.H3Index(c), C.H3Index(other), &out)

	return DirectedEdge(out)
}

// DirectedEdges returns 6 directed edges with h as the origin.
func (c Cell) DirectedEdges() []DirectedEdge {
	out := make([]C.H3Index, numCellEdges) // always 6 directed edges
	C.originToDirectedEdges(C.H3Index(c), &out[0])

	return edgesFromC(out)
}

func (e DirectedEdge) IsValid() bool {
	return C.isValidDirectedEdge(C.H3Index(e)) == 1
}

// Origin returns the origin cell of this directed edge.
func (e DirectedEdge) Origin() Cell {
	var out C.H3Index
	C.getDirectedEdgeOrigin(C.H3Index(e), &out)

	return Cell(out)
}

// Destination returns the destination cell of this directed edge.
func (e DirectedEdge) Destination() Cell {
	var out C.H3Index
	C.getDirectedEdgeDestination(C.H3Index(e), &out)

	return Cell(out)
}

// Cells returns the origin and destination cells in that order.
func (e DirectedEdge) Cells() []Cell {
	out := make([]C.H3Index, numEdgeCells)
	C.directedEdgeToCells(C.H3Index(e), &out[0])

	return cellsFromC(out, false, false)
}

// Boundary provides the coordinates of the boundary of the directed edge. Note,
// the type returned is CellBoundary, but the coordinates will be from the
// center of the origin to the center of the destination. There may be more than
// 2 coordinates to account for crossing faces.
func (e DirectedEdge) Boundary() CellBoundary {
	var out C.CellBoundary
	C.directedEdgeToBoundary(C.H3Index(e), &out)

	return cellBndryFromC(&out)
}

// CompactCells merges full sets of children into their parent H3Index
// recursively, until no more merges are possible.
func CompactCells(in []Cell) []Cell {
	cin := cellsToC(in)
	csz := C.int64_t(len(in))
	// worst case no compaction so we need a set **at least** as large as the
	// input
	cout := make([]C.H3Index, csz)
	C.compactCells(&cin[0], &cout[0], csz)

	return cellsFromC(cout, false, true)
}

// UncompactCells splits every H3Index in in if its resolution is greater
// than resolution recursively. Returns all the H3Indexes at resolution resolution.
func UncompactCells(in []Cell, resolution int) []Cell {
	cin := cellsToC(in)
	var csz C.int64_t
	C.uncompactCellsSize(&cin[0], C.int64_t(len(cin)), C.int(resolution), &csz)

	cout := make([]C.H3Index, csz)
	C.uncompactCells(
		&cin[0], C.int64_t(len(in)),
		&cout[0], csz,
		C.int(resolution))

	return cellsFromC(cout, false, true)
}

// ChildPosToCell returns the child of cell a at a given position within an ordered list of all
// children at the specified resolution.
func ChildPosToCell(position int, a Cell, resolution int) Cell {
	var out C.H3Index

	C.childPosToCell(C.int64_t(position), C.H3Index(a), C.int(resolution), &out)

	return Cell(out)
}

// ChildPosToCell returns the child cell at a given position within an ordered list of all
// children at the specified resolution.
func (c Cell) ChildPosToCell(position int, resolution int) Cell {
	return ChildPosToCell(position, c, resolution)
}

// CellToChildPos returns the position of the cell a within an ordered list of all children of the cell's parent
// at the specified resolution.
func CellToChildPos(a Cell, resolution int) int {
	var out C.int64_t

	C.cellToChildPos(C.H3Index(a), C.int(resolution), &out)

	return int(out)
}

// ChildPos returns the position of the cell within an ordered list of all children of the cell's parent
// at the specified resolution.
func (c Cell) ChildPos(resolution int) int {
	return CellToChildPos(c, resolution)
}

func GridDistance(a, b Cell) int {
	var out C.int64_t
	C.gridDistance(C.H3Index(a), C.H3Index(b), &out)

	return int(out)
}

func (c Cell) GridDistance(other Cell) int {
	return GridDistance(c, other)
}

func GridPath(a, b Cell) []Cell {
	var outsz C.int64_t
	C.gridPathCellsSize(C.H3Index(a), C.H3Index(b), &outsz)

	out := make([]C.H3Index, outsz)
	C.gridPathCells(C.H3Index(a), C.H3Index(b), &out[0])

	return cellsFromC(out, false, false)
}

func (c Cell) GridPath(other Cell) []Cell {
	return GridPath(c, other)
}

func CellToLocalIJ(origin, cell Cell) CoordIJ {
	var out C.CoordIJ
	C.cellToLocalIj(C.H3Index(origin), C.H3Index(cell), 0, &out)

	return CoordIJ{int(out.i), int(out.j)}
}

func LocalIJToCell(origin Cell, ij CoordIJ) Cell {
	var out C.H3Index
	C.localIjToCell(C.H3Index(origin), ij.toCPtr(), 0, &out)

	return Cell(out)
}

func maxGridDiskSize(k int) int {
	return 3*k*(k+1) + 1
}

func latLngFromC(cg C.LatLng) LatLng {
	g := LatLng{}
	g.Lat = RadsToDegs * float64(cg.lat)
	g.Lng = RadsToDegs * float64(cg.lng)

	return g
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

	return 6 * k //nolint:gomnd // math formula
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
	// OPT: This could be more efficient if we unsafely cast the C array to a
	// []H3Index.
	out := make([]Cell, 0, len(chs))

	for i := range chs {
		if prune && chs[i] <= 0 {
			continue
		}

		out = append(out, Cell(chs[i]))
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
	// OPT: This could be more efficient if we unsafely cast the array to a
	// []C.H3Index.
	out := make([]C.H3Index, len(chs))
	for i := range chs {
		out[i] = C.H3Index(chs[i])
	}

	return out
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
	return fmt.Sprintf("(%.5f, %.5f)", g.Lat, g.Lng)
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
