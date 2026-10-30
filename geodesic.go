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

package h3

import (
	"iter"
	"math"
)

// The geodesic polygon fill treats polygon edges as great-circle arcs rather
// than straight lines in latitude and longitude. All of its geometry runs on
// the unit sphere in Cartesian coordinates: a great circle is the intersection
// of the sphere with a plane through its center, so which side of an edge a
// point lies on is the sign of one dot product with the plane normal, and
// where two edges cross is the cross product of their normals. The chart
// singularities at the poles and the antimeridian do not exist in this form.

const (
	// geodesicSideEps classifies arc endpoints as on the same side of a
	// great-circle plane. For coincident arcs (a cell edge lying on a polygon
	// edge) the triple products are mathematically zero but reach about 2e-32
	// in floating point, so epsilon squared (1e-32) sits at the noise floor.
	// This value is four orders of magnitude above it and far below the
	// smallest genuine same-side product seen in practice (about 1e-18), so
	// coincident arcs fall through to the collinear-overlap check while
	// non-coincident same-side pairs still exit early.
	geodesicSideEps = 1e-28

	// geodesicCapEps loosens the sphere-cap tests, which compare cosines and
	// need more slack than the vector tests.
	geodesicCapEps = 1e-8

	// geodesicHemisphereEps is how far below the hemisphere plane a vertex may
	// sit and still count as inside the hemisphere.
	geodesicHemisphereEps = -1e-12

	// geodesicPoleEps is how far behind the hemisphere pole a query point may
	// sit before it is rejected outright.
	geodesicPoleEps = -1e-10

	// geodesicCentroidMinNormSq is the squared norm below which the vertex
	// centroid is too short to serve as a hemisphere pole candidate.
	geodesicCentroidMinNormSq = 1e-12

	// geodesicGreatCircleEps is the largest distance from a common plane at
	// which every vertex still counts as lying on one great circle.
	geodesicGreatCircleEps = 1e-14

	// geodesicMinLoopVerts is the fewest vertices a loop may have.
	geodesicMinLoopVerts = 3

	// geodesicGreatCircleMinVerts is the fewest vertices at which an
	// all-on-one-great-circle loop is rejected; smaller loops keep the legacy
	// point, line and triangle handling.
	geodesicGreatCircleMinVerts = 4

	// sphereCapScaleFactor makes a cap of radius maxEdgeLengthRads[res] *
	// sphereCapScaleFactor enclose the cell and every descendant. Children stay
	// inside their parent's edge length, but a grandchild's boundary reaches
	// 3.5% beyond it and deeper descendants about 4.5%, so the reference
	// library's factor of 1.01 lets the coarse-cell prune drop a cell whose
	// boundary crosses a small polygon at a descendant's outlying vertex.
	sphereCapScaleFactor = 1.05
)

// capCosRadius is cos(maxEdgeLengthRads[res] * sphereCapScaleFactor) for each
// resolution, the cosine of the bounding cap's angular radius.
var capCosRadius = func() (table [MaxResolution + 1]float64) {
	for res := range table {
		table[res] = math.Cos(maxEdgeLengthRads[res] * sphereCapScaleFactor)
	}

	return table
}()

// unitAxes are the Cartesian axes, probed when bounding arcs and loops.
var unitAxes = [...]vec3d{{x: 1}, {y: 1}, {z: 1}}

// cardinalPoints are the six points where an axis meets the sphere.
var cardinalPoints = [...]vec3d{{x: 1}, {x: -1}, {y: 1}, {y: -1}, {z: 1}, {z: -1}}

