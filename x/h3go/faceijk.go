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

	"github.com/uber/h3-go/v4/internal/h3core"
)

// --- Face projection ---

// closestFace returns the icosahedron face whose center is nearest to the unit
// vector v, along with the squared chord distance to that face center. The
// nearest face is the one onto which v is gnomonically projected.
func (v vec3d) closestFace() (face int, sqd float64) {
	sqd = 5.0

	for f := range NumIcosaFaces {
		s := faceCenterPoint[f].distSq(v)
		if s < sqd {
			face = f
			sqd = s
		}
	}

	return face, sqd
}

// tangentBasis returns an orthonormal basis for the plane tangent to the unit
// sphere at point v: north points along the projection of the +Z pole onto that
// plane, and east is perpendicular to it (north × v). Together they let azimuths
// be measured in the local tangent plane at v.
func (v vec3d) tangentBasis() (north, east vec3d) {
	northPole := vec3d{0, 0, 1}
	north = northPole.linComb(1.0, -northPole.dot(v), v)
	north.normalize()
	east = north.cross(v)

	return north, east
}

// azimuthRads returns the azimuth, in radians, of point p2 as seen from point v,
// measured in v's local tangent plane (clockwise from north). It is used to find
// the angle of a geographic point relative to a face center.
func (v vec3d) azimuthRads(p2 vec3d) float64 {
	north, east := v.tangentBasis()
	p2Proj := p2.linComb(1.0, -p2.dot(v), v)
	p2Proj.normalize()

	return math.Atan2(p2Proj.dot(east), p2Proj.dot(north))
}

// posAngleRads normalizes an angle in radians into the range [0, 2π).
func posAngleRads(rads float64) float64 {
	tmp := rads
	if rads < 0 {
		tmp += m2PI
	}

	if tmp >= m2PI {
		tmp -= m2PI
	}

	return tmp
}

// toHex2d gnomonically projects the unit vector v onto its closest icosahedron
// face and returns that face plus the point's 2D Hex coordinates (centered on
// the face center) at the given resolution. The radius is scaled by √7 per
// resolution and the angle is rotated for odd (Class III) resolutions. A point
// at the face center maps to the origin.
func (v vec3d) toHex2d(res int) (face int, hex vec2d) {
	var sqd float64

	face, sqd = v.closestFace()

	r := math.Acos(1 - sqd*0.5)
	if r < epsilon {
		return face, hex
	}

	theta := posAngleRads(
		faceAxesAzRadsCII[face][0] -
			posAngleRads(faceCenterPoint[face].azimuthRads(v)))

	if isResClassIII(res) {
		theta = posAngleRads(theta - mAP7RotRads)
	}

	r = math.Tan(r)
	r *= invRes0UGnomonic

	for range res {
		r *= mSqrt7
	}

	hex.x = r * math.Cos(theta)
	hex.y = r * math.Sin(theta)

	return face, hex
}

// toFaceIjk projects the unit vector v to a face and converts the resulting 2D
// Hex coordinates into face-centered IJK coordinates at the given resolution.
func (v vec3d) toFaceIjk(res int) faceIJK {
	face, hex := v.toHex2d(res)

	return faceIJK{face: face, coord: hex.toCoordIJK()}
}

// --- CoordIJK helpers ---

// normalize reduces an IJK coordinate to its canonical form by subtracting the
// minimum component from all three, so that at least one component is zero (IJK
// coordinates are only defined up to a uniform offset).
func (c *coordIJK) normalize() {
	m := c.i
	if c.j < m {
		m = c.j
	}

	if c.k < m {
		m = c.k
	}

	c.i -= m
	c.j -= m
	c.k -= m
}

// upAp7 transforms an IJK coordinate to the next coarser resolution on the
// Class II aperture-7 grid (the "up" direction), rounding to the nearest parent
// cell and re-normalizing.
func (c *coordIJK) upAp7() {
	i := c.i - c.k
	j := c.j - c.k
	c.i = int(math.Round(float64(3*i-j) * mOneSeventh))
	c.j = int(math.Round(float64(i+2*j) * mOneSeventh))
	c.k = 0
	c.normalize()
}

