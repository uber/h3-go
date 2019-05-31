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

// Package h3 is the go binding for Uber's H3 Geo Index system.
// It uses cgo to link with a statically compiled h3 library
package h3

/*
#cgo CFLAGS: -std=c99
#cgo CFLAGS: -DH3_HAVE_VLA=1
#cgo CFLAGS: -I ${SRCDIR}/include
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include <h3api.h>
#include <h3Index.h>
*/
import "C"
import (
	"errors"
	"math"
	"strconv"
)

const (
	// MaxCellBndryVerts is the maximum number of vertices that can be used
	// to represent the shape of a cell.
	MaxCellBndryVerts = C.MAX_CELL_BNDRY_VERTS

	// InvalidH3Index is a sentinel value for an invalid H3 index.
	InvalidH3Index = C.H3_INVALID_INDEX
)

var (
	// ErrPentagonEncountered is returned by functions that encounter a pentagon
	// and cannot handle it.
	ErrPentagonEncountered = errors.New("pentagon encountered")

	// ErrInvalidResolution is returned when the requested resolution is not valid
	ErrInvalidResolution = errors.New("resolution invalid")

	// conversion units for faster maths
	deg2rad = math.Pi / 180.0
	rad2deg = 180.0 / math.Pi
)

// H3Index is a type alias for the C type `H3Index`.  Effectively H3Index is a
// `uint64`.
type H3Index = C.H3Index

// GeoBoundary is a slice of `GeoCoord`.  Note, `len(GeoBoundary)` will never
// exceed `MaxCellBndryVerts`.
type GeoBoundary []GeoCoord

// GeoCoord is a struct for geographic coordinates.
type GeoCoord struct {
	Latitude, Longitude float64
}

func (g GeoCoord) toCPtr() *C.GeoCoord {
	return &C.GeoCoord{
		lat: C.double(deg2rad * g.Latitude),
		lon: C.double(deg2rad * g.Longitude),
	}
}

// --- INDEXING ---
//
// This section defines bindings for H3 indexing functions.
// Additional documentation available at
// https://uber.github.io/h3/#/documentation/api-reference/indexing

// FromGeo returns the H3Index at resolution `res` for a geographic coordinate.
func FromGeo(geoCoord GeoCoord, res int) H3Index {
	return H3Index(C.geoToH3(geoCoord.toCPtr(), C.int(res)))
}

// ToGeo returns the geographic centerpoint of an H3Index `h`.
func ToGeo(h H3Index) GeoCoord {
	g := C.GeoCoord{}
	C.h3ToGeo(h, &g)
	return geoCoordFromC(g)
}

// ToGeoBoundary returns a `GeoBoundary` of the H3Index `h`.
func ToGeoBoundary(h H3Index) GeoBoundary {
	gb := new(C.GeoBoundary)
	C.h3ToGeoBoundary(h, gb)
	return geoBndryFromC(gb)
}

// --- INSPECTION ---
// This section defines bindings for H3 inspection functions.
// Additional documentation available at
// https://uber.github.io/h3/#/documentation/api-reference/inspection

// Resolution returns the resolution of `h`.
func Resolution(h H3Index) int {
	return int(C.h3GetResolution(h))
}

// BaseCell returns the integer ID of the base cell the H3Index `h` belongs to.
func BaseCell(h H3Index) int {
	return int(C.h3GetBaseCell(h))
}

// FromString returns an H3Index parsed from a string.
func FromString(hStr string) H3Index {
	h, err := strconv.ParseUint(hStr, 16, 64)
	if err != nil {
		return 0
	}
	return H3Index(h)
}

// ToString returns a string representation of an H3Index.
func ToString(h H3Index) string {
	return strconv.FormatUint(uint64(h), 16)
}

// IsValid returns true if `h` is valid.
func IsValid(h H3Index) bool {
	return C.h3IsValid(h) == 1
}

// IsResClassIII returns true if `h` is a class III index. If false, `h` is a
// class II index.
func IsResClassIII(h H3Index) bool {
	return C.h3IsResClassIII(h) == 1
}

// IsPentagon returns true if `h` is a pentagon.
func IsPentagon(h H3Index) bool {
	return C.h3IsPentagon(h) == 1
}

// --- NEIGHBORS ---
// This section defines bindings for H3 neighbor traversal functions.
// Additional documentation available at
// https://uber.github.io/h3/#/documentation/api-reference/neighbors

