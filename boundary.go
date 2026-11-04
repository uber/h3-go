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

type (
	// CellBoundary is the ordered set of geographic vertices that outline a cell.
	// It never has more vertices than a cell has topological vertices plus its
	// distortion vertices.
	CellBoundary []LatLng
)

// CellToBoundary returns the geographic boundary of a cell as an ordered list of
// vertices in degrees.
func CellToBoundary(c Cell) (CellBoundary, error) {
	return c.Boundary()
}

// Boundary returns the geographic boundary of the cell as an ordered list of
// vertices in degrees.
func (c Cell) Boundary() (CellBoundary, error) {
	return c.appendBoundary(make(CellBoundary, 0, MaxCellBndryVerts))
}

// appendBoundary appends the geographic boundary of the cell to out and returns
// the extended slice. Callers that consume the boundary immediately can pass a
// stack buffer to avoid a heap allocation.
func (c Cell) appendBoundary(out CellBoundary) (CellBoundary, error) {
	fijk, err := c.toFaceIjk()
	if err != nil {
		return nil, err
	}

	res := c.Resolution()
	if c.IsPentagon() {
		return fijk.pentToCellBoundary(out, res, 0, numPentVerts), nil
	}

	return fijk.toCellBoundary(out, res, 0, numHexVerts), nil
}

// boundaryVec3 appends the cell's boundary as unit vectors on the sphere to
// out, in the same order as Boundary.
func (c Cell) boundaryVec3(out []vec3d) ([]vec3d, error) {
	fijk, err := c.toFaceIjk()
	if err != nil {
		return nil, err
	}

	res := c.Resolution()
	if c.IsPentagon() {
		return fijk.pentToCellBoundaryVec3(out, res, 0, numPentVerts), nil
	}

	return fijk.toCellBoundaryVec3(out, res, 0, numHexVerts), nil
}

// centerVec3 returns the cell's center as a unit vector on the sphere.
func (c Cell) centerVec3() (vec3d, error) {
	fijk, err := c.toFaceIjk()
	if err != nil {
		return vec3d{}, err
	}

	return fijk.toVec3(c.Resolution()), nil
}

// toCellBoundary appends the geographic boundary of a hexagon cell, built from
// its FaceIJK address, to out. It walks length topological vertices starting at
// start, adjusting each onto the correct face and inserting an extra vertex
// wherever a Class III cell edge crosses an icosahedron face edge (so each half
// of the edge projects with the correct face). start and length select a
// sub-range of the loop, which directed-edge boundaries use to project a single
// edge.
func (fijk faceIJK) toCellBoundary(out CellBoundary, res, start, length int) CellBoundary {
	var buf [MaxCellBndryVerts]vec3d
	for _, point := range fijk.toCellBoundaryVec3(buf[:0], res, start, length) {
		out = append(out, point.toLatLng())
	}

	return out
}

// toCellBoundaryVec3 is toCellBoundary with the boundary left as unit vectors
// on the sphere, the form the geodesic polygon fill works in.
func (fijk faceIJK) toCellBoundaryVec3(out []vec3d, res, start, length int) []vec3d {
	adjRes, fijkVerts := fijk.toVerts(res)
	centerFace := fijk.face

	// When returning the whole loop, run one extra iteration to catch a
	// distortion vertex on the last edge.
	additionalIteration := 0
	if length == numHexVerts {
		additionalIteration = 1
	}

	lastFace := -1
	lastOverage := noOverage

	for vert := start; vert < start+length+additionalIteration; vert++ {
		v := vert % numHexVerts
		vfijk := fijkVerts[v]

		var ov overage

		vfijk, ov = vfijk.adjustOverageClassII(adjRes, false, true)

		// Class III cell edges can cross an icosahedron edge; Class II edges have
		// their vertices on the face edge with no line intersection.
		if isResClassIII(res) && vert > start && vfijk.face != lastFace && lastOverage != faceEdge {
			lastV := (v + numHexVerts - 1) % numHexVerts
			orig2d0 := fijkVerts[lastV].coord.toHex2d()
			orig2d1 := fijkVerts[v].coord.toHex2d()

			face2 := lastFace
			if lastFace == centerFace {
				face2 = vfijk.face
			}

			edge0, edge1 := icosaEdge(adjRes, adjacentFaceDir[centerFace][face2])
			inter := orig2d0.intersect(orig2d1, edge0, edge1)

			// If the intersection lands on a hexagon vertex, both adjacent edges
			// lie on a single face and no extra vertex is needed.
			if !orig2d0.almostEquals(inter) && !orig2d1.almostEquals(inter) {
				out = append(out, inter.toVec3(centerFace, adjRes, true))
			}
		}

		// The final extra iteration only tests for an edge intersection.
		if vert < start+numHexVerts {
			out = append(out, vfijk.coord.toHex2d().toVec3(vfijk.face, adjRes, true))
		}

		lastFace = vfijk.face
		lastOverage = ov
	}

	return out
}