// upAp7r transforms an IJK coordinate to the next coarser resolution on the
// Class III (rotated) aperture-7 grid, rounding to the nearest parent cell and
// re-normalizing.
func (c *coordIJK) upAp7r() {
	i := c.i - c.k
	j := c.j - c.k
	c.i = int(math.Round(float64(2*i+j) * mOneSeventh))
	c.j = int(math.Round(float64(3*j-i) * mOneSeventh))
	c.k = 0
	c.normalize()
}

// downAp7 transforms an IJK coordinate to the next finer resolution on the
// Class II aperture-7 grid (the "down" direction, the exact inverse of upAp7),
// then re-normalizes.
func (c *coordIJK) downAp7() {
	i, j, k := c.i, c.j, c.k
	c.i = 3*i + j
	c.j = 3*j + k
	c.k = i + 3*k
	c.normalize()
}

// downAp7r transforms an IJK coordinate to the next finer resolution on the
// Class III (rotated) aperture-7 grid (the inverse of upAp7r), then
// re-normalizes.
func (c *coordIJK) downAp7r() {
	i, j, k := c.i, c.j, c.k
	c.i = 3*i + k
	c.j = i + 3*j
	c.k = j + 3*k
	c.normalize()
}

// toDigit maps a unit IJK coordinate (one of the seven cells in a single
// aperture-7 neighborhood: the center plus its six neighbors) to the
// corresponding H3 digit (0–6) via a lookup table.
func (c coordIJK) toDigit() int {
	c.normalize()
	i, j, k := c.i, c.j, c.k

	return unitIjkToDigitLUT[i][j][k]
}

// add returns the component-wise sum of two IJK coordinates.
func (c coordIJK) add(b coordIJK) coordIJK {
	return coordIJK{i: c.i + b.i, j: c.j + b.j, k: c.k + b.k}
}

// scale returns an IJK coordinate with each component multiplied by factor.
func (c coordIJK) scale(factor int) coordIJK {
	return coordIJK{i: c.i * factor, j: c.j * factor, k: c.k * factor}
}

// rotate60ccw returns the IJK coordinate rotated 60° counterclockwise about the
// origin, re-normalized.
func (c coordIJK) rotate60ccw() coordIJK {
	out := coordIJK{i: c.i + c.k, j: c.i + c.j, k: c.j + c.k}
	out.normalize()

	return out
}

// rotate60cw returns the IJK coordinate rotated 60° clockwise about the origin,
// re-normalized.
func (c coordIJK) rotate60cw() coordIJK {
	out := coordIJK{i: c.i + c.j, j: c.j + c.k, k: c.i + c.k}
	out.normalize()

	return out
}

// sub returns the component-wise difference c - b.
func (c coordIJK) sub(b coordIJK) coordIJK {
	return coordIJK{i: c.i - b.i, j: c.j - b.j, k: c.k - b.k}
}

// distance returns the grid distance between two IJK coordinates: the largest
// component of their normalized difference (normalization leaves all components
// non-negative).
func (c coordIJK) distance(b coordIJK) int {
	diff := c.sub(b)
	diff.normalize()

	return max(diff.i, diff.j, diff.k)
}

// unitToDigit maps an IJK coordinate to its digit if, once normalized, it is the
// center or one of the six unit neighbors, returning invalidDigit otherwise.
func (c coordIJK) unitToDigit() int {
	c.normalize()

	if c.i < 0 || c.i > 1 || c.j < 0 || c.j > 1 || c.k < 0 || c.k > 1 {
		return invalidDigit
	}

	return unitIjkToDigitLUT[c.i][c.j][c.k]
}

// toCube converts an IJK coordinate in place to cube coordinates, suitable for
// linear interpolation along a grid line.
func (c *coordIJK) toCube() {
	c.i = -c.i + c.k
	c.j = c.j - c.k
	c.k = -c.i - c.j
}

