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

// This file checks the float64 projection from a cell's face coordinates to a
// point on the unit sphere against the same formula evaluated in 256-bit
// arithmetic. Agreement with another implementation cannot tell accuracy from
// shared error; this test can, because the whole chain from integer IJK
// coordinates to a unit vector is rational arithmetic plus square roots, plus
// the sine and cosine of twenty fixed face-axis azimuths, all of which
// math/big evaluates exactly for this purpose. The chain is compared where it
// ends, in unit-vector space, by chord length, which equals the angular
// separation for separations this small and avoids the asin and atan2 of the
// final conversion to degrees.

import (
	"math/big"
	"os"
	"strconv"
	"testing"
)

const (
	// exactPrec is the working precision of the reference evaluation, in bits.
	// It is nearly five times float64's 53, so the reference's own rounding is
	// irrelevant at the bound below.
	exactPrec = 256

	// exactProjectionBoundDeg is the largest angular separation, in degrees of
	// arc, allowed between the float64 result and the exact one. One unit in
	// the last place of a unit-vector component is about 1.3e-14 degrees, so
	// the bound is roughly eight ulps; the largest separation observed over
	// every cell to res 4 and samples to res 15 is 3.9e-14 degrees, about
	// three ulps. The conformance suite's coordinate tolerance is a hundred
	// times looser.
	exactProjectionBoundDeg = 1e-13

	// exactProjectionDefaultMaxRes is the finest resolution enumerated in full
	// by default. H3GO_EXACT_PROJECTION_MAXRES raises it; every resolution is
	// also sampled regardless.
	exactProjectionDefaultMaxRes = 2

	// exactSeriesCutoff ends the sine and cosine series once a term is below
	// this, far under the precision in use.
	exactSeriesCutoff = 1e-90
)

// bigVec3 is a 3D vector with big.Float components.
type bigVec3 struct {
	x, y, z *big.Float
}

// bigOf returns a big.Float at the working precision holding value exactly.
func bigOf(value float64) *big.Float {
	return new(big.Float).SetPrec(exactPrec).SetFloat64(value)
}

// bigInt returns a big.Float at the working precision holding value exactly.
func bigInt(value int64) *big.Float {
	return new(big.Float).SetPrec(exactPrec).SetInt64(value)
}

// bigAdd returns a + b.
func bigAdd(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(exactPrec).Add(a, b)
}

// bigSub returns a - b.
func bigSub(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(exactPrec).Sub(a, b)
}

// bigMul returns a * b.
func bigMul(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(exactPrec).Mul(a, b)
}

// bigQuo returns a / b.
func bigQuo(a, b *big.Float) *big.Float {
	return new(big.Float).SetPrec(exactPrec).Quo(a, b)
}

// bigSqrt returns the square root of a.
func bigSqrt(a *big.Float) *big.Float {
	return new(big.Float).SetPrec(exactPrec).Sqrt(a)
}

// bigSinCos returns the sine and cosine of angle by Taylor series. The angles
// used here are below 2*pi, where the series converges with no range reduction
// and the working precision absorbs the cancellation among the early terms.
func bigSinCos(angle *big.Float) (sin, cos *big.Float) {
	sin = bigInt(0)
	cos = bigInt(1)
	term := bigInt(1)
	cutoff := bigOf(exactSeriesCutoff)

	for degree := int64(1); ; degree++ {
		term = bigQuo(bigMul(term, angle), bigInt(degree))

		if new(big.Float).Abs(term).Cmp(cutoff) < 0 {
			return sin, cos
		}

		// Terms cycle through +sin, -cos, -sin, +cos as the degree advances.
		switch degree % 4 {
		case 1:
			sin = bigAdd(sin, term)
		case 2:
			cos = bigSub(cos, term)
		case 3:
			sin = bigSub(sin, term)
		default:
			cos = bigAdd(cos, term)
		}
	}
}

// dot returns the dot product of two vectors.
func (v bigVec3) dot(other bigVec3) *big.Float {
	return bigAdd(bigAdd(bigMul(v.x, other.x), bigMul(v.y, other.y)), bigMul(v.z, other.z))
}

// cross returns the cross product v x other.
func (v bigVec3) cross(other bigVec3) bigVec3 {
	return bigVec3{
		x: bigSub(bigMul(v.y, other.z), bigMul(v.z, other.y)),
		y: bigSub(bigMul(v.z, other.x), bigMul(v.x, other.z)),
		z: bigSub(bigMul(v.x, other.y), bigMul(v.y, other.x)),
	}
}