type (
	// aabb is an axis-aligned bounding box in Cartesian space on the unit
	// sphere.
	aabb struct {
		min vec3d
		max vec3d
	}

	// sphereCap bounds a region of the sphere by a center vector and the
	// cosine of its angular radius.
	sphereCap struct {
		center    vec3d
		cosRadius float64
	}

	// geodesicEdge caches a polygon vertex and the great-circle edge leaving it.
	geodesicEdge struct {
		vert      vec3d   // vertex on the unit sphere
		edgeCross vec3d   // normal of the great circle through this edge
		edgeDot   float64 // dot product of the edge's two vertices
		box       aabb    // bounding box of the edge arc
	}

	// geodesicLoop is a polygon loop on the unit sphere.
	geodesicLoop struct {
		edges             []geodesicEdge
		centroid          vec3d // normalized vertex centroid
		hemispherePole    vec3d // pole of a hemisphere holding every vertex
		hasHemispherePole bool
	}

	// geodesicPolygon is the acceleration structure the geodesic fill queries.
	geodesicPolygon struct {
		holes     []geodesicLoop
		geoloop   geodesicLoop
		box       aabb  // bounding box of the whole outer loop
		capCenter vec3d // center of a cap holding the whole outer loop
		// capCos[res] is the cosine of the outer loop's cap radius plus a
		// cell cap radius at res: a cell whose center is further from
		// capCenter than that cannot touch the polygon.
		capCos [MaxResolution + 1]float64
	}
)

// --- Bounding boxes and caps ---

// emptyAABB returns the inverted empty box, min above max on every axis, so
// that the first update sets both corners.
func emptyAABB() aabb {
	return aabb{min: vec3d{x: 1, y: 1, z: 1}, max: vec3d{x: -1, y: -1, z: -1}}
}

// update grows the box to include the point.
func (b *aabb) update(v vec3d) {
	b.min = vec3d{x: math.Min(b.min.x, v.x), y: math.Min(b.min.y, v.y), z: math.Min(b.min.z, v.z)}
	b.max = vec3d{x: math.Max(b.max.x, v.x), y: math.Max(b.max.y, v.y), z: math.Max(b.max.z, v.z)}
}

// normSq returns the squared length of the vector.
func (v vec3d) normSq() float64 {
	return v.dot(v)
}

// neg returns the opposite vector.
func (v vec3d) neg() vec3d {
	return vec3d{x: -v.x, y: -v.y, z: -v.z}
}

// add returns the vector sum.
func (v vec3d) add(b vec3d) vec3d {
	return vec3d{x: v.x + b.x, y: v.y + b.y, z: v.z + b.z}
}

// normalized returns the unit vector in the direction of v.
func (v vec3d) normalized() vec3d {
	v.normalize()

	return v
}

// isOnArc reports whether the point p on the great circle with normal n lies
// on the arc from v1 to v2: the turns from v1 to p and from p to v2 both go
// the way of the normal.
func isOnArc(p, v1, v2, n vec3d) bool {
	return v1.cross(p).dot(n) >= -epsilon && p.cross(v2).dot(n) >= -epsilon
}

// updateWithArcExtrema grows a box that already holds the arc's endpoints to
// cover the arc from v1 to v2 along the great circle with normal n. Each
// coordinate is extremized on the circle where the axis projects onto the
// circle's plane, so those points and their opposites are tested for lying on
// the arc.
func (b *aabb) updateWithArcExtrema(v1, v2, n vec3d) {
	// A near-zero normal is a degenerate edge whose endpoints bound it already.
	if n.normSq() < epsilon*epsilon {
		return
	}

	normal := n.normalized()

	for _, axis := range unitAxes {
		extremum := normal.cross(axis.cross(normal))

		// An axis nearly aligned with the normal has no extremum on the circle.
		if extremum.normSq() < epsilon*epsilon {
			continue
		}

		extremum = extremum.normalized()
		if isOnArc(extremum, v1, v2, normal) {
			b.update(extremum)
		}

		if opposite := extremum.neg(); isOnArc(opposite, v1, v2, normal) {
			b.update(opposite)
		}
	}
}

