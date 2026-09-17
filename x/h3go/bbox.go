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

import "math"

// Longitude span constants in degrees, the units of the public API. They stand
// in for the H3 C library's radian M_PI / M_2PI when detecting and normalizing
// antimeridian-crossing geometry.
const (
	piDeg     = 180.0
	twoPiDeg  = 360.0
	halfPiDeg = 90.0

	// twoPiRad is a full longitude turn in radians, used to normalize
	// antimeridian-crossing geometry where the math must run at the radian scale
	// of the H3 C library.
	twoPiRad = 2 * math.Pi
)

// longitudeNormalization selects how a longitude is shifted so two bounding
// boxes, either of which may cross the antimeridian, can be compared in one
// frame of reference.
type longitudeNormalization int

const (
	normalizeNone longitudeNormalization = iota
	normalizeEast
	normalizeWest
)

// bbox is a geographic bounding box with degree coordinates, matching the units
// of the public GeoPolygon. east < west indicates a box crossing the
// antimeridian.
type bbox struct {
	north, south, east, west float64
}

// isTransmeridian reports whether the bounding box crosses the antimeridian.
func (b bbox) isTransmeridian() bool {
	return b.east < b.west
}

// contains reports whether the bounding box contains the point.
func (b bbox) contains(point LatLng) bool {
	if point.Lat < b.south || point.Lat > b.north {
		return false
	}

	if b.isTransmeridian() {
		return point.Lng >= b.west || point.Lng <= b.east
	}

	return point.Lng >= b.west && point.Lng <= b.east
}

// toBbox computes the bounding box of a loop of coordinates. It does not
// support loops with adjacent points more than 180° of longitude apart (treated
// as antimeridian crossings) or loops containing a pole.
func (loop GeoLoop) toBbox() bbox {
	if len(loop) == 0 {
		return bbox{}
	}

	out := bbox{north: -math.MaxFloat64, south: math.MaxFloat64, east: -math.MaxFloat64, west: math.MaxFloat64}
	minPosLng := math.MaxFloat64
	maxNegLng := -math.MaxFloat64
	isTransmeridian := false

	for i := range loop {
		coord := loop[i]
		next := loop[(i+1)%len(loop)]

		out.south = min(out.south, coord.Lat)
		out.west = min(out.west, coord.Lng)
		out.north = max(out.north, coord.Lat)
		out.east = max(out.east, coord.Lng)

		if coord.Lng > 0 && coord.Lng < minPosLng {
			minPosLng = coord.Lng
		}

		if coord.Lng < 0 && coord.Lng > maxNegLng {
			maxNegLng = coord.Lng
		}

		if math.Abs(coord.Lng-next.Lng) > piDeg {
			isTransmeridian = true
		}
	}

	if isTransmeridian {
		out.east = maxNegLng
		out.west = minPosLng
	}

	return out
}

// toBboxes returns the bounding box for the outer loop followed by
// one for each hole, in order.
func (polygon GeoPolygon) toBboxes() []bbox {
	bboxes := make([]bbox, len(polygon.Holes)+1)
	bboxes[0] = polygon.GeoLoop.toBbox()

	for i := range polygon.Holes {
		bboxes[i+1] = polygon.Holes[i].toBbox()
	}

	return bboxes
}

// hexRadiusKm returns the radius of a cell in kilometers, the distance from its
// center to its first boundary vertex. It is only called with known-valid cells
// (the pentagons of a resolution), so any projection error is ignored, matching
// the H3 C library's error-free _hexRadiusKm.
func (c Cell) hexRadiusKm() float64 {
	fijk, _ := c.toFaceIjk()

	res := c.Resolution()
	center := fijk.toVec3(res).toLatLng()

	var boundary CellBoundary
	if c.IsPentagon() {
		boundary = fijk.pentToCellBoundary(res, 0, numPentVerts)
	} else {
		boundary = fijk.toCellBoundary(res, 0, numHexVerts)
	}

	return GreatCircleDistanceKm(center, boundary[0])
}

// bboxHexEstimate estimates the number of cells of the given resolution that fit
// within the Cartesian-projected bounding box.
func bboxHexEstimate(box bbox, res int) (int, error) {
	pentagons, err := Pentagons(res)
	if err != nil {
		return 0, err
	}

	pentagonRadiusKm := pentagons[0].hexRadiusKm()

	// Area of a regular hexagon is 3/2*sqrt(3) * r * r. The pentagon has the most
	// distortion (smallest edges), shrunk by 20% in case the box perfectly bounds
	// a pentagon.
	pentagonAreaKm2 := 0.8 * (2.59807621135 * pentagonRadiusKm * pentagonRadiusKm)

	corner1 := LatLng{Lat: box.north, Lng: box.east}
	corner2 := LatLng{Lat: box.south, Lng: box.west}
	diagonalKm := GreatCircleDistanceKm(corner1, corner2)

	lngDiff := math.Abs(corner1.Lng - corner2.Lng)
	latDiff := math.Abs(corner1.Lat - corner2.Lat)

	if lngDiff == 0 || latDiff == 0 {
		return 0, ErrFailed
	}

	length := max(lngDiff, latDiff)
	width := min(lngDiff, latDiff)
	ratio := length / width

	// Derived constant, clamped to 3 as higher values drag the estimate to zero.
	area := diagonalKm * diagonalKm / min(3.0, ratio)

	estimate := math.Ceil(area / pentagonAreaKm2)
	if math.IsInf(estimate, 0) || math.IsNaN(estimate) {
		return 0, ErrFailed
	}

	return max(int(estimate), 1), nil
}

