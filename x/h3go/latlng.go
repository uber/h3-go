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
	"math"
	"strconv"
)

type (
	// LatLng is a struct for geographic coordinates in degrees.
	LatLng struct {
		Lat, Lng float64
	}
)

// LatLng string formatting parameters, matching the H3 C library.
const (
	latLngFloatPrecision = 5
	latLngStringSize     = 32
)

// --- Average cell sizes ---

var (
	// hexAreaAvgKm2 holds the average hexagon area at each resolution in square
	// kilometers, indexed by resolution.
	hexAreaAvgKm2 = [MaxResolution + 1]float64{
		4.357449416078383e+06, 6.097884417941332e+05, 8.680178039899720e+04,
		1.239343465508816e+04, 1.770347654491307e+03, 2.529038581819449e+02,
		3.612906216441245e+01, 5.161293359717191e+00, 7.373275975944177e-01,
		1.053325134272067e-01, 1.504750190766435e-02, 2.149643129451879e-03,
		3.070918756316060e-04, 4.387026794728296e-05, 6.267181135324313e-06,
		8.953115907605790e-07,
	}

	// hexAreaAvgM2 holds the average hexagon area at each resolution in square
	// meters, indexed by resolution.
	hexAreaAvgM2 = [MaxResolution + 1]float64{
		4.357449416078390e+12, 6.097884417941339e+11, 8.680178039899731e+10,
		1.239343465508818e+10, 1.770347654491309e+09, 2.529038581819452e+08,
		3.612906216441250e+07, 5.161293359717198e+06, 7.373275975944188e+05,
		1.053325134272069e+05, 1.504750190766437e+04, 2.149643129451882e+03,
		3.070918756316063e+02, 4.387026794728301e+01, 6.267181135324322e+00,
		8.953115907605802e-01,
	}

	// hexEdgeLenAvgKm holds the average hexagon edge length at each resolution in
	// kilometers, indexed by resolution.
	hexEdgeLenAvgKm = [MaxResolution + 1]float64{
		1281.256011, 483.0568391, 182.5129565, 68.97922179,
		26.07175968, 9.854090990, 3.724532667, 1.406475763,
		0.531414010, 0.200786148, 0.075863783, 0.028663897,
		0.010830188, 0.004092010, 0.001546100, 0.000584169,
	}

	// hexEdgeLenAvgM holds the average hexagon edge length at each resolution in
	// meters, indexed by resolution.
	hexEdgeLenAvgM = [MaxResolution + 1]float64{
		1281256.011, 483056.8391, 182512.9565, 68979.22179,
		26071.75968, 9854.090990, 3724.532667, 1406.475763,
		531.4140101, 200.7861476, 75.86378287, 28.66389748,
		10.83018784, 4.092010473, 1.546099657, 0.584168630,
	}
)

// NewLatLng creates a LatLng from a latitude and longitude in degrees.
func NewLatLng(lat, lng float64) LatLng {
	return LatLng{Lat: lat, Lng: lng}
}

// Cell returns the cell at the given resolution that contains the coordinate.
func (g LatLng) Cell(res int) (Cell, error) {
	return LatLngToCell(g, res)
}

// String returns the coordinate formatted as "(lat, lng)" in degrees.
func (g LatLng) String() string {
	buf := make([]byte, 0, latLngStringSize)
	buf = append(buf, '(')
	buf = strconv.AppendFloat(buf, g.Lat, 'f', latLngFloatPrecision, 64)
	buf = append(buf, ',', ' ')
	buf = strconv.AppendFloat(buf, g.Lng, 'f', latLngFloatPrecision, 64)
	buf = append(buf, ')')

	return string(buf)
}

// LatLngToCellString returns the string form of the cell at resolution that
// contains the coordinate. It is a convenience wrapper for LatLngToCell followed
// by Cell.String.
func LatLngToCellString(latitude, longitude float64, res int) (string, error) {
	cell, err := NewLatLng(latitude, longitude).Cell(res)
	if err != nil {
		return "", err
	}

	return cell.String(), nil
}

// LatLngToCell returns the Cell at resolution for a geographic coordinate.
func LatLngToCell(latLng LatLng, res int) (Cell, error) {
	if res < 0 || res > MaxResolution {
		return 0, ErrResolutionDomain
	}

	if math.IsNaN(latLng.Lat) || math.IsInf(latLng.Lat, 0) ||
		math.IsNaN(latLng.Lng) || math.IsInf(latLng.Lng, 0) {
		return 0, ErrLatLngDomain
	}

	lat := latLng.Lat * DegsToRads
	lng := latLng.Lng * DegsToRads
	v := latLngToVec3(lat, lng)
	fijk := v.toFaceIjk(res)

	return fijk.toH3(res)
}

// CellToLatLng returns the geographic center point of a cell in degrees.
func CellToLatLng(c Cell) (LatLng, error) {
	return c.LatLng()
}

// LatLng returns the geographic center point of the cell in degrees.
func (c Cell) LatLng() (LatLng, error) {
	fijk, err := c.toFaceIjk()
	if err != nil {
		return LatLng{}, err
	}

	return fijk.toVec3(c.Resolution()).toLatLng(), nil
}