// overlapsAABB reports whether the cap may intersect the box. It is false only
// when an intersection is impossible: either the box corner nearest the cap's
// center is outside the cap, or the box lies entirely outside the unit sphere.
func (c sphereCap) overlapsAABB(b aabb) bool {
	far := vec3d{
		x: pickCorner(c.center.x, b.min.x, b.max.x),
		y: pickCorner(c.center.y, b.min.y, b.max.y),
		z: pickCorner(c.center.z, b.min.z, b.max.z),
	}
	if far.dot(c.center) < c.cosRadius-geodesicCapEps {
		return false
	}

	nearOrigin := vec3d{
		x: math.Max(b.min.x, math.Min(b.max.x, 0)),
		y: math.Max(b.min.y, math.Min(b.max.y, 0)),
		z: math.Max(b.min.z, math.Min(b.max.z, 0)),
	}

	return nearOrigin.normSq() <= 1+geodesicCapEps
}

// pickCorner returns the box bound on the side the center coordinate points to.
func pickCorner(center, lo, hi float64) float64 {
	if center >= 0 {
		return hi
	}

	return lo
}

// sphereCap returns the cap that encloses the cell and all of its descendants.
func (c Cell) sphereCap() (sphereCap, error) {
	center, err := c.centerVec3()
	if err != nil {
		return sphereCap{}, err
	}

	return sphereCap{center: center, cosRadius: capCosRadius[c.Resolution()]}, nil
}

// --- Predicates ---

// geodesicEdgesCross reports whether the arc a1 to a2 and the arc b1 to b2
// intersect on the unit sphere, with normalB the normal of b's great circle.
// It culls pairs whose endpoints lie on the same side of the other's plane,
// handles near-parallel arcs by a one-dimensional projection, and otherwise
// checks that the intersection of the two great circles lies on both arcs.
func geodesicEdgesCross(a1, a2, b1, b2, normalB vec3d) bool {
	normalA := a1.cross(a2)

	b1Side, b2Side := normalA.dot(b1), normalA.dot(b2)
	a1Side, a2Side := normalB.dot(a1), normalB.dot(a2)

	if b1Side*b2Side > geodesicSideEps || a1Side*a2Side > geodesicSideEps {
		return false
	}

	line := normalA.cross(normalB)

	if line.normSq() < epsilon*epsilon {
		// The arcs lie on one great circle: overlap in one dimension along it.
		ref := a2.linComb(1, -1, a1)
		if ref.normSq() < epsilon*epsilon {
			return false
		}

		projA1, projA2 := orderPair(a1.dot(ref), a2.dot(ref))
		projB1, projB2 := orderPair(b1.dot(ref), b2.dot(ref))

		return projA1 <= projB2 && projB1 <= projA2
	}

	midA, midB := a1.add(a2), b1.add(b2)

	// Antipodal endpoints leave the arc's midpoint undefined; count a crossing.
	if midA.normSq() < epsilon*epsilon || midB.normSq() < epsilon*epsilon {
		return true
	}

	return line.dot(midA)*line.dot(midB) >= -epsilon
}

// orderPair returns the two values in ascending order.
func orderPair(a, b float64) (lo, hi float64) {
	if a > b {
		return b, a
	}

	return a, b
}

// containsPoint reports whether the point lies inside the loop by spherical
// winding: the signed angles subtended by consecutive vertices, seen from the
// point, sum to about ±2π inside and about 0 outside. A point behind the
// loop's hemisphere pole is rejected first, since the winding sum is spurious
// for points on the far side of the sphere.
func (l *geodesicLoop) containsPoint(point vec3d) bool {
	if l.hasHemispherePole && l.hemispherePole.dot(point) < geodesicPoleEps {
		return false
	}

	total := 0.0
	dotPrev := point.dot(l.edges[0].vert)

	for i := range l.edges {
		dotNext := point.dot(l.edges[(i+1)%len(l.edges)].vert)

		y := point.dot(l.edges[i].edgeCross)
		x := l.edges[i].edgeDot - dotPrev*dotNext

		total += math.Atan2(y, x)
		dotPrev = dotNext
	}

	return math.Abs(total) > math.Pi
}

