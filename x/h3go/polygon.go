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

// dblEpsilon is the difference between 1 and the least value greater than 1 that
// is representable as a float64. It nudges the ray-casting test off exact vertex
// latitudes and longitudes, matching the H3 C library's use of DBL_EPSILON.
const dblEpsilon = 2.2204460492503131e-16

// normalizeLng shifts a negative longitude (in radians) east by a full turn when
// the loop is transmeridian, so a crossing loop's coordinates compare in one
// frame.
func normalizeLng(lng float64, isTransmeridian bool) float64 {
	if isTransmeridian && lng < 0 {
		return lng + twoPiRad
	}

	return lng
}

// pointInsideGeoLoop reports whether the loop contains the coordinate, using a
// ray-casting test. It fails fast when the point is outside the loop's bounding
// box.
//
// The bounding-box test runs in degrees, the units of the public GeoLoop, but
// the ray casting converts to radians first: the dblEpsilon vertex nudges below
// only break ties correctly at radian coordinate scale, where dblEpsilon exceeds
// half a ULP. At degree scale (a coordinate ~57x larger) the nudge rounds away,
// which would drop cells whose center latitude exactly matches a polygon vertex
// (see uber/h3#595).
func pointInsideGeoLoop(loop GeoLoop, box bbox, coord LatLng) bool {
	if !box.contains(coord) {
		return false
	}

	isTransmeridian := box.isTransmeridian()
	contains := false

	lat := coord.Lat * DegsToRads
	lng := normalizeLng(coord.Lng*DegsToRads, isTransmeridian)

	for i := range loop {
		a := LatLng{Lat: loop[i].Lat * DegsToRads, Lng: loop[i].Lng * DegsToRads}
		next := loop[(i+1)%len(loop)]
		b := LatLng{Lat: next.Lat * DegsToRads, Lng: next.Lng * DegsToRads}

		// Ray casting requires the second point to be the higher one.
		if a.Lat > b.Lat {
			a, b = b, a
		}

		// Nudge north off an exact vertex latitude to avoid counting the ray
		// crossing the same vertex twice on successive segments.
		if lat == a.Lat || lat == b.Lat {
			lat += dblEpsilon
		}

		// Skip segments the horizontal ray cannot reach.
		if lat < a.Lat || lat > b.Lat {
			continue
		}

		aLng := normalizeLng(a.Lng, isTransmeridian)
		bLng := normalizeLng(b.Lng, isTransmeridian)

		// Bias westerly on an exact longitude match to break ties consistently.
		if aLng == lng || bLng == lng {
			lng -= dblEpsilon
		}

		// Longitude of the segment at the point's latitude.
		ratio := (lat - a.Lat) / (b.Lat - a.Lat)
		testLng := normalizeLng(aLng+(bLng-aLng)*ratio, isTransmeridian)

		if testLng > lng {
			contains = !contains
		}
	}

	return contains
}

// pointInsidePolygon reports whether the polygon contains the coordinate: inside
// the outer loop and outside every hole. bboxes holds the outer loop's bounding
// box followed by one per hole.
func pointInsidePolygon(polygon GeoPolygon, bboxes []bbox, coord LatLng) bool {
	contains := pointInsideGeoLoop(polygon.GeoLoop, bboxes[0], coord)
	if !contains {
		return false
	}

	for i := range polygon.Holes {
		if pointInsideGeoLoop(polygon.Holes[i], bboxes[i+1], coord) {
			return false
		}
	}

	return true
}

// lineCrossesLine reports whether segment a1→a2 intersects segment b1→b2. This
// is a purely Cartesian test that ignores antimeridian wrapping and poles.
func lineCrossesLine(a1, a2, b1, b2 LatLng) bool {
	denom := (b2.Lng-b1.Lng)*(a2.Lat-a1.Lat) - (b2.Lat-b1.Lat)*(a2.Lng-a1.Lng)
	if denom == 0 {
		return false
	}

	test := ((b2.Lat-b1.Lat)*(a1.Lng-b1.Lng) - (b2.Lng-b1.Lng)*(a1.Lat-b1.Lat)) / denom
	if test < 0 || test > 1 {
		return false
	}

	test = ((a2.Lat-a1.Lat)*(a1.Lng-b1.Lng) - (a2.Lng-a1.Lng)*(a1.Lat-b1.Lat)) / denom

	return test >= 0 && test <= 1
}