// linComb returns scaleA*v + scaleB*other.
func (v bigVec3) linComb(scaleA, scaleB *big.Float, other bigVec3) bigVec3 {
	return bigVec3{
		x: bigAdd(bigMul(scaleA, v.x), bigMul(scaleB, other.x)),
		y: bigAdd(bigMul(scaleA, v.y), bigMul(scaleB, other.y)),
		z: bigAdd(bigMul(scaleA, v.z), bigMul(scaleB, other.z)),
	}
}

// normalized returns v scaled to unit length.
func (v bigVec3) normalized() bigVec3 {
	scale := bigQuo(bigInt(1), bigSqrt(v.dot(v)))

	return bigVec3{x: bigMul(v.x, scale), y: bigMul(v.y, scale), z: bigMul(v.z, scale)}
}

// exactOracle evaluates the projection in vec2d.toVec3 at the working
// precision. Its constants are the definitions the float64 code approximates:
// the face center vectors and axis azimuths are the package's literals, the
// gnomonic scale is the package's literal, and the irrational factors are
// computed exactly rather than taken from the rounded float64 constants.
type exactOracle struct {
	center, north, east [NumIcosaFaces]bigVec3
	axisCos, axisSin    [NumIcosaFaces]*big.Float
	sqrt3Half, rSqrt7   *big.Float
	oneThird, gnomonic  *big.Float
	rotCos, rotSin      *big.Float
}

// newExactOracle builds the oracle's constants.
func newExactOracle() *exactOracle {
	oracle := &exactOracle{
		sqrt3Half: bigQuo(bigSqrt(bigInt(3)), bigInt(2)),
		rSqrt7:    bigQuo(bigInt(1), bigSqrt(bigInt(7))),
		oneThird:  bigQuo(bigInt(1), bigInt(3)),
		gnomonic:  bigOf(res0UGnomonic),
		// The Class III rotation is asin(sqrt(3/28)), so its sine and cosine
		// are the square roots of 3/28 and 25/28.
		rotSin: bigSqrt(bigQuo(bigInt(3), bigInt(28))),
		rotCos: bigSqrt(bigQuo(bigInt(25), bigInt(28))),
	}

	northPole := bigVec3{x: bigInt(0), y: bigInt(0), z: bigInt(1)}

	for face := range NumIcosaFaces {
		point := faceCenterPoint[face]
		center := bigVec3{x: bigOf(point.x), y: bigOf(point.y), z: bigOf(point.z)}
		oracle.center[face] = center

		north := northPole.linComb(bigInt(1), bigSub(bigInt(0), northPole.dot(center)), center).normalized()
		oracle.north[face] = north
		oracle.east[face] = north.cross(center)

		oracle.axisSin[face], oracle.axisCos[face] = bigSinCos(bigOf(faceAxesAzRadsCII[face][0]))
	}

	return oracle
}

// hex2d is coordIJK.toHex2d at the working precision.
func (o *exactOracle) hex2d(ijk coordIJK) (x, y *big.Float) {
	iAxis := bigInt(int64(ijk.i - ijk.k))
	jAxis := bigInt(int64(ijk.j - ijk.k))

	x = bigSub(iAxis, bigQuo(jAxis, bigInt(2)))
	y = bigMul(jAxis, o.sqrt3Half)

	return x, y
}

// toVec3 is vec2d.toVec3 at the working precision, step for step.
func (o *exactOracle) toVec3(x, y *big.Float, face, res int, substrate bool) bigVec3 {
	mag := bigSqrt(bigAdd(bigMul(x, x), bigMul(y, y)))
	if mag.Sign() == 0 {
		return o.center[face]
	}

	cosTheta := bigQuo(x, mag)
	sinTheta := bigQuo(y, mag)

	radius := mag
	for range res {
		radius = bigMul(radius, o.rSqrt7)
	}

	if substrate {
		radius = bigMul(radius, o.oneThird)
		if isResClassIII(res) {
			radius = bigMul(radius, o.rSqrt7)
		}
	}

	radius = bigMul(radius, o.gnomonic)

	if !substrate && isResClassIII(res) {
		cosTheta, sinTheta = bigSub(bigMul(cosTheta, o.rotCos), bigMul(sinTheta, o.rotSin)),
			bigAdd(bigMul(sinTheta, o.rotCos), bigMul(cosTheta, o.rotSin))
	}

	cosAz := bigAdd(bigMul(o.axisCos[face], cosTheta), bigMul(o.axisSin[face], sinTheta))
	sinAz := bigSub(bigMul(o.axisSin[face], cosTheta), bigMul(o.axisCos[face], sinTheta))

	invHyp := bigQuo(bigInt(1), bigSqrt(bigAdd(bigInt(1), bigMul(radius, radius))))
	sinR := bigMul(radius, invHyp)

	dir := o.north[face].linComb(cosAz, sinAz, o.east[face])

	return o.center[face].linComb(invHyp, sinR, dir).normalized()
}