// toAABB returns the bounding box of the loop: its vertices, the extrema of
// its arcs, and any cardinal point of the sphere the loop encloses.
func (l *geodesicLoop) toAABB() aabb {
	box := emptyAABB()

	for i := range l.edges {
		box.update(l.edges[i].vert)
		box.updateWithArcExtrema(l.edges[i].vert, l.edges[(i+1)%len(l.edges)].vert, l.edges[i].edgeCross)
	}

	for _, point := range cardinalPoints {
		if l.containsPoint(point) {
			box.update(point)
		}
	}

	return box
}

// candidateContainsVerts reports whether every vertex lies in the closed
// hemisphere about the candidate pole.
func candidateContainsVerts(candidate vec3d, edges []geodesicEdge) bool {
	if candidate.normSq() < epsilon*epsilon {
		return false
	}

	pole := candidate.normalized()
	for i := range edges {
		if pole.dot(edges[i].vert) < geodesicHemisphereEps {
			return false
		}
	}

	return true
}

// hemispherePole returns a pole whose closed hemisphere holds every vertex, if
// one exists. It tries the vertex centroid, then each vertex, then the cross
// product of each vertex pair and its opposite. The centroid succeeds in the
// common case; the pairwise search is the exhaustive fallback.
func hemispherePole(edges []geodesicEdge) (vec3d, bool) {
	var centroid vec3d
	for i := range edges {
		centroid = centroid.add(edges[i].vert)
	}

	// A near-zero centroid has no reliable direction.
	if centroid.normSq() >= geodesicCentroidMinNormSq && candidateContainsVerts(centroid, edges) {
		return centroid.normalized(), true
	}

	for i := range edges {
		if candidateContainsVerts(edges[i].vert, edges) {
			return edges[i].vert, true
		}
	}

	for i := range edges {
		for j := i + 1; j < len(edges); j++ {
			candidate := edges[i].vert.cross(edges[j].vert)
			if candidateContainsVerts(candidate, edges) {
				return candidate.normalized(), true
			}

			if opposite := candidate.neg(); candidateContainsVerts(opposite, edges) {
				return opposite.normalized(), true
			}
		}
	}

	return vec3d{}, false
}

// verticesOnGreatCircle reports whether every vertex lies on one great circle,
// a loop with no enclosed area and ambiguous containment. Loops of fewer than
// four vertices keep the legacy point, line and triangle handling.
func verticesOnGreatCircle(edges []geodesicEdge) bool {
	if len(edges) < geodesicGreatCircleMinVerts {
		return false
	}

	var normal vec3d
	found := false

	for i := 0; i < len(edges) && !found; i++ {
		for j := i + 1; j < len(edges); j++ {
			normal = edges[i].vert.cross(edges[j].vert)
			if normal.normSq() > geodesicCentroidMinNormSq {
				found = true

				break
			}
		}
	}

	if !found {
		return false
	}

	normal = normal.normalized()
	for i := range edges {
		if math.Abs(normal.dot(edges[i].vert)) > geodesicGreatCircleEps {
			return false
		}
	}

	return true
}

// --- Construction ---

