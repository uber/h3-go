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
	"slices"
	"sort"
)

// Counter-clockwise orderings of a cell's directed edges into its linked loop.
// idxHex is for hexagons (six edges); idxPent is for pentagons (five edges).
var (
	idxHex  = [numCellEdges]int{0, 4, 3, 5, 1, 2}
	idxPent = [numCellEdges - 1]int{0, 1, 3, 2, 4}
)

// arc is one directed edge of a cell during multipolygon assembly. Arcs form
// doubly-linked loops (counter-clockwise) and a union-find forest whose roots
// identify connected components (each component becomes one polygon).
type arc struct {
	parent  *arc
	prev    *arc
	next    *arc
	id      DirectedEdge
	rank    int
	removed bool
	visited bool
}

// root returns the representative arc of the connected component, compressing
// the path along the way.
func (a *arc) root() *arc {
	if a.parent == a {
		return a
	}

	a.parent = a.parent.root()

	return a.parent
}

// union merges the connected components of two arcs, attaching the lower-rank
// root under the higher-rank one.
func union(first, second *arc) {
	first = first.root()
	second = second.root()

	if first.rank < second.rank {
		first, second = second, first
	}

	if first != second {
		first.rank += second.rank
		second.parent = first
	}
}

// validateCellSet checks that the cells are all valid, share one resolution, and
// contain no duplicates, matching the H3 C library's contract.
func validateCellSet(cells []Cell) error {
	res := cells[0].Resolution()
	for _, cell := range cells {
		if !cell.IsValid() {
			return ErrCellInvalid
		}

		if cell.Resolution() != res {
			return ErrResolutionMismatch
		}
	}

	if len(cells) >= 2 {
		sorted := slices.Clone(cells)
		slices.Sort(sorted)

		for i := 1; i < len(sorted); i++ {
			if sorted[i] == sorted[i-1] {
				return ErrDuplicateInput
			}
		}
	}

	return nil
}

// buildArcs creates the arcs for every cell, linking each cell's edges into a
// counter-clockwise loop and indexing them by edge id for reverse lookup. Every
// arc in a cell starts in the cell's own connected component.
func buildArcs(cells []Cell) ([]*arc, map[DirectedEdge]*arc) {
	var arcs []*arc

	index := make(map[DirectedEdge]*arc)

	for _, cell := range cells {
		// cell is valid here, so enumerating its edges cannot fail.
		edges, _ := cell.DirectedEdges()
		count := len(edges)

		block := make([]*arc, count)
		for i := range block {
			block[i] = &arc{id: edges[i], rank: 1}
		}

		for i := range block {
			block[i].parent = block[0]
		}

		order := idxHex[:]
		if count == numCellEdges-1 {
			order = idxPent[:]
		}

		for i := range block {
			cur := order[i]
			prev := order[(i-1+count)%count]
			next := order[(i+1)%count]
			block[cur].prev = block[prev]
			block[cur].next = block[next]
		}

		for i := range block {
			arcs = append(arcs, block[i])
			index[block[i].id] = block[i]
		}
	}

	return arcs, index
}

// cancelArcPairs removes each pair of opposite edges shared by two adjacent
// cells, stitching the linked loops back together and merging the two arcs'
// connected components. What remains are the outline loops of the cell set.
func cancelArcPairs(arcs []*arc, index map[DirectedEdge]*arc) {
	for _, current := range arcs {
		if current.removed {
			continue
		}

		// current.id is a valid edge, so reversing it cannot fail.
		reversed, _ := current.id.Reverse()

		opposite, ok := index[reversed]
		if !ok {
			continue
		}

		current.removed = true
		opposite.removed = true

		current.next.prev = opposite.prev
		current.prev.next = opposite.next
		opposite.next.prev = current.prev
		opposite.prev.next = current.next

		union(current, opposite)
	}
}

// outlineLoop holds one assembled boundary loop with the connected component it
// belongs to and its area, used to order loops within and across polygons.
type outlineLoop struct {
	loop GeoLoop
	root DirectedEdge
	area float64
}