// KRing implements the C function `kRing`.
func KRing(origin H3Index, k int) []H3Index {
	out := make([]C.H3Index, rangeSize(k))
	C.kRing(origin, C.int(k), &out[0])
	return h3SliceFromC(out)
}

// KRingDistances implements the C function `kRingDistances`.
func KRingDistances(origin H3Index, k int) [][]H3Index {
	rsz := rangeSize(k)
	outHexes := make([]C.H3Index, rsz)
	outDists := make([]C.int, rsz)
	C.kRingDistances(origin, C.int(k), &outHexes[0], &outDists[0])

	ret := make([][]H3Index, k+1)
	for i := 0; i <= k; i++ {
		ret[i] = make([]H3Index, 0, ringSize(i))
	}

	for i, d := range outDists {
		ret[d] = append(ret[d], H3Index(outHexes[i]))
	}
	return ret
}

// HexRange implements the C function `hexRange`.
func HexRange(origin H3Index, k int) ([]H3Index, error) {
	out := make([]C.H3Index, rangeSize(k))
	if rv := C.hexRange(origin, C.int(k), &out[0]); rv != 0 {
		return nil, ErrPentagonEncountered
	}
	return h3SliceFromC(out), nil
}

// HexRangeDistances implements the C function `hexRangeDistances`.
func HexRangeDistances(origin H3Index, k int) ([][]H3Index, error) {
	rsz := rangeSize(k)
	outHexes := make([]C.H3Index, rsz)
	outDists := make([]C.int, rsz)
	rv := C.hexRangeDistances(origin, C.int(k), &outHexes[0], &outDists[0])
	if rv != 0 {
		return nil, ErrPentagonEncountered
	}

	ret := make([][]H3Index, k+1)
	for i := 0; i <= k; i++ {
		ret[i] = make([]H3Index, 0, ringSize(i))
	}

	for i, d := range outDists {
		ret[d] = append(ret[d], H3Index(outHexes[i]))
	}
	return ret, nil
}

// HexRanges implements the C function `hexRanges`.
func HexRanges(origins []H3Index, k int) ([][]H3Index, error) {
	rsz := rangeSize(k)
	outHexes := make([]C.H3Index, rsz*len(origins))
	inHexes := h3SliceToC(origins)
	rv := C.hexRanges(&inHexes[0], C.int(len(origins)), C.int(k), &outHexes[0])
	if rv != 0 {
		return nil, ErrPentagonEncountered
	}

	ret := make([][]H3Index, len(origins))
	for i := 0; i < len(origins); i++ {
		ret[i] = make([]H3Index, rsz)
		for j := 0; j < rsz; j++ {
			ret[i][j] = H3Index(outHexes[i*rsz+j])
		}
	}
	return ret, nil
}

// HexRing implements the C function `hexRing`.
func HexRing(origin H3Index, k int) ([]H3Index, error) {
	out := make([]C.H3Index, ringSize(k))
	if rv := C.hexRing(origin, C.int(k), &out[0]); rv != 0 {
		return nil, ErrPentagonEncountered
	}
	return h3SliceFromC(out), nil
}

// AreNeighbors returns true if `h1` and `h2` are neighbors.  Two
// indexes are neighbors if they share an edge.
func AreNeighbors(h1, h2 H3Index) bool {
	return C.h3IndexesAreNeighbors(h1, h2) == 1
}

// --- HIERARCHY ---
// This section defines bindings for H3 hierarchical functions.
// Additional documentation available at
// https://uber.github.io/h3/#/documentation/api-reference/hierarchy

// ToParent returns the `H3Index` of the cell that contains `child` at
// resolution `parentRes`.  `parentRes` must be less than the resolution of
// `child`.
func ToParent(child H3Index, parentRes int) (parent H3Index) {
	return H3Index(C.h3ToParent(C.H3Index(child), C.int(parentRes)))
}

// ToChildren returns all the `H3Index`es of `parent` at resolution `childRes`.
// `childRes` must be larger than the resolution of `parent`.
func ToChildren(parent H3Index, childRes int) []H3Index {
	p := C.H3Index(parent)
	csz := C.int(childRes)
	out := make([]C.H3Index, int(C.maxH3ToChildrenSize(p, csz)))
	C.h3ToChildren(p, csz, &out[0])
	return h3SliceFromC(out)
}

// Compact merges full sets of children into their parent `H3Index`
// recursively, until no more merges are possible.
func Compact(in []H3Index) []H3Index {
	cin := h3SliceToC(in)
	csz := C.int(len(in))
	// worst case no compaction so we need a set **at least** as large as the
	// input
	cout := make([]C.H3Index, csz)
	C.compact(&cin[0], &cout[0], csz)
	return h3SliceFromC(cout)
}