// lineHexEstimate estimates the number of cells of the given resolution needed
// to trace the Cartesian-projected line between two points.
func lineHexEstimate(origin, destination LatLng, res int) (int, error) {
	pentagons, err := Pentagons(res)
	if err != nil {
		return 0, err
	}

	pentagonRadiusKm := pentagons[0].hexRadiusKm()

	distKm := GreatCircleDistanceKm(origin, destination)

	distCeil := math.Ceil(distKm / (2 * pentagonRadiusKm))
	if math.IsInf(distCeil, 0) || math.IsNaN(distCeil) {
		return 0, ErrFailed
	}

	return max(int(distCeil), 1), nil
}

// width returns the longitude span of the bounding box in degrees, accounting
// for an antimeridian crossing.
func (b bbox) width() float64 {
	if b.isTransmeridian() {
		return b.east - b.west + twoPiDeg
	}

	return b.east - b.west
}

// height returns the latitude span of the bounding box in degrees.
func (b bbox) height() float64 {
	return b.north - b.south
}

// applyNormalization shifts a longitude east or west by a full turn so that two
// boxes can be compared in a common frame; normalizeNone leaves it unchanged.
func applyNormalization(lng float64, normalization longitudeNormalization) float64 {
	switch normalization {
	case normalizeEast:
		if lng < 0 {
			return lng + twoPiDeg
		}
	case normalizeWest:
		if lng > 0 {
			return lng - twoPiDeg
		}
	case normalizeNone:
	}

	return lng
}

// normalizationFor determines the longitude normalization scheme for two
// bounding boxes, either or both of which may cross the antimeridian, so they
// can be operated on with standard Cartesian comparisons.
func normalizationFor(first, second bbox) (longitudeNormalization, longitudeNormalization) {
	firstTrans := first.isTransmeridian()
	secondTrans := second.isTransmeridian()
	firstTrendsEast := first.west-second.east < second.west-first.east

	var firstNorm, secondNorm longitudeNormalization

	switch {
	case !firstTrans:
		firstNorm = normalizeNone
	case secondTrans, firstTrendsEast:
		firstNorm = normalizeEast
	default:
		firstNorm = normalizeWest
	}

	switch {
	case !secondTrans:
		secondNorm = normalizeNone
	case firstTrans:
		secondNorm = normalizeEast
	case firstTrendsEast:
		secondNorm = normalizeWest
	default:
		secondNorm = normalizeEast
	}

	return firstNorm, secondNorm
}

// overlaps reports whether two bounding boxes overlap, accounting for
// antimeridian crossings.
func (b bbox) overlaps(other bbox) bool {
	if b.north < other.south || b.south > other.north {
		return false
	}

	bNorm, otherNorm := normalizationFor(b, other)

	if applyNormalization(b.east, bNorm) < applyNormalization(other.west, otherNorm) ||
		applyNormalization(b.west, bNorm) > applyNormalization(other.east, otherNorm) {
		return false
	}

	return true
}

// containsBBox reports whether the bounding box fully contains another box,
// accounting for antimeridian crossings.
func (b bbox) containsBBox(other bbox) bool {
	if b.north < other.north || b.south > other.south {
		return false
	}

	bNorm, otherNorm := normalizationFor(b, other)

	return applyNormalization(b.west, bNorm) <= applyNormalization(other.west, otherNorm) &&
		applyNormalization(b.east, bNorm) >= applyNormalization(other.east, otherNorm)
}

// toCellBoundary converts the bounding box to a four-vertex cell boundary in
// counter-clockwise order.
func (b bbox) toCellBoundary() CellBoundary {
	return CellBoundary{
		{Lat: b.north, Lng: b.east},
		{Lat: b.north, Lng: b.west},
		{Lat: b.south, Lng: b.west},
		{Lat: b.south, Lng: b.east},
	}
}

// scaled returns the bounding box scaled about its center by the given factor,
// normalized to the latitude and longitude domains. Both width and height are
// scaled by the factor (so the area scales by factor squared).
func (b bbox) scaled(scale float64) bbox {
	widthBuffer := (b.width()*scale - b.width()) * 0.5
	heightBuffer := (b.height()*scale - b.height()) * 0.5

	b.north += heightBuffer
	if b.north > halfPiDeg {
		b.north = halfPiDeg
	}

	b.south -= heightBuffer
	if b.south < -halfPiDeg {
		b.south = -halfPiDeg
	}

	b.east += widthBuffer
	if b.east > piDeg {
		b.east -= twoPiDeg
	}

	if b.east < -piDeg {
		b.east += twoPiDeg
	}

	b.west -= widthBuffer
	if b.west > piDeg {
		b.west -= twoPiDeg
	}

	if b.west < -piDeg {
		b.west += twoPiDeg
	}

	return b
}
