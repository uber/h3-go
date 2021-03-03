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

package h3

/*
#cgo CFLAGS: -std=c99
#cgo CFLAGS: -DH3_HAVE_VLA=1
#cgo CFLAGS: -I ${SRCDIR}
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include <h3_h3api.h>
#include <h3_h3Index.h>
*/
import "C"

// --- MISCELLANEOUS H3 FUNCTIONS ---

// DegsToRads conversion from degree to radians
func DegsToRads(degrees float64) float64 {
	// 	return H3Index(C.geoToH3(geoCoord.toCPtr(), C.int(res)))
	return float64(C.degsToRads(C.double(degrees)))
}

// RadsToDegs conversion from radians to degrees
func RadsToDegs(radians float64) float64 {
	return float64(C.radsToDegs(C.double(radians)))
}

// PointDistRads "great circle distance" between pairs of GeoCoord points in radians
func PointDistRads(a GeoCoord, b GeoCoord) float64 {
	return float64(C.pointDistRads(a.toCPtr(), b.toCPtr()))
}

// PointDistKm "great circle distance" between pairs of GeoCoord points in kilometers
func PointDistKm(a GeoCoord, b GeoCoord) float64 {
	return float64(C.pointDistKm(a.toCPtr(), b.toCPtr()))
}

// PointDistM "great circle distance" between pairs of GeoCoord points in meters
func PointDistM(a GeoCoord, b GeoCoord) float64 {
	return float64(C.pointDistM(a.toCPtr(), b.toCPtr()))
}

// HexAreaKm2 average hexagon area in square kilometers (excludes pentagons)
func HexAreaKm2(res int) float64 {
	return float64(C.hexAreaKm2(C.int(res)))
}

// HexAreaM2 average hexagon area in square meters (excludes pentagons)
func HexAreaM2(res int) float64 {
	return float64(C.hexAreaM2(C.int(res)))
}

// CellAreaRads2 exact area for a specific cell (hexagon or pentagon) in radians^2
func CellAreaRads2(h H3Index) float64 {
	return float64(C.cellAreaRads2(h))
}

// CellAreaKm2 exact area for a specific cell (hexagon or pentagon) in kilometers^2
func CellAreaKm2(h H3Index) float64 {
	return float64(C.cellAreaKm2(h))
}

// CellAreaM2 exact area for a specific cell (hexagon or pentagon) in meters^2
func CellAreaM2(h H3Index) float64 {
	return float64(C.cellAreaM2(h))
}

// EdgeLengthKm average hexagon edge length in kilometers (excludes pentagons)
func EdgeLengthKm(res int) float64 {
	return float64(C.edgeLengthKm(C.int(res)))
}

// EdgeLengthM average hexagon edge length in meters (excludes pentagons)
func EdgeLengthM(res int) float64 {
	return float64(C.edgeLengthM(C.int(res)))
}

// ExactEdgeLengthRads exact length for a specific unidirectional edge in radians
func ExactEdgeLengthRads(edge H3Index) float64 {
	return float64(C.exactEdgeLengthRads(edge))
}

// ExactEdgeLengthKm exact length for a specific unidirectional edge in kilometers
func ExactEdgeLengthKm(edge H3Index) float64 {
	return float64(C.exactEdgeLengthKm(edge))
}

// ExactEdgeLengthM exact length for a specific unidirectional edge in meters
func ExactEdgeLengthM(edge H3Index) float64 {
	return float64(C.exactEdgeLengthM(edge))
}

// NumHexagons number of cells (hexagons and pentagons) for a given resolution
func NumHexagons(res int) int64 {
	return int64(C.numHexagons(C.int(res)))
}

// Res0IndexCount returns the number of resolution 0 cells (hexagons and pentagons)
func Res0IndexCount() int {
	return int(C.res0IndexCount())
}

// GetRes0Indexes provides all base cells in H3Index format
func GetRes0Indexes() []H3Index {
	out := make([]C.H3Index, Res0IndexCount())
	C.getRes0Indexes(&out[0])
	return h3SliceFromC(out)
}

// PentagonIndexCount returns the number of pentagons per resolution
func PentagonIndexCount() int {
	return int(C.pentagonIndexCount())
}

// GetPentagonIndexes generates all pentagons at the specified resolution
func GetPentagonIndexes(res int) []H3Index {
	out := make([]C.H3Index, PentagonIndexCount())
	C.getPentagonIndexes(C.int(res), &out[0])
	return h3SliceFromC(out)
}