// pentToCellBoundary appends the geographic boundary of a pentagon cell, built
// from its FaceIJK address, to out. Every Class III pentagon edge crosses an
// icosahedron edge, so a crossing vertex is inserted on each such edge. start
// and length select a sub-range of the loop, which directed-edge boundaries use
// to project a single edge.
func (fijk faceIJK) pentToCellBoundary(out CellBoundary, res, start, length int) CellBoundary {
	var buf [MaxCellBndryVerts]vec3d
	for _, point := range fijk.pentToCellBoundaryVec3(buf[:0], res, start, length) {
		out = append(out, point.toLatLng())
	}

	return out
}

// pentToCellBoundaryVec3 is pentToCellBoundary with the boundary left as unit
// vectors on the sphere.
func (fijk faceIJK) pentToCellBoundaryVec3(out []vec3d, res, start, length int) []vec3d {
	adjRes, fijkVerts := fijk.pentToVerts(res)

	additionalIteration := 0
	if length == numPentVerts {
		additionalIteration = 1
	}

	var lastFijk faceIJK

	for vert := start; vert < start+length+additionalIteration; vert++ {
		v := vert % numPentVerts
		vfijk := fijkVerts[v]
		vfijk, _ = vfijk.adjustPentVertOverage(adjRes)

		if isResClassIII(res) && vert > start {
			orig2d0 := lastFijk.coord.toHex2d()

			// Re-express the current vertex on the previous vertex's face so the
			// two 2D points share a coordinate system for the intersection.
			currentToLastDir := adjacentFaceDir[vfijk.face][lastFijk.face]
			fijkOrient := faceNeighbors[vfijk.face][currentToLastDir]

			tmpFijk := faceIJK{face: fijkOrient.face, coord: vfijk.coord}

			ijk := tmpFijk.coord
			for range fijkOrient.ccwRot60 {
				ijk = ijk.rotate60ccw()
			}

			ijk = ijk.add(fijkOrient.translate.scale(unitScaleByCIIres[adjRes] * 3)).normalize()
			tmpFijk.coord = ijk

			orig2d1 := ijk.toHex2d()

			edge0, edge1 := icosaEdge(adjRes, adjacentFaceDir[tmpFijk.face][vfijk.face])
			inter := orig2d0.intersect(orig2d1, edge0, edge1)
			out = append(out, inter.toVec3(tmpFijk.face, adjRes, true))
		}

		if vert < start+numPentVerts {
			out = append(out, vfijk.coord.toHex2d().toVec3(vfijk.face, adjRes, true))
		}

		lastFijk = vfijk
	}

	return out
}

// icosaEdge returns the two 2D substrate endpoints of the icosahedron face edge
// in the given direction (dirIJ/dirJK/dirKI) at the given Class II resolution.
// These bound the segment a crossing cell edge is intersected against.
func icosaEdge(res, dir int) (edge0, edge1 vec2d) {
	maxDim := float64(maxDimByCIIres[res])
	v0 := vec2d{x: 3.0 * maxDim, y: 0.0}
	v1 := vec2d{x: -1.5 * maxDim, y: 3.0 * mSqrt3Half * maxDim}
	v2 := vec2d{x: -1.5 * maxDim, y: -3.0 * mSqrt3Half * maxDim}

	switch dir {
	case dirIJ:
		return v0, v1
	case dirJK:
		return v1, v2
	default: // dirKI
		return v2, v0
	}
}