// fromCube converts cube coordinates in place back to a normalized IJK
// coordinate, the inverse of toCube.
func (c *coordIJK) fromCube() {
	c.i = -c.i
	c.k = 0
	c.normalize()
}

// downAp3 transforms an IJK coordinate to the next finer resolution on the
// Class II aperture-3 substrate grid (counterclockwise), then re-normalizes.
func (c *coordIJK) downAp3() {
	i, j, k := c.i, c.j, c.k
	c.i = 2*i + j
	c.j = 2*j + k
	c.k = i + 2*k
	c.normalize()
}

// downAp3r transforms an IJK coordinate to the next finer resolution on the
// Class III aperture-3 substrate grid (clockwise), then re-normalizes.
func (c *coordIJK) downAp3r() {
	i, j, k := c.i, c.j, c.k
	c.i = 2*i + k
	c.j = i + 2*j
	c.k = j + 2*k
	c.normalize()
}

// neighbor returns the IJK coordinate of the cell one step from c in the given
// digit direction. The center and invalid digits leave the coordinate
// unchanged.
func (c coordIJK) neighbor(digit int) coordIJK {
	if digit > centerDigit && digit < invalidDigit {
		c = c.add(unitVecs[digit])
		c.normalize()
	}

	return c
}

// toHex2d returns the center of the hexagon at IJK coordinate c in 2D Hex
// (Cartesian) coordinates on its face.
func (c coordIJK) toHex2d() vec2d {
	i := c.i - c.k
	j := c.j - c.k

	return vec2d{
		x: float64(i) - 0.5*float64(j),
		y: float64(j) * mSqrt3Half,
	}
}

// --- Vec2d helpers ---

// toCoordIJK converts a 2D Hex coordinate (Cartesian, on a face) into the
// nearest hexagon's normalized IJK coordinate. The rounding logic selects the
// containing hex cell, and the sign-folding handles the three 120°-symmetric
// sectors when x or y is negative.
func (v vec2d) toCoordIJK() coordIJK {
	var h coordIJK

	h.k = 0

	a1 := math.Abs(v.x)
	a2 := math.Abs(v.y)

	x2 := a2 * mRSin60
	x1 := a1 + x2/2.0

	m1 := int(x1)
	m2 := int(x2)

	r1 := x1 - float64(m1)
	r2 := x2 - float64(m2)

	if r1 < 0.5 {
		if r1 < 1.0/3.0 {
			if r2 < (1.0+r1)/2.0 {
				h.i = m1
				h.j = m2
			} else {
				h.i = m1
				h.j = m2 + 1
			}
		} else {
			if r2 < (1.0 - r1) {
				h.j = m2
			} else {
				h.j = m2 + 1
			}

			if (1.0-r1) <= r2 && r2 < (2.0*r1) {
				h.i = m1 + 1
			} else {
				h.i = m1
			}
		}
	} else {
		if r1 < 2.0/3.0 {
			if r2 < (1.0 - r1) {
				h.j = m2
			} else {
				h.j = m2 + 1
			}

			if (2.0*r1-1.0) < r2 && r2 < (1.0-r1) {
				h.i = m1
			} else {
				h.i = m1 + 1
			}
		} else {
			if r2 < (r1 / 2.0) {
				h.i = m1 + 1
				h.j = m2
			} else {
				h.i = m1 + 1
				h.j = m2 + 1
			}
		}
	}

	if v.x < 0.0 {
		if h.j%2 == 0 {
			axisi := h.j / 2
			diff := h.i - axisi
			h.i -= 2 * diff
		} else {
			axisi := (h.j + 1) / 2
			diff := h.i - axisi
			h.i -= 2*diff + 1
		}
	}

	if v.y < 0.0 {
		h.i -= (2*h.j + 1) / 2
		h.j = -h.j
	}

	h.normalize()

	return h
}

