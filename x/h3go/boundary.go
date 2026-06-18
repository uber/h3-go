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
	fijk, err := c.toFaceIjk()
	if err != nil {
		return nil, err
	}

	res := c.Resolution()
	if c.IsPentagon() {
		return fijk.pentToCellBoundary(res, 0, numPentVerts), nil
	}

	return fijk.toCellBoundary(res, 0, numHexVerts), nil
}

// toCellBoundary builds the geographic boundary of a hexagon cell from its
// FaceIJK address. It walks length topological vertices starting at start,
// adjusting each onto the correct face and inserting an extra vertex wherever a
// Class III cell edge crosses an icosahedron face edge (so each half of the edge
// projects with the correct face). start and length select a sub-range of the
// loop, which directed-edge boundaries use to project a single edge.
func (fijk faceIJK) toCellBoundary(res, start, length int) CellBoundary {
	adjRes, fijkVerts := fijk.toVerts(res)
	centerFace := fijk.face

	// When returning the whole loop, run one extra iteration to catch a
	// distortion vertex on the last edge.
	additionalIteration := 0
	if length == numHexVerts {
		additionalIteration = 1
	}

	var boundary CellBoundary

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
				boundary = append(boundary, inter.toVec3(centerFace, adjRes, true).toLatLng())
			}
		}

		// The final extra iteration only tests for an edge intersection.
		if vert < start+numHexVerts {
			boundary = append(boundary, vfijk.coord.toHex2d().toVec3(vfijk.face, adjRes, true).toLatLng())
		}

		lastFace = vfijk.face
		lastOverage = ov
	}

	return boundary
}

// pentToCellBoundary builds the geographic boundary of a pentagon cell from its
// FaceIJK address. Every Class III pentagon edge crosses an icosahedron edge, so
// a crossing vertex is inserted on each such edge. start and length select a
// sub-range of the loop, which directed-edge boundaries use to project a single
// edge.
func (fijk faceIJK) pentToCellBoundary(res, start, length int) CellBoundary {
	adjRes, fijkVerts := fijk.pentToVerts(res)

	additionalIteration := 0
	if length == numPentVerts {
		additionalIteration = 1
	}

	var boundary CellBoundary

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

			ijk = ijk.add(fijkOrient.translate.scale(unitScaleByCIIres[adjRes] * 3))
			ijk.normalize()
			tmpFijk.coord = ijk

			orig2d1 := ijk.toHex2d()

			edge0, edge1 := icosaEdge(adjRes, adjacentFaceDir[tmpFijk.face][vfijk.face])
			inter := orig2d0.intersect(orig2d1, edge0, edge1)
			boundary = append(boundary, inter.toVec3(tmpFijk.face, adjRes, true).toLatLng())
		}

		if vert < start+numPentVerts {
			boundary = append(boundary, vfijk.coord.toHex2d().toVec3(vfijk.face, adjRes, true).toLatLng())
		}

		lastFijk = vfijk
	}

	return boundary
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