// Uncompact splits every `H3Index` in `in` if its resolution is greater than
// `res` recursively. Returns all the `H3Index`es at resolution `res`.
func Uncompact(in []H3Index, res int) ([]H3Index, error) {
	cin := h3SliceToC(in)
	maxUncompactSz := C.maxUncompactSize(&cin[0], C.int(len(in)), C.int(res))
	if maxUncompactSz < 0 {
		// A size of less than zero indicates an error uncompacting such as the
		// requested resolution being less than the resolution of the hexagons.
		return nil, ErrInvalidResolution
	}
	cout := make([]C.H3Index, maxUncompactSz)
	C.uncompact(
		&cin[0], C.int(len(in)),
		&cout[0], maxUncompactSz,
		C.int(res))
	return h3SliceFromC(cout), nil
}

// --- REGIONS ---

// TODO(gilley) solve nested c struct problem for Polyfill funcs

// UnidirectionalEdge returns a unidirectional `H3Index` from `origin` to
// `destination`.
func UnidirectionalEdge(origin, destination H3Index) H3Index {
	return H3Index(C.getH3UnidirectionalEdge(origin, destination))
}

// UnidirectionalEdgeIsValid returns true if `edge` is a valid unidirectional
// edge index.
func UnidirectionalEdgeIsValid(edge H3Index) bool {
	return C.h3UnidirectionalEdgeIsValid(edge) == 1
}

// OriginFromUnidirectionalEdge returns the origin of a unidirectional
// edge.
func OriginFromUnidirectionalEdge(edge H3Index) H3Index {
	return H3Index(C.getOriginH3IndexFromUnidirectionalEdge(edge))
}

// DestinationFromUnidirectionalEdge returns the destination of a
// unidirectional edge.
func DestinationFromUnidirectionalEdge(edge H3Index) H3Index {
	return H3Index(C.getDestinationH3IndexFromUnidirectionalEdge(edge))
}

// FromUnidirectionalEdge returns the origin and destination from a
// unidirectional edge.
func FromUnidirectionalEdge(
	edge H3Index,
) (origin, destination H3Index) {
	cout := make([]C.H3Index, 2)
	C.getH3IndexesFromUnidirectionalEdge(edge, &cout[0])
	origin = H3Index(cout[0])
	destination = H3Index(cout[1])
	return
}

// ToUnidirectionalEdges returns the six (or five if pentagon) unidirectional
// edges from `h` to each of `h`'s neighbors.
func ToUnidirectionalEdges(h H3Index) []H3Index {
	// allocating max size, `h3SliceFromC` will adjust cap
	cout := make([]C.H3Index, 6)
	C.getH3UnidirectionalEdgesFromHexagon(h, &cout[0])
	return h3SliceFromC(cout)
}

// UnidirectionalEdgeBoundary returns the geocoordinates of a unidirectional
// edge boundary.
func UnidirectionalEdgeBoundary(edge H3Index) GeoBoundary {
	gb := new(C.GeoBoundary)
	C.getH3UnidirectionalEdgeBoundary(edge, gb)
	return geoBndryFromC(gb)
}

func geoCoordFromC(cg C.GeoCoord) GeoCoord {
	g := GeoCoord{}
	g.Latitude = rad2deg * float64(cg.lat)
	g.Longitude = rad2deg * float64(cg.lon)
	return g
}

func geoBndryFromC(cb *C.GeoBoundary) GeoBoundary {
	g := make(GeoBoundary, 0, MaxCellBndryVerts)
	for i := C.int(0); i < cb.numVerts; i++ {
		g = append(g, geoCoordFromC(cb.verts[i]))
	}
	return g
}

func h3SliceFromC(chs []C.H3Index) []H3Index {
	out := make([]H3Index, 0, len(chs))
	for _, ch := range chs {
		// C API returns a sparse array of indexes in the event pentagons and
		// deleted sequences are encountered.
		if ch == InvalidH3Index {
			continue
		}
		out = append(out, H3Index(ch))
	}
	return out
}

func h3SliceToC(hs []H3Index) []C.H3Index {
	out := make([]C.H3Index, len(hs))
	for i, h := range hs {
		out[i] = h
	}
	return out
}

func ringSize(k int) int {
	if k == 0 {
		return 1
	}
	return 6 * k
}

func rangeSize(k int) int {
	return int(C.maxKringSize(C.int(k)))
}