// icosaEdge is boundary.go's icosaEdge at the working precision.
func (o *exactOracle) icosaEdge(res, dir int) (edge0, edge1 [2]*big.Float) {
	maxDim := bigInt(int64(maxDimByCIIres[res]))
	halfSpan := bigMul(bigOf(-1.5), maxDim)
	height := bigMul(bigMul(bigInt(3), o.sqrt3Half), maxDim)

	vertex0 := [2]*big.Float{bigMul(bigInt(3), maxDim), bigInt(0)}
	vertex1 := [2]*big.Float{halfSpan, height}
	vertex2 := [2]*big.Float{halfSpan, bigSub(bigInt(0), height)}

	switch dir {
	case dirIJ:
		return vertex0, vertex1
	case dirJK:
		return vertex1, vertex2
	default:
		return vertex2, vertex0
	}
}

// intersect is vec2d.intersect at the working precision: the intersection of
// the line through from and to with the line through edge0 and edge1.
func (o *exactOracle) intersect(from, to, edge0, edge1 [2]*big.Float) [2]*big.Float {
	s1 := [2]*big.Float{bigSub(to[0], from[0]), bigSub(to[1], from[1])}
	s2 := [2]*big.Float{bigSub(edge1[0], edge0[0]), bigSub(edge1[1], edge0[1])}

	numerator := bigSub(
		bigMul(s2[0], bigSub(from[1], edge0[1])),
		bigMul(s2[1], bigSub(from[0], edge0[0])),
	)
	denominator := bigAdd(bigSub(bigInt(0), bigMul(s2[0], s1[1])), bigMul(s1[0], s2[1]))
	scale := bigQuo(numerator, denominator)

	return [2]*big.Float{bigAdd(from[0], bigMul(scale, s1[0])), bigAdd(from[1], bigMul(scale, s1[1]))}
}

// separationDeg returns the angular separation between a float64 unit vector
// and an exact one, in degrees of arc. The chord length is used directly; at
// these magnitudes it equals the angle to far more digits than are reported.
func separationDeg(got vec3d, want bigVec3) float64 {
	diff := bigVec3{
		x: bigSub(bigOf(got.x), want.x),
		y: bigSub(bigOf(got.y), want.y),
		z: bigSub(bigOf(got.z), want.z),
	}

	chord, _ := bigSqrt(diff.dot(diff)).Float64()

	return chord * RadsToDegs
}

// projectionPoint is one evaluation of the projection: a 2D point on a face in
// both representations, with the resolution and substrate flag it projects
// under.
type projectionPoint struct {
	face, res int
	substrate bool
	float     vec2d
	exact     [2]*big.Float
}

// boundaryPoints returns every projection the cell's boundary performs, in the
// order boundary.go performs them, mirroring toCellBoundary and
// pentToCellBoundary so that the inserted face-crossing vertices are covered
// as well as the topological ones.
func boundaryPoints(oracle *exactOracle, cell Cell, fijk faceIJK) []projectionPoint {
	if cell.IsPentagon() {
		return pentagonBoundaryPoints(oracle, fijk, cell.Resolution())
	}

	return hexagonBoundaryPoints(oracle, fijk, cell.Resolution())
}

// vertexPoint builds the projection of a substrate vertex.
func vertexPoint(oracle *exactOracle, vfijk faceIJK, adjRes int) projectionPoint {
	exactX, exactY := oracle.hex2d(vfijk.coord)

	return projectionPoint{
		face: vfijk.face, res: adjRes, substrate: true,
		float: vfijk.coord.toHex2d(), exact: [2]*big.Float{exactX, exactY},
	}
}