// mag returns the magnitude (length) of a 2D vector.
func (v vec2d) mag() float64 {
	return math.Sqrt(v.x*v.x + v.y*v.y)
}

// intersect returns the intersection point of the line through v,p1 and the line
// through p2,p3. The lines are assumed to intersect away from their endpoints.
func (v vec2d) intersect(p1, p2, p3 vec2d) vec2d {
	s1 := vec2d{x: p1.x - v.x, y: p1.y - v.y}
	s2 := vec2d{x: p3.x - p2.x, y: p3.y - p2.y}

	t := (s2.x*(v.y-p2.y) - s2.y*(v.x-p2.x)) /
		(-s2.x*s1.y + s1.x*s2.y)

	return vec2d{x: v.x + t*s1.x, y: v.y + t*s1.y}
}

// almostEquals reports whether two 2D vectors are equal within fltEpsilon, used
// to detect when an edge intersection coincides with an existing vertex.
func (v vec2d) almostEquals(b vec2d) bool {
	return math.Abs(v.x-b.x) < fltEpsilon && math.Abs(v.y-b.y) < fltEpsilon
}

// toVec3 converts a 2D Hex coordinate on a face into a 3D unit vector. It is the
// inverse of the gnomonic projection: the radius is inverse-scaled per
// resolution (and by a further third for substrate grids), the gnomonic scaling
// is undone with atan, and the result is placed at the computed azimuth from the
// face center. A point at the face center maps to the face center vector.
func (v vec2d) toVec3(face, res int, substrate bool) vec3d {
	r := v.mag()
	if r < epsilon {
		return faceCenterPoint[face]
	}

	theta := math.Atan2(v.y, v.x)

	for range res {
		r *= mRSqrt7
	}

	if substrate {
		// Callers only project substrate grids at Class II (even) resolutions,
		// so no Class III substrate adjustment is needed here.
		r *= mOneThird
	}

	r *= res0UGnomonic
	r = math.Atan(r)

	if !substrate && isResClassIII(res) {
		theta = posAngleRads(theta + mAP7RotRads)
	}

	theta = posAngleRads(faceAxesAzRadsCII[face][0] - theta)

	north, east := faceCenterPoint[face].tangentBasis()
	dir := north.linComb(math.Cos(theta), math.Sin(theta), east)
	out := faceCenterPoint[face].linComb(math.Cos(r), math.Sin(r), dir)
	out.normalize()

	return out
}

// --- Digit rotation ---

// rotate60ccw returns the digit reached by rotating the given H3 digit 60°
// counterclockwise about the center. The center digit is unchanged.
func rotate60ccw(digit int) int {
	switch digit {
	case kAxesDigit:
		return ikAxesDigit
	case ikAxesDigit:
		return iAxesDigit
	case iAxesDigit:
		return ijAxesDigit
	case ijAxesDigit:
		return jAxesDigit
	case jAxesDigit:
		return jkAxesDigit
	case jkAxesDigit:
		return kAxesDigit
	default:
		return digit
	}
}

// rotate60cw returns the digit reached by rotating the given H3 digit 60°
// clockwise about the center. The center digit is unchanged.
func rotate60cw(digit int) int {
	switch digit {
	case kAxesDigit:
		return jkAxesDigit
	case jkAxesDigit:
		return jAxesDigit
	case jAxesDigit:
		return ijAxesDigit
	case ijAxesDigit:
		return iAxesDigit
	case iAxesDigit:
		return ikAxesDigit
	case ikAxesDigit:
		return kAxesDigit
	default:
		return digit
	}
}

// --- H3 index rotation ---

// rotate60ccw returns c with every resolution digit rotated 60°
// counterclockwise, rotating the whole index about its base cell center.
func (c Cell) rotate60ccw() Cell {
	res := c.Resolution()
	for r := 1; r <= res; r++ {
		c = c.setIndexDigit(r, rotate60ccw(c.indexDigit(r)))
	}

	return c
}