// newGeodesicLoop converts a loop in degrees into geodesic form. A loop whose
// vertices all lie on one great circle is rejected, and when rejectLarge is set
// so is a loop that does not fit in a hemisphere, which the containment test
// cannot resolve. Every rejection is ErrDomain.
func newGeodesicLoop(loop GeoLoop, rejectLarge bool) (geodesicLoop, error) {
	if len(loop) < geodesicMinLoopVerts {
		return geodesicLoop{}, ErrDomain
	}

	edges := make([]geodesicEdge, len(loop))
	for i, vert := range loop {
		if math.IsNaN(vert.Lat) || math.IsInf(vert.Lat, 0) || math.IsNaN(vert.Lng) || math.IsInf(vert.Lng, 0) {
			return geodesicLoop{}, ErrDomain
		}

		edges[i].vert = latLngToVec3(vert.Lat*DegsToRads, vert.Lng*DegsToRads)
	}

	if verticesOnGreatCircle(edges) {
		return geodesicLoop{}, ErrDomain
	}

	out := geodesicLoop{edges: edges}

	if rejectLarge {
		pole, ok := hemispherePole(edges)
		if !ok {
			return geodesicLoop{}, ErrDomain
		}

		out.hemispherePole, out.hasHemispherePole = pole, true
	}

	for i := range edges {
		v1, v2 := edges[i].vert, edges[(i+1)%len(edges)].vert

		out.centroid = out.centroid.add(v1)

		edges[i].edgeCross = v1.cross(v2)
		edges[i].edgeDot = v1.dot(v2)

		box := emptyAABB()
		box.update(v1)
		box.update(v2)
		box.updateWithArcExtrema(v1, v2, edges[i].edgeCross)
		edges[i].box = box
	}

	out.centroid = out.centroid.normalized()

	return out, nil
}

// newGeodesicPolygon converts a polygon in degrees into the geodesic
// acceleration structure. The outer loop must fit in a hemisphere; holes need
// not, but every loop needs at least three finite vertices.
func newGeodesicPolygon(polygon GeoPolygon) (*geodesicPolygon, error) {
	outer, err := newGeodesicLoop(polygon.GeoLoop, true)
	if err != nil {
		return nil, err
	}

	out := &geodesicPolygon{geoloop: outer}

	if len(polygon.Holes) > 0 {
		out.holes = make([]geodesicLoop, len(polygon.Holes))
		for i, hole := range polygon.Holes {
			out.holes[i], err = newGeodesicLoop(hole, false)
			if err != nil {
				return nil, err
			}
		}
	}

	out.box = out.geoloop.toAABB()

	var capRadius float64
	out.capCenter, capRadius = out.geoloop.boundingCap()

	for res := range out.capCos {
		out.capCos[res] = math.Cos(math.Min(math.Pi, capRadius+maxEdgeLengthRads[res]*sphereCapScaleFactor))
	}

	return out, nil
}

// boundingCap returns the loop's hemisphere pole and the angular radius about
// it of the smallest cap holding the loop. The farthest point is a vertex:
// along an arc, the angle from the pole peaks only at the point opposite the
// pole's projection onto the arc's plane, and an arc through that point has
// both endpoints past a quarter turn from the pole, which the hemisphere test
// rules out.
func (l *geodesicLoop) boundingCap() (vec3d, float64) {
	minDot := 1.0
	for i := range l.edges {
		minDot = math.Min(minDot, l.hemispherePole.dot(l.edges[i].vert))
	}

	return l.hemispherePole, math.Acos(math.Max(-1, math.Min(1, minDot)))
}

// --- Queries ---

// capIntersects reports whether a cell's bounding cap may touch the polygon's
// outer loop.
func (p *geodesicPolygon) capIntersects(region sphereCap, res int) bool {
	return region.center.dot(p.capCenter) >= p.capCos[res]-geodesicCapEps && region.overlapsAABB(p.box)
}

// boundaryIntersects reports whether a cell boundary crosses any loop of the
// polygon. Arcs whose boxes miss the cell's cap are skipped; the rest are
// tested pairwise against the boundary's edges.
func (p *geodesicPolygon) boundaryIntersects(boundary []vec3d, region sphereCap) bool {
	if p.geoloop.boundaryIntersects(boundary, region) {
		return true
	}

	for i := range p.holes {
		if p.holes[i].boundaryIntersects(boundary, region) {
			return true
		}
	}

	return false
}

// boundaryIntersects reports whether a cell boundary crosses the loop.
func (l *geodesicLoop) boundaryIntersects(boundary []vec3d, region sphereCap) bool {
	for i := range l.edges {
		if !region.overlapsAABB(l.edges[i].box) {
			continue
		}

		edge := &l.edges[i]
		next := l.edges[(i+1)%len(l.edges)].vert

		for j := range boundary {
			if geodesicEdgesCross(boundary[j], boundary[(j+1)%len(boundary)], edge.vert, next, edge.edgeCross) {
				return true
			}
		}
	}

	return false
}