// crossingPoint builds the projection of the point where the cell edge from
// one substrate vertex to the next crosses an icosahedron face edge.
func crossingPoint(oracle *exactOracle, from, to coordIJK, face, adjRes, dir int) projectionPoint {
	edge0, edge1 := icosaEdge(adjRes, dir)
	exactEdge0, exactEdge1 := oracle.icosaEdge(adjRes, dir)

	fromX, fromY := oracle.hex2d(from)
	toX, toY := oracle.hex2d(to)

	return projectionPoint{
		face: face, res: adjRes, substrate: true,
		float: from.toHex2d().intersect(to.toHex2d(), edge0, edge1),
		exact: oracle.intersect([2]*big.Float{fromX, fromY}, [2]*big.Float{toX, toY}, exactEdge0, exactEdge1),
	}
}

// hexagonBoundaryPoints mirrors faceIJK.toCellBoundary for a whole loop.
func hexagonBoundaryPoints(oracle *exactOracle, fijk faceIJK, res int) []projectionPoint {
	adjRes, verts := fijk.toVerts(res)
	centerFace := fijk.face
	points := make([]projectionPoint, 0, maxCellBoundaryVerts)

	lastFace := -1
	lastOverage := noOverage

	for vert := range numHexVerts + 1 {
		index := vert % numHexVerts
		vfijk, ov := verts[index].adjustOverageClassII(adjRes, false, true)

		if isResClassIII(res) && vert > 0 && vfijk.face != lastFace && lastOverage != faceEdge {
			lastIndex := (index + numHexVerts - 1) % numHexVerts

			face2 := lastFace
			if lastFace == centerFace {
				face2 = vfijk.face
			}

			crossing := crossingPoint(oracle, verts[lastIndex].coord, verts[index].coord,
				centerFace, adjRes, adjacentFaceDir[centerFace][face2])

			from := verts[lastIndex].coord.toHex2d()
			to := verts[index].coord.toHex2d()

			if !from.almostEquals(crossing.float) && !to.almostEquals(crossing.float) {
				points = append(points, crossing)
			}
		}

		if vert < numHexVerts {
			points = append(points, vertexPoint(oracle, vfijk, adjRes))
		}

		lastFace = vfijk.face
		lastOverage = ov
	}

	return points
}

// pentagonBoundaryPoints mirrors faceIJK.pentToCellBoundary for a whole loop.
func pentagonBoundaryPoints(oracle *exactOracle, fijk faceIJK, res int) []projectionPoint {
	adjRes, verts := fijk.pentToVerts(res)
	points := make([]projectionPoint, 0, maxCellBoundaryVerts)

	var lastFijk faceIJK

	for vert := range numPentVerts + 1 {
		index := vert % numPentVerts
		vfijk, _ := verts[index].adjustPentVertOverage(adjRes)

		if isResClassIII(res) && vert > 0 {
			currentToLastDir := adjacentFaceDir[vfijk.face][lastFijk.face]
			fijkOrient := faceNeighbors[vfijk.face][currentToLastDir]

			ijk := vfijk.coord
			for range fijkOrient.ccwRot60 {
				ijk = ijk.rotate60ccw()
			}

			ijk = ijk.add(fijkOrient.translate.scale(unitScaleByCIIres[adjRes] * 3)).normalize()

			points = append(points, crossingPoint(oracle, lastFijk.coord, ijk,
				fijkOrient.face, adjRes, adjacentFaceDir[fijkOrient.face][vfijk.face]))
		}

		if vert < numPentVerts {
			points = append(points, vertexPoint(oracle, vfijk, adjRes))
		}

		lastFijk = vfijk
	}

	return points
}

// exactProjectionMaxRes returns the finest resolution to enumerate in full.
func exactProjectionMaxRes(t *testing.T) int {
	t.Helper()

	raw := os.Getenv("H3GO_EXACT_PROJECTION_MAXRES")
	if raw == "" {
		return exactProjectionDefaultMaxRes
	}

	maxRes, err := strconv.Atoi(raw)
	if err != nil || maxRes < 0 || maxRes > MaxResolution {
		t.Fatalf("H3GO_EXACT_PROJECTION_MAXRES = %q, want 0..%d", raw, MaxResolution)
	}

	return maxRes
}