// rotate60cw returns c with every resolution digit rotated 60° clockwise,
// rotating the whole index about its base cell center.
func (c Cell) rotate60cw() Cell {
	res := c.Resolution()
	for r := 1; r <= res; r++ {
		c = c.setIndexDigit(r, rotate60cw(c.indexDigit(r)))
	}

	return c
}

// leadingNonZeroDigit returns the first (coarsest) non-center resolution digit
// of c, or the center digit if every digit is the center.
func (c Cell) leadingNonZeroDigit() int {
	for r := 1; r <= c.Resolution(); r++ {
		d := c.indexDigit(r)
		if d != centerDigit {
			return d
		}
	}

	return centerDigit
}

// rotatePent60ccw rotates c 60° counterclockwise about a pentagon base cell
// center. Pentagons have a deleted k-axis subsequence, so as the leading digit
// is first encountered the index is rotated an extra step when needed to skip
// over the missing direction and keep the index canonical.
func (c Cell) rotatePent60ccw() Cell {
	foundFirstNonZero := false

	for r := 1; r <= c.Resolution(); r++ {
		c = c.setIndexDigit(r, rotate60ccw(c.indexDigit(r)))

		if !foundFirstNonZero && c.indexDigit(r) != centerDigit {
			foundFirstNonZero = true

			if c.leadingNonZeroDigit() == kAxesDigit {
				c = c.rotate60ccw()
			}
		}
	}

	return c
}

// rotatePent60cw rotates c 60° clockwise about a pentagon base cell center,
// skipping the deleted k-axis subsequence as the leading digit is first
// encountered so the index stays canonical.
func (c Cell) rotatePent60cw() Cell {
	foundFirstNonZero := false

	for r := 1; r <= c.Resolution(); r++ {
		c = c.setIndexDigit(r, rotate60cw(c.indexDigit(r)))

		if !foundFirstNonZero && c.indexDigit(r) != centerDigit {
			foundFirstNonZero = true

			if c.leadingNonZeroDigit() == kAxesDigit {
				c = c.rotate60cw()
			}
		}
	}

	return c
}

// --- FaceIJK to H3 index ---

// toH3 encodes a face-centered IJK coordinate at the given resolution into an H3
// cell index. It sets the mode and resolution, walks from the finest resolution
// up to the base cell (deriving each resolution digit from the offset between a
// cell and its parent's center), then applies the base cell's canonical
// rotations — with the extra pentagon handling for pentagon base cells. It
// returns ErrFailed if the coordinate is out of the encodable range.
func (fijk faceIJK) toH3(res int) (Cell, error) {
	h := Cell(h3Init)
	h |= Cell(cellMode) << modeOffset
	h |= Cell(res) << resolutionOffset

	if res == 0 {
		if fijk.coord.i > maxFaceCoord || fijk.coord.j > maxFaceCoord ||
			fijk.coord.k > maxFaceCoord {
			return 0, ErrFailed
		}

		bc := faceIjkBaseCells[fijk.face][fijk.coord.i][fijk.coord.j][fijk.coord.k].baseCell
		h |= Cell(bc) << baseCellOffset

		return h, nil
	}

	fijkBC := fijk
	ijk := &fijkBC.coord

	for r := res - 1; r >= 0; r-- {
		lastIJK := *ijk

		var lastCenter coordIJK

		if isResClassIII(r + 1) {
			ijk.upAp7()
			lastCenter = *ijk
			lastCenter.downAp7()
		} else {
			ijk.upAp7r()
			lastCenter = *ijk
			lastCenter.downAp7r()
		}

		diff := coordIJK{
			i: lastIJK.i - lastCenter.i,
			j: lastIJK.j - lastCenter.j,
			k: lastIJK.k - lastCenter.k,
		}
		diff.normalize()
		h = h.setIndexDigit(r+1, diff.toDigit())
	}

	if fijkBC.coord.i > maxFaceCoord || fijkBC.coord.j > maxFaceCoord ||
		fijkBC.coord.k > maxFaceCoord {
		return 0, ErrFailed
	}

	bcRot := faceIjkBaseCells[fijkBC.face][fijkBC.coord.i][fijkBC.coord.j][fijkBC.coord.k]
	baseCell := bcRot.baseCell
	numRots := bcRot.ccwRot60

	h |= Cell(baseCell) << baseCellOffset

	if h3core.IsBaseCellPentagon[baseCell] {
		if h.leadingNonZeroDigit() == kAxesDigit {
			offsets := baseCellCWOffsetPent[baseCell]
			if offsets[0] == fijkBC.face || offsets[1] == fijkBC.face {
				h = h.rotate60cw()
			} else {
				h = h.rotate60ccw()
			}
		}

		for range numRots {
			h = h.rotatePent60ccw()
		}
	} else {
		for range numRots {
			h = h.rotate60ccw()
		}
	}

	return h, nil
}