// containsPoint reports whether the point lies inside the outer loop and
// outside every hole.
func (p *geodesicPolygon) containsPoint(point vec3d) bool {
	if !p.geoloop.containsPoint(point) {
		return false
	}

	for i := range p.holes {
		if p.holes[i].containsPoint(point) {
			return false
		}
	}

	return true
}

// vertexCell returns the cell at res containing the loop's first vertex.
func (l *geodesicLoop) vertexCell(res int) Cell {
	cell, _ := l.edges[0].vert.toFaceIjk(res).toH3(res)

	return cell
}

// --- Traversal ---

// cells yields every cell at res the polygon includes under the mode, which
// must be ContainmentCenter, ContainmentFull or ContainmentOverlapping. It
// walks the global hierarchy depth first, pruning coarse cells whose bounding
// cap misses the polygon and classifying cells at the target resolution by
// their center first and their boundary only when the center leaves the answer
// open.
func (p *geodesicPolygon) cells(res int, mode ContainmentMode) iter.Seq[Cell] {
	return func(yield func(Cell) bool) {
		cell := baseCellNumToCell(0)
		for cell != 0 {
			cellRes := cell.Resolution()

			if cellRes < res {
				// A coarse cell is a candidate only if its cap can touch the polygon.
				region, _ := cell.sphereCap()
				if !p.capIntersects(region, cellRes) {
					cell = nextCell(cell)

					continue
				}

				cell, _ = cell.CenterChild(cellRes + 1)

				continue
			}

			if p.includes(cell, mode) && !yield(cell) {
				return
			}

			cell = nextCell(cell)
		}
	}
}

// includes reports whether a cell at the target resolution belongs to the fill.
func (p *geodesicPolygon) includes(cell Cell, mode ContainmentMode) bool {
	center, _ := cell.centerVec3()
	centerInside := p.containsPoint(center)

	switch {
	case mode == ContainmentCenter:
		return centerInside
	case mode == ContainmentOverlapping && centerInside:
		return true
	case mode == ContainmentFull && !centerInside:
		return false
	}

	// The center alone did not decide, so look at the boundary.
	var buf [MaxCellBndryVerts]vec3d
	boundary, _ := cell.boundaryVec3(buf[:0])
	region, _ := cell.sphereCap()
	crosses := p.boundaryIntersects(boundary, region)

	if mode == ContainmentFull {
		return p.fullyContains(cell, boundary, crosses)
	}

	// Overlapping with the center outside: a crossing settles it; otherwise the
	// polygon, or a hole, may sit entirely inside the cell, which its first
	// vertex reveals.
	return crosses || p.anyLoopVertexIn(cell)
}

// fullyContains reports whether a cell whose center is inside the polygon is
// entirely inside it: its boundary crosses no loop, a boundary vertex is inside
// the filled polygon, and no hole lies inside the cell.
func (p *geodesicPolygon) fullyContains(cell Cell, boundary []vec3d, crosses bool) bool {
	if crosses || !p.containsPoint(boundary[0]) {
		return false
	}

	res := cell.Resolution()
	for i := range p.holes {
		if p.holes[i].vertexCell(res) == cell {
			return false
		}
	}

	return true
}

// anyLoopVertexIn reports whether the first vertex of the outer loop or of any
// hole falls in the cell. An outer vertex means the polygon is smaller than the
// cell; a hole vertex means the hole is inside the cell, so the filled region
// around it still overlaps the cell.
func (p *geodesicPolygon) anyLoopVertexIn(cell Cell) bool {
	res := cell.Resolution()
	if p.geoloop.vertexCell(res) == cell {
		return true
	}

	for i := range p.holes {
		if p.holes[i].vertexCell(res) == cell {
			return true
		}
	}

	return false
}