// cellBoundaryCrossesGeoLoop reports whether any segment of the cell boundary
// intersects any segment of the loop. Crossing means line-segment intersection;
// it does not include containment.
func cellBoundaryCrossesGeoLoop(loop GeoLoop, loopBBox bbox, boundary CellBoundary, boundaryBBox bbox) bool {
	if !loopBBox.overlaps(boundaryBBox) {
		return false
	}

	loopNorm, boundaryNorm := normalizationFor(loopBBox, boundaryBBox)

	normalBoundary := make([]LatLng, len(boundary))
	for i := range boundary {
		normalBoundary[i] = LatLng{Lat: boundary[i].Lat, Lng: applyNormalization(boundary[i].Lng, boundaryNorm)}
	}

	normalBoundaryBBox := bbox{
		north: boundaryBBox.north,
		south: boundaryBBox.south,
		east:  applyNormalization(boundaryBBox.east, boundaryNorm),
		west:  applyNormalization(boundaryBBox.west, boundaryNorm),
	}

	for i := range loop {
		loop1 := LatLng{Lat: loop[i].Lat, Lng: applyNormalization(loop[i].Lng, loopNorm)}
		next := loop[(i+1)%len(loop)]
		loop2 := LatLng{Lat: next.Lat, Lng: applyNormalization(next.Lng, loopNorm)}

		// Skip segments that cannot reach the boundary's bounding box.
		if (loop1.Lat >= normalBoundaryBBox.north && loop2.Lat >= normalBoundaryBBox.north) ||
			(loop1.Lat <= normalBoundaryBBox.south && loop2.Lat <= normalBoundaryBBox.south) ||
			(loop1.Lng <= normalBoundaryBBox.west && loop2.Lng <= normalBoundaryBBox.west) ||
			(loop1.Lng >= normalBoundaryBBox.east && loop2.Lng >= normalBoundaryBBox.east) {
			continue
		}

		for j := range normalBoundary {
			other := normalBoundary[(j+1)%len(normalBoundary)]
			if lineCrossesLine(loop1, loop2, normalBoundary[j], other) {
				return true
			}
		}
	}

	return false
}

// cellBoundaryInsidePolygon reports whether the cell boundary is completely
// contained by the polygon: its first vertex is inside, it crosses neither the
// outer loop nor any hole, and it contains no hole.
func cellBoundaryInsidePolygon(polygon GeoPolygon, bboxes []bbox, boundary CellBoundary, boundaryBBox bbox) bool {
	// Fails fast when the first vertex is outside the bounding box.
	if !pointInsidePolygon(polygon, bboxes, boundary[0]) {
		return false
	}

	if cellBoundaryCrossesGeoLoop(polygon.GeoLoop, bboxes[0], boundary, boundaryBBox) {
		return false
	}

	boundaryLoop := GeoLoop(boundary)

	for i := range polygon.Holes {
		hole := polygon.Holes[i]
		if len(hole) > 0 &&
			(pointInsideGeoLoop(boundaryLoop, boundaryBBox, hole[0]) ||
				cellBoundaryCrossesGeoLoop(hole, bboxes[i+1], boundary, boundaryBBox)) {
			return false
		}
	}

	return true
}

// cellBoundaryCrossesPolygon reports whether any part of the cell boundary
// crosses the polygon's outer loop or any hole. Crossing means line-segment
// intersection; it does not include containment.
func cellBoundaryCrossesPolygon(polygon GeoPolygon, bboxes []bbox, boundary CellBoundary, boundaryBBox bbox) bool {
	if cellBoundaryCrossesGeoLoop(polygon.GeoLoop, bboxes[0], boundary, boundaryBBox) {
		return true
	}

	for i := range polygon.Holes {
		if cellBoundaryCrossesGeoLoop(polygon.Holes[i], bboxes[i+1], boundary, boundaryBBox) {
			return true
		}
	}

	return false
}