// --- FaceIJK to Vec3d / center point ---

// toVec3 returns the 3D unit vector at the center of the cell addressed by fijk
// at the given resolution.
func (fijk faceIJK) toVec3(res int) vec3d {
	return fijk.coord.toHex2d().toVec3(fijk.face, res, false)
}

// --- H3 index to FaceIJK ---

// toFaceIjkWithInitializedFijk walks c's resolution digits down from its base
// cell's home coordinate (passed in fijk) to the face-centered IJK coordinate of
// the cell. It returns the updated address and whether the result may have
// overflowed onto an adjacent face (possibleOverage), which the caller resolves.
func (c Cell) toFaceIjkWithInitializedFijk(fijk faceIJK) (faceIJK, bool) {
	res := c.Resolution()

	// A center base cell hierarchy with no off-center digits stays on this face.
	isCenter := fijk.coord.i == 0 && fijk.coord.j == 0 && fijk.coord.k == 0
	possibleOverage := h3core.IsBaseCellPentagon[c.BaseCellNumber()] || (res != 0 && !isCenter)

	ijk := fijk.coord

	for r := 1; r <= res; r++ {
		if isResClassIII(r) { // rotate ccw
			ijk.downAp7()
		} else { // Class II: rotate cw
			ijk.downAp7r()
		}

		ijk = ijk.neighbor(c.indexDigit(r))
	}

	fijk.coord = ijk

	return fijk, possibleOverage
}

// toFaceIjk converts an H3 cell into its FaceIJK address, resolving any overage
// onto the correct adjacent icosahedron face (with the extra handling pentagon
// base cells require). It returns ErrCellInvalid if the index's base cell is out
// of range.
func (c Cell) toFaceIjk() (faceIJK, error) {
	baseCell := c.BaseCellNumber()
	if baseCell >= NumBaseCells {
		return faceIJK{}, ErrCellInvalid
	}

	// Adjust for the pentagonal missing sequence: all of sub-sequence 5 needs to
	// be rotated (and some of sub-sequence 4 is handled during overage below).
	if h3core.IsBaseCellPentagon[baseCell] && c.leadingNonZeroDigit() == ikAxesDigit {
		c = c.rotate60cw()
	}

	fijk, possibleOverage := c.toFaceIjkWithInitializedFijk(baseCellHomeFijk[baseCell])
	if !possibleOverage {
		return fijk, nil
	}

	origIJK := fijk.coord

	// If we're in Class III, drop into the next finer Class II grid to adjust.
	res := c.Resolution()
	if isResClassIII(res) {
		fijk.coord.downAp7r()

		res++
	}

	// A pentagon base cell with a leading 4 digit requires special handling.
	pentLeading4 := h3core.IsBaseCellPentagon[baseCell] && c.leadingNonZeroDigit() == iAxesDigit

	var ov overage

	fijk, ov = fijk.adjustOverageClassII(res, pentLeading4, false)
	if ov != noOverage {
		// A pentagon base cell can have secondary overages.
		if h3core.IsBaseCellPentagon[baseCell] {
			for {
				var o overage

				fijk, o = fijk.adjustOverageClassII(res, false, false)
				if o == noOverage {
					break
				}
			}
		}

		if res != c.Resolution() {
			fijk.coord.upAp7r()
		}
	} else if res != c.Resolution() {
		fijk.coord = origIJK
	}

	return fijk, nil
}