// exactProjectionCells returns the cells to check at res: every cell when res
// is at or below maxRes, otherwise one descendant per base cell chosen by a
// fixed digit pattern, so that every resolution, face and digit sequence shape
// is exercised without enumerating the fine grids.
func exactProjectionCells(t *testing.T, res, maxRes int) []Cell {
	t.Helper()

	baseCells, err := Res0Cells()
	if err != nil {
		t.Fatal(err)
	}

	if res <= maxRes {
		cells := make([]Cell, 0, len(baseCells))

		for _, base := range baseCells {
			children, err := base.Children(res)
			if err != nil {
				t.Fatal(err)
			}

			cells = append(cells, children...)
		}

		return cells
	}

	cells := make([]Cell, 0, len(baseCells))

	for _, base := range baseCells {
		cell := base
		for childRes := 1; childRes <= res; childRes++ {
			children, err := cell.Children(childRes)
			if err != nil {
				t.Fatal(err)
			}

			// A different child at each step, varying with the base cell so the
			// sampled paths differ across the icosahedron.
			cell = children[(base.BaseCellNumber()*31+childRes*7)%len(children)]
		}

		cells = append(cells, cell)
	}

	return cells
}

// exactProjectionCases returns the resolution scenarios the two tests share.
func exactProjectionCases(t *testing.T) []struct {
	name string
	res  int
} {
	t.Helper()

	maxRes := exactProjectionMaxRes(t)
	cases := make([]struct {
		name string
		res  int
	}, 0, MaxResolution+1)

	for res := range MaxResolution + 1 {
		name := "res" + strconv.Itoa(res) + "_sampled"
		if res <= maxRes {
			name = "res" + strconv.Itoa(res) + "_all"
		}

		cases = append(cases, struct {
			name string
			res  int
		}{name: name, res: res})
	}

	return cases
}

// TestExactProjectionCenters checks cell centers: faceIJK.toVec3 against the
// exact evaluation of the same chain from the cell's integer coordinates.
func TestExactProjectionCenters(t *testing.T) {
	t.Parallel()

	oracle := newExactOracle()
	maxRes := exactProjectionMaxRes(t)

	for _, scenario := range exactProjectionCases(t) {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			worst := 0.0
			worstCell := Cell(0)

			for _, cell := range exactProjectionCells(t, scenario.res, maxRes) {
				fijk, err := cell.toFaceIjk()
				if err != nil {
					t.Fatal(err)
				}

				exactX, exactY := oracle.hex2d(fijk.coord)
				want := oracle.toVec3(exactX, exactY, fijk.face, scenario.res, false)
				got := fijk.toVec3(scenario.res)

				if sep := separationDeg(got, want); sep > worst {
					worst, worstCell = sep, cell
				}
			}

			t.Logf("worst center separation %.3g deg at %s", worst, worstCell)

			if worst > exactProjectionBoundDeg {
				t.Errorf("center of %s is %.3g deg from the exact projection, bound %.1g",
					worstCell, worst, exactProjectionBoundDeg)
			}
		})
	}
}

// TestExactProjectionBoundaries checks every point a cell boundary projects,
// topological vertices and inserted face crossings alike, against the exact
// evaluation of the substrate chain.
func TestExactProjectionBoundaries(t *testing.T) {
	t.Parallel()

	oracle := newExactOracle()
	maxRes := exactProjectionMaxRes(t)

	for _, scenario := range exactProjectionCases(t) {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			worst := 0.0
			worstCell := Cell(0)
			crossings := 0

			for _, cell := range exactProjectionCells(t, scenario.res, maxRes) {
				fijk, err := cell.toFaceIjk()
				if err != nil {
					t.Fatal(err)
				}

				points := boundaryPoints(oracle, cell, fijk)

				boundary, err := cell.Boundary()
				if err != nil || len(boundary) != len(points) {
					t.Fatalf("%s: mirrored walk has %d points, Boundary() has %d (%v)", cell, len(points), len(boundary), err)
				}

				for _, point := range points {
					want := oracle.toVec3(point.exact[0], point.exact[1], point.face, point.res, point.substrate)
					got := point.float.toVec3(point.face, point.res, point.substrate)

					if sep := separationDeg(got, want); sep > worst {
						worst, worstCell = sep, cell
					}
				}

				crossings += len(points) - topologicalVertexCount(cell)
			}

			t.Logf("worst boundary separation %.3g deg at %s, %d face crossings checked", worst, worstCell, crossings)

			if worst > exactProjectionBoundDeg {
				t.Errorf("boundary of %s is %.3g deg from the exact projection, bound %.1g",
					worstCell, worst, exactProjectionBoundDeg)
			}
		})
	}
}

// topologicalVertexCount returns how many corners the cell has.
func topologicalVertexCount(cell Cell) int {
	if cell.IsPentagon() {
		return numPentVerts
	}

	return numHexVerts
}
