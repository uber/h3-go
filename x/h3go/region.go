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

// polygonToCellsBuffer is added to the estimated cell count to cover small
// polygons near icosahedron edges at odd resolutions, where line tracing needs a
// little more room than the estimator provides.
const polygonToCellsBuffer = 12

type (
	// GeoLoop is an ordered list of geographic coordinates in degrees describing
	// a closed loop; the final point is implicitly connected back to the first.
	GeoLoop []LatLng

	// GeoPolygon is a GeoLoop outer boundary with zero or more GeoLoop holes.
	GeoPolygon struct {
		GeoLoop GeoLoop
		Holes   []GeoLoop
	}
)

// maxPolygonToCellsSize returns an upper bound on the number of cells that
// PolygonToCells may produce for the polygon at the given resolution. It is the
// larger of the bounding-box cell estimate and the total vertex count, plus a
// small buffer.
func maxPolygonToCellsSize(polygon GeoPolygon, res int) (int, error) {
	numHexagons, err := bboxHexEstimate(polygon.GeoLoop.toBbox(), res)
	if err != nil {
		return 0, err
	}

	// The estimate usually exceeds the vertex count, but guard the rare case it
	// does not.
	totalVerts := len(polygon.GeoLoop)
	for i := range polygon.Holes {
		totalVerts += len(polygon.Holes[i])
	}

	numHexagons = max(numHexagons, totalVerts)

	return numHexagons + polygonToCellsBuffer, nil
}

// getEdgeHexagons traces a loop with cells of the given resolution, adding every
// cell whose center the loop passes through to the search set. These seed the
// flood fill in PolygonToCells.
func getEdgeHexagons(loop GeoLoop, res int, search map[Cell]bool) error {
	for i := range loop {
		origin := loop[i]
		destination := loop[(i+1)%len(loop)]

		numHexes, err := lineHexEstimate(origin, destination, res)
		if err != nil {
			return err
		}

		invNumHexes := 1.0 / float64(numHexes)
		for j := range numHexes {
			interpolate := LatLng{
				Lat: origin.Lat*float64(numHexes-j)*invNumHexes + destination.Lat*float64(j)*invNumHexes,
				Lng: origin.Lng*float64(numHexes-j)*invNumHexes + destination.Lng*float64(j)*invNumHexes,
			}

			// res and finiteness are already validated by lineHexEstimate above,
			// so this conversion cannot fail here.
			cell, _ := LatLngToCell(interpolate, res)
			search[cell] = true
		}
	}

	return nil
}

// PolygonToCells returns the cells of the given resolution whose centers fall
// within the polygon. The polygon is considered in Cartesian (lat/lng) space:
// the outer loop minus any holes. Output ordering is not significant.
//
// The algorithm traces the loops with cells, then flood-fills outward from those
// seeds, keeping every neighbor whose center is contained, until no new cells are
// found. This means two adjacent polygons with no overlap produce disjoint cell
// sets.
func PolygonToCells(polygon GeoPolygon, res int) ([]Cell, error) {
	if len(polygon.GeoLoop) == 0 {
		return nil, nil
	}

	bboxes := polygon.toBboxes()

	// 1. Trace the outer loop and any holes to seed the search set. Tracing the
	// first loop surfaces an invalid resolution.
	search := make(map[Cell]bool)
	for _, loop := range append([]GeoLoop{polygon.GeoLoop}, polygon.Holes...) {
		if err := getEdgeHexagons(loop, res, search); err != nil {
			return nil, err
		}
	}

	// A degenerate bounding box (zero width or height) cannot be filled.
	sizeHint, err := maxPolygonToCellsSize(polygon, res)
	if err != nil {
		return nil, err
	}

	// 2. Flood fill: from each search cell, test it and its neighbors for
	// containment, and use the newly contained cells as the next search set.
	found := make(map[Cell]bool, sizeHint)

	searchList := make([]Cell, 0, len(search))
	for cell := range search {
		searchList = append(searchList, cell)
	}

	for len(searchList) > 0 {
		searchList = polygonFloodStep(polygon, bboxes, searchList, found)
	}

	out := make([]Cell, 0, len(found))
	for cell := range found {
		out = append(out, cell)
	}

	return out, nil
}

// polygonFloodStep expands one generation of the polygon flood fill: for each
// cell in searchList, it tests the cell and its neighbors and records those whose
// center is inside the polygon, returning the newly found cells. Every cell here
// comes from a prior LatLngToCell or grid-disk step, so it is always valid and
// the projection calls cannot fail.
func polygonFloodStep(polygon GeoPolygon, bboxes []bbox, searchList []Cell, found map[Cell]bool) []Cell {
	var nextSearch []Cell

	// disk is reused across cells so the grid-disk lookup below allocates once
	// rather than once per search cell.
	var disk []Cell

	for _, searchHex := range searchList {
		disk, _ = searchHex.gridDiskInto(1, disk[:0])

		for _, hex := range disk {
			if found[hex] {
				continue
			}

			center, _ := hex.LatLng()
			if !pointInsidePolygon(polygon, bboxes, center) {
				continue
			}

			found[hex] = true

			nextSearch = append(nextSearch, hex)
		}
	}

	return nextSearch
}

// Cells returns the cells of the given resolution whose centers fall within the
// polygon.
func (p GeoPolygon) Cells(res int) ([]Cell, error) {
	return PolygonToCells(p, res)
}