// adjustOverageClassII adjusts a FaceIJK address so it is expressed relative to
// the correct icosahedron face when the coordinate has spilled past the edge of
// its current face. pentLeading4 selects the pentagon missing-sequence fix-up,
// and substrate selects the finer substrate grid used while building cell
// boundaries. It returns the adjusted address and the overage classification.
func (fijk faceIJK) adjustOverageClassII(res int, pentLeading4, substrate bool) (faceIJK, overage) {
	ov := noOverage
	ijk := fijk.coord

	maxDim := maxDimByCIIres[res]
	if substrate {
		maxDim *= 3
	}

	sum := ijk.i + ijk.j + ijk.k

	switch {
	case substrate && sum == maxDim: // on the face edge
		ov = faceEdge
	case sum > maxDim: // overage onto an adjacent face
		ov = newFace

		var fijkOrient faceOrientIJK

		switch {
		case ijk.k > 0 && ijk.j > 0: // jk quadrant
			fijkOrient = faceNeighbors[fijk.face][dirJK]
		case ijk.k > 0: // ik quadrant
			fijkOrient = faceNeighbors[fijk.face][dirKI]

			if pentLeading4 {
				// Translate to the pentagon center, rotate to skip the missing
				// sequence, then translate back to the triangle center.
				tmp := coordIJK{i: ijk.i - maxDim, j: ijk.j, k: ijk.k}.rotate60cw()
				ijk = coordIJK{i: tmp.i + maxDim, j: tmp.j, k: tmp.k}
			}
		default: // ij quadrant
			fijkOrient = faceNeighbors[fijk.face][dirIJ]
		}

		fijk.face = fijkOrient.face

		for range fijkOrient.ccwRot60 {
			ijk = ijk.rotate60ccw()
		}

		unitScale := unitScaleByCIIres[res]
		if substrate {
			unitScale *= 3
		}

		ijk = ijk.add(fijkOrient.translate.scale(unitScale))
		ijk.normalize()

		// Overage points on pentagon boundaries can end up on a face edge.
		if substrate && ijk.i+ijk.j+ijk.k == maxDim {
			ov = faceEdge
		}
	}

	fijk.coord = ijk

	return fijk, ov
}

// adjustPentVertOverage repeatedly applies the substrate overage adjustment to a
// pentagon vertex until it no longer crosses onto a new face, returning the
// final address and overage classification.
func (fijk faceIJK) adjustPentVertOverage(res int) (faceIJK, overage) {
	var ov overage

	for {
		fijk, ov = fijk.adjustOverageClassII(res, false, true)
		if ov != newFace {
			break
		}
	}

	return fijk, ov
}

// toVerts returns the substrate FaceIJK addresses of a hexagon's six vertices,
// along with the (possibly incremented) substrate resolution. The cell center is
// moved into an aperture-33r substrate grid, and Class III cells get an extra
// clockwise aperture-7 step to land on a Class II substrate.
func (fijk faceIJK) toVerts(res int) (int, [numHexVerts]faceIJK) {
	// Vertices of an origin-centered cell, listed ccw from the i-axis, in the
	// Class II and Class III substrate grids respectively.
	vertsCII := [numHexVerts]coordIJK{
		{2, 1, 0}, {1, 2, 0}, {0, 2, 1}, {0, 1, 2}, {1, 0, 2}, {2, 0, 1},
	}
	vertsCIII := [numHexVerts]coordIJK{
		{5, 4, 0}, {1, 5, 0}, {0, 5, 4}, {0, 1, 5}, {4, 0, 5}, {5, 0, 1},
	}

	verts := vertsCII
	if isResClassIII(res) {
		verts = vertsCIII
	}

	fijk.coord.downAp3()
	fijk.coord.downAp3r()

	if isResClassIII(res) {
		fijk.coord.downAp7r()

		res++
	}

	var out [numHexVerts]faceIJK

	for i := range numHexVerts {
		coord := fijk.coord.add(verts[i])
		coord.normalize()
		out[i] = faceIJK{face: fijk.face, coord: coord}
	}

	return res, out
}