// buildOutlineLoops walks the remaining arcs into boundary loops, recording each
// loop's connected component and enclosed area. Loops are sorted by component,
// then by area so that each polygon's loops are contiguous with its outer loop
// (smallest enclosed area) first.
func buildOutlineLoops(arcs []*arc) []outlineLoop {
	for _, current := range arcs {
		current.visited = false
	}

	var loops []outlineLoop

	for _, start := range arcs {
		if start.visited || start.removed {
			continue
		}

		var verts GeoLoop

		current := start
		for {
			// current.id is valid, so its boundary cannot fail.
			boundary, _ := current.id.Boundary()
			verts = append(verts, boundary[:len(boundary)-1]...)
			current.visited = true
			current = current.next

			if current.id == start.id {
				break
			}
		}

		loops = append(loops, outlineLoop{
			root: start.root().id,
			loop: verts,
			area: CellBoundary(verts).areaRads2(),
		})
	}

	sort.SliceStable(loops, func(i, j int) bool {
		if loops[i].root != loops[j].root {
			return loops[i].root < loops[j].root
		}

		return loops[i].area < loops[j].area
	})

	return loops
}

// assembleMultiPolygon groups contiguous same-component loops into polygons
// (outer loop first, the rest holes) and orders the polygons by decreasing outer
// loop area.
func assembleMultiPolygon(loops []outlineLoop) []GeoPolygon {
	type sortablePoly struct {
		poly      GeoPolygon
		outerArea float64
	}

	var polys []sortablePoly

	for i := 0; i < len(loops); {
		j := i
		for j < len(loops) && loops[j].root == loops[i].root {
			j++
		}

		poly := GeoPolygon{GeoLoop: loops[i].loop}
		for k := i + 1; k < j; k++ {
			poly.Holes = append(poly.Holes, loops[k].loop)
		}

		polys = append(polys, sortablePoly{poly: poly, outerArea: loops[i].area})
		i = j
	}

	sort.SliceStable(polys, func(i, j int) bool {
		return polys[i].outerArea > polys[j].outerArea
	})

	out := make([]GeoPolygon, len(polys))
	for i := range polys {
		out[i] = polys[i].poly
	}

	return out
}

// globeMultiPolygon returns the eight-triangle representation of the entire
// globe, used when the cell set covers the whole sphere and leaves no outline.
func globeMultiPolygon() []GeoPolygon {
	verts := [8][3]LatLng{
		{{Lat: halfPiDeg, Lng: 0}, {Lat: 0, Lng: 0}, {Lat: 0, Lng: halfPiDeg}},
		{{Lat: halfPiDeg, Lng: 0}, {Lat: 0, Lng: halfPiDeg}, {Lat: 0, Lng: piDeg}},
		{{Lat: halfPiDeg, Lng: 0}, {Lat: 0, Lng: piDeg}, {Lat: 0, Lng: -halfPiDeg}},
		{{Lat: halfPiDeg, Lng: 0}, {Lat: 0, Lng: -halfPiDeg}, {Lat: 0, Lng: 0}},
		{{Lat: -halfPiDeg, Lng: 0}, {Lat: 0, Lng: 0}, {Lat: 0, Lng: -halfPiDeg}},
		{{Lat: -halfPiDeg, Lng: 0}, {Lat: 0, Lng: -halfPiDeg}, {Lat: 0, Lng: -piDeg}},
		{{Lat: -halfPiDeg, Lng: 0}, {Lat: 0, Lng: -piDeg}, {Lat: 0, Lng: halfPiDeg}},
		{{Lat: -halfPiDeg, Lng: 0}, {Lat: 0, Lng: halfPiDeg}, {Lat: 0, Lng: 0}},
	}

	out := make([]GeoPolygon, len(verts))
	for i := range verts {
		out[i] = GeoPolygon{GeoLoop: GeoLoop{verts[i][0], verts[i][1], verts[i][2]}}
	}

	return out
}

// CellsToMultiPolygon outlines a set of cells as GeoPolygons: each polygon has
// one counter-clockwise outer loop followed by any clockwise holes, and polygons
// are ordered by decreasing outer loop area. The cells must all be valid, share
// one resolution, and contain no duplicates.
func CellsToMultiPolygon(cells []Cell) ([]GeoPolygon, error) {
	if len(cells) == 0 {
		return nil, nil
	}

	if err := validateCellSet(cells); err != nil {
		return nil, err
	}

	arcs, index := buildArcs(cells)
	cancelArcPairs(arcs, index)

	loops := buildOutlineLoops(arcs)
	if len(loops) == 0 {
		return globeMultiPolygon(), nil
	}

	return assembleMultiPolygon(loops), nil
}