// --- Distances ---

// GreatCircleDistanceRads returns the great-circle distance between two points
// in radians, using the haversine formula. The points are given in degrees.
func GreatCircleDistanceRads(a, b LatLng) float64 {
	aLat := a.Lat * DegsToRads
	aLng := a.Lng * DegsToRads
	bLat := b.Lat * DegsToRads
	bLng := b.Lng * DegsToRads

	sinLat := math.Sin((bLat - aLat) / 2)
	sinLng := math.Sin((bLng - aLng) / 2)

	h := sinLat*sinLat + math.Cos(aLat)*math.Cos(bLat)*sinLng*sinLng

	return 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

// GreatCircleDistanceKm returns the great-circle distance between two points in
// kilometers. The points are given in degrees.
func GreatCircleDistanceKm(a, b LatLng) float64 {
	return GreatCircleDistanceRads(a, b) * earthRadiusKm
}

// GreatCircleDistanceM returns the great-circle distance between two points in
// meters. The points are given in degrees.
func GreatCircleDistanceM(a, b LatLng) float64 {
	return GreatCircleDistanceKm(a, b) * 1000
}

// --- Edge lengths ---

// EdgeLengthRads returns the exact length of the directed edge in radians, the
// sum of the great-circle distances between consecutive boundary vertexes.
func EdgeLengthRads(e DirectedEdge) (float64, error) {
	boundary, err := e.Boundary()
	if err != nil {
		return 0, err
	}

	length := 0.0
	for i := 0; i < len(boundary)-1; i++ {
		length += GreatCircleDistanceRads(boundary[i], boundary[i+1])
	}

	return length, nil
}

// EdgeLengthKm returns the exact length of the directed edge in kilometers.
func EdgeLengthKm(e DirectedEdge) (float64, error) {
	rads, err := EdgeLengthRads(e)

	return rads * earthRadiusKm, err
}

// EdgeLengthM returns the exact length of the directed edge in meters.
func EdgeLengthM(e DirectedEdge) (float64, error) {
	km, err := EdgeLengthKm(e)

	return km * 1000, err
}

// HexagonAreaAvgKm2 returns the average area of a hexagon at the given
// resolution in square kilometers.
func HexagonAreaAvgKm2(res int) (float64, error) {
	if res < 0 || res > MaxResolution {
		return 0, ErrResolutionDomain
	}

	return hexAreaAvgKm2[res], nil
}

// HexagonAreaAvgM2 returns the average area of a hexagon at the given resolution
// in square meters.
func HexagonAreaAvgM2(res int) (float64, error) {
	if res < 0 || res > MaxResolution {
		return 0, ErrResolutionDomain
	}

	return hexAreaAvgM2[res], nil
}

// HexagonEdgeLengthAvgKm returns the average edge length of a hexagon at the
// given resolution in kilometers.
func HexagonEdgeLengthAvgKm(res int) (float64, error) {
	if res < 0 || res > MaxResolution {
		return 0, ErrResolutionDomain
	}

	return hexEdgeLenAvgKm[res], nil
}

// HexagonEdgeLengthAvgM returns the average edge length of a hexagon at the given
// resolution in meters.
func HexagonEdgeLengthAvgM(res int) (float64, error) {
	if res < 0 || res > MaxResolution {
		return 0, ErrResolutionDomain
	}

	return hexEdgeLenAvgM[res], nil
}

// --- Vec3d math ---

func latLngToVec3(lat, lng float64) vec3d {
	r := math.Cos(lat)

	return vec3d{
		x: math.Cos(lng) * r,
		y: math.Sin(lng) * r,
		z: math.Sin(lat),
	}
}

// toLatLng converts a 3D unit vector on the sphere into a geographic coordinate
// in degrees.
func (v vec3d) toLatLng() LatLng {
	return LatLng{
		Lat: RadsToDegs * math.Asin(v.z),
		Lng: RadsToDegs * math.Atan2(v.y, v.x),
	}
}

func (v vec3d) dot(b vec3d) float64 {
	return v.x*b.x + v.y*b.y + v.z*b.z
}

func (v vec3d) cross(b vec3d) vec3d {
	return vec3d{
		x: v.y*b.z - v.z*b.y,
		y: v.z*b.x - v.x*b.z,
		z: v.x*b.y - v.y*b.x,
	}
}

// linComb returns the linear combination scaleA·v + scaleB·other.
func (v vec3d) linComb(scaleA, scaleB float64, other vec3d) vec3d {
	return vec3d{
		x: scaleA*v.x + scaleB*other.x,
		y: scaleA*v.y + scaleB*other.y,
		z: scaleA*v.z + scaleB*other.z,
	}
}

func (v *vec3d) normalize() {
	sq := v.dot(*v)

	s := 0.0
	if sq > 0 {
		s = 1.0 / math.Sqrt(sq)
	}

	v.x *= s
	v.y *= s
	v.z *= s
}

func (v vec3d) distSq(b vec3d) float64 {
	d := v.linComb(1.0, -1.0, b)

	return d.dot(d)
}