// pentToVerts returns the substrate FaceIJK addresses of a pentagon's five
// vertices, along with the (possibly incremented) substrate resolution, using
// the same substrate construction as toVerts.
func (fijk faceIJK) pentToVerts(res int) (int, [numPentVerts]faceIJK) {
	vertsCII := [numPentVerts]coordIJK{
		{2, 1, 0}, {1, 2, 0}, {0, 2, 1}, {0, 1, 2}, {1, 0, 2},
	}
	vertsCIII := [numPentVerts]coordIJK{
		{5, 4, 0}, {1, 5, 0}, {0, 5, 4}, {0, 1, 5}, {4, 0, 5},
	}

	verts := vertsCII
	if isResClassIII(res) {
		verts = vertsCIII
	}

	fijk.coord.downAp3()
	fijk.coord.downAp3r()

	if isResClassIII(res) {
		fijk.coord.downAp7r()

		res++
	}

	var out [numPentVerts]faceIJK

	for i := range numPentVerts {
		coord := fijk.coord.add(verts[i])
		coord.normalize()
		out[i] = faceIJK{face: fijk.face, coord: coord}
	}

	return res, out
}

// invalidFace marks an unused slot while collecting a cell's icosahedron faces.
const invalidFace = -1

// IcosahedronFaces returns the icosahedron faces (0-19) that the cell intersects,
// in no particular order. A hexagon touches one or two faces; a pentagon touches
// five.
func (c Cell) IcosahedronFaces() ([]int, error) {
	res := c.Resolution()
	isPent := c.IsPentagon()

	// Class II pentagons have every vertex on an icosahedron edge, so the
	// vertex-based check is ambiguous. Their direct child pentagons cross the
	// same faces, so use those instead. A Class II pentagon is at an even
	// resolution below the maximum, so the center child always exists.
	if isPent && !isResClassIII(res) {
		childPentagon, _ := c.CenterChild(res + 1)

		return childPentagon.IcosahedronFaces()
	}

	fijk, err := c.toFaceIjk()
	if err != nil {
		return nil, err
	}

	var (
		vertexCount int
		fijkVerts   [numHexVerts]faceIJK
		adjRes      int
	)

	if isPent {
		vertexCount = numPentVerts

		var pentVerts [numPentVerts]faceIJK

		adjRes, pentVerts = fijk.pentToVerts(res)
		copy(fijkVerts[:], pentVerts[:])
	} else {
		vertexCount = numHexVerts
		adjRes, fijkVerts = fijk.toVerts(res)
	}

	// A pentagon touches five faces, a hexagon at most two.
	faceCount := numEdgeCells
	if isPent {
		faceCount = numPentVerts
	}

	faces := make([]int, faceCount)
	for i := range faces {
		faces[i] = invalidFace
	}

	for i := range vertexCount {
		vert := fijkVerts[i]
		if isPent {
			vert, _ = vert.adjustPentVertOverage(adjRes)
		} else {
			vert, _ = vert.adjustOverageClassII(adjRes, false, true)
		}

		// Use the output array as a small hash set: find the first empty slot or
		// the slot already holding this face.
		pos := 0
		for faces[pos] != invalidFace && faces[pos] != vert.face {
			pos++

			if pos >= faceCount {
				return nil, ErrFailed
			}
		}

		faces[pos] = vert.face
	}

	out := make([]int, 0, faceCount)

	for _, face := range faces {
		if face != invalidFace {
			out = append(out, face)
		}
	}

	return out, nil
}
