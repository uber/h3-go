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

// vec3ToClosestFace returns the icosahedron face whose center is nearest to the
// unit vector v, along with the squared chord distance to that face center. The
// nearest face is the one onto which v is gnomonically projected.
func vec3ToClosestFace(v vec3d) (face int, sqd float64) {
	sqd = 5.0

	for f := range numIcosaFaces {
		s := vec3DistSq(faceCenterPoint[f], v)
		if s < sqd {
			face = f
			sqd = s
		}
	}

	return face, sqd
}

// vec3TangentBasis returns an orthonormal basis for the plane tangent to the
// unit sphere at point p: north points along the projection of the +Z pole onto
// that plane, and east is perpendicular to it (north × p). Together they let
// azimuths be measured in the local tangent plane at p.
func vec3TangentBasis(p vec3d) (north, east vec3d) {
	northPole := vec3d{0, 0, 1}
	north = vec3LinComb(1.0, northPole, -vec3Dot(northPole, p), p)
	vec3Normalize(&north)
	east = vec3Cross(north, p)

	return north, east
}

// vec3AzimuthRads returns the azimuth, in radians, of point p2 as seen from
// point p1, measured in p1's local tangent plane (clockwise from north). It is
// used to find the angle of a geographic point relative to a face center.
func vec3AzimuthRads(p1, p2 vec3d) float64 {
	north, east := vec3TangentBasis(p1)
	p2Proj := vec3LinComb(1.0, p2, -vec3Dot(p2, p1), p1)
	vec3Normalize(&p2Proj)

	return math.Atan2(vec3Dot(p2Proj, east), vec3Dot(p2Proj, north))
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

// vec3ToHex2d gnomonically projects the unit vector p onto its closest
// icosahedron face and returns that face plus the point's 2D Hex coordinates
// (centered on the face center) at the given resolution. The radius is scaled
// by √7 per resolution and the angle is rotated for odd (Class III)
// resolutions. A point at the face center maps to the origin.
func vec3ToHex2d(p vec3d, res int) (face int, v vec2d) {
	var sqd float64

	face, sqd = vec3ToClosestFace(p)

	r := math.Acos(1 - sqd*0.5)
	if r < epsilon {
		return face, v
	}

	theta := posAngleRads(
		faceAxesAzRadsCII[face][0] -
			posAngleRads(vec3AzimuthRads(faceCenterPoint[face], p)))

	if res%2 == 1 {
		theta = posAngleRads(theta - mAP7RotRads)
	}

	r = math.Tan(r)
	r *= invRes0UGnomonic

	for range res {
		r *= mSqrt7
	}

	v.x = r * math.Cos(theta)
	v.y = r * math.Sin(theta)

	return face, v
}

// vec3ToFaceIjk projects the unit vector p to a face and converts the resulting
// 2D Hex coordinates into face-centered IJK coordinates at the given resolution.
func vec3ToFaceIjk(p vec3d, res int) faceIJK {
	face, v := vec3ToHex2d(p, res)

	return faceIJK{face: face, coord: hex2dToCoordIJK(v)}
}

// --- CoordIJK helpers ---

// ijkNormalize reduces an IJK coordinate to its canonical form by subtracting
// the minimum component from all three, so that at least one component is zero
// (IJK coordinates are only defined up to a uniform offset).
func ijkNormalize(c *coordIJK) {
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

// hex2dToCoordIJK converts a 2D Hex coordinate (Cartesian, on a face) into the
// nearest hexagon's normalized IJK coordinate. The rounding logic selects the
// containing hex cell, and the sign-folding handles the three 120°-symmetric
// sectors when x or y is negative.
func hex2dToCoordIJK(v vec2d) coordIJK {
	var h coordIJK

	h.k = 0

	a1 := math.Abs(v.x)
	a2 := math.Abs(v.y)

	x2 := a2 * mRSin60
	x1 := a1 + x2/2.0 //nolint:mnd // hex grid rounding

	m1 := int(x1)
	m2 := int(x2)

	r1 := x1 - float64(m1)
	r2 := x2 - float64(m2)

	if r1 < 0.5 { //nolint:mnd // hex grid rounding
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

			if (1.0-r1) <= r2 && r2 < (2.0*r1) { //nolint:mnd // hex grid rounding
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
			if r2 < (r1 / 2.0) { //nolint:mnd // hex grid rounding
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
			axisi := h.j / 2 //nolint:mnd // hex grid axis folding
			diff := h.i - axisi
			h.i -= 2 * diff //nolint:mnd // hex grid axis folding
		} else {
			axisi := (h.j + 1) / 2 //nolint:mnd // hex grid axis folding
			diff := h.i - axisi
			h.i -= 2*diff + 1 //nolint:mnd // hex grid axis folding
		}
	}

	if v.y < 0.0 {
		h.i -= (2*h.j + 1) / 2 //nolint:mnd // hex grid axis folding
		h.j = -h.j
	}

	ijkNormalize(&h)

	return h
}

// upAp7 transforms an IJK coordinate to the next coarser resolution on the
// Class II aperture-7 grid (the "up" direction), rounding to the nearest parent
// cell and re-normalizing.
func upAp7(ijk *coordIJK) {
	i := ijk.i - ijk.k
	j := ijk.j - ijk.k
	ijk.i = int(math.Round(float64(3*i-j) * mOneSeventh))
	ijk.j = int(math.Round(float64(i+2*j) * mOneSeventh))
	ijk.k = 0
	ijkNormalize(ijk)
}

// upAp7r transforms an IJK coordinate to the next coarser resolution on the
// Class III (rotated) aperture-7 grid, rounding to the nearest parent cell and
// re-normalizing.
func upAp7r(ijk *coordIJK) {
	i := ijk.i - ijk.k
	j := ijk.j - ijk.k
	ijk.i = int(math.Round(float64(2*i+j) * mOneSeventh))
	ijk.j = int(math.Round(float64(3*j-i) * mOneSeventh))
	ijk.k = 0
	ijkNormalize(ijk)
}

// downAp7 transforms an IJK coordinate to the next finer resolution on the
// Class II aperture-7 grid (the "down" direction, the exact inverse of upAp7),
// then re-normalizes.
func downAp7(ijk *coordIJK) {
	i, j, k := ijk.i, ijk.j, ijk.k
	ijk.i = 3*i + j //nolint:mnd // aperture-7 matrix coefficients
	ijk.j = 3*j + k //nolint:mnd // aperture-7 matrix coefficients
	ijk.k = i + 3*k //nolint:mnd // aperture-7 matrix coefficients
	ijkNormalize(ijk)
}

// downAp7r transforms an IJK coordinate to the next finer resolution on the
// Class III (rotated) aperture-7 grid (the inverse of upAp7r), then
// re-normalizes.
func downAp7r(ijk *coordIJK) {
	i, j, k := ijk.i, ijk.j, ijk.k
	ijk.i = 3*i + k //nolint:mnd // aperture-7 matrix coefficients
	ijk.j = i + 3*j //nolint:mnd // aperture-7 matrix coefficients
	ijk.k = j + 3*k //nolint:mnd // aperture-7 matrix coefficients
	ijkNormalize(ijk)
}

// unitIjkToDigit maps a unit IJK coordinate (one of the seven cells in a single
// aperture-7 neighborhood: the center plus its six neighbors) to the
// corresponding H3 digit (0–6) via a lookup table.
func unitIjkToDigit(ijk coordIJK) int {
	ijkNormalize(&ijk)
	i, j, k := ijk.i, ijk.j, ijk.k

	return unitIjkToDigitLUT[i][j][k]
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

// h3Rotate60ccw returns h with every resolution digit rotated 60°
// counterclockwise, rotating the whole index about its base cell center.
func h3Rotate60ccw(h Cell) Cell {
	res := resolution(h)
	for r := 1; r <= res; r++ {
		h = setIndexDigit(h, r, rotate60ccw(getIndexDigit(h, r)))
	}

	return h
}

// h3Rotate60cw returns h with every resolution digit rotated 60° clockwise,
// rotating the whole index about its base cell center.
func h3Rotate60cw(h Cell) Cell {
	res := resolution(h)
	for r := 1; r <= res; r++ {
		h = setIndexDigit(h, r, rotate60cw(getIndexDigit(h, r)))
	}

	return h
}

// h3LeadingNonZeroDigit returns the first (coarsest) non-center resolution digit
// of h, or the center digit if every digit is the center.
func h3LeadingNonZeroDigit(h Cell) int {
	for r := 1; r <= resolution(h); r++ {
		d := getIndexDigit(h, r)
		if d != centerDigit {
			return d
		}
	}

	return centerDigit
}

// h3RotatePent60ccw rotates h 60° counterclockwise about a pentagon base cell
// center. Pentagons have a deleted k-axis subsequence, so as the leading digit
// is first encountered the index is rotated an extra step when needed to skip
// over the missing direction and keep the index canonical.
func h3RotatePent60ccw(h Cell) Cell {
	foundFirstNonZero := false

	for r := 1; r <= resolution(h); r++ {
		h = setIndexDigit(h, r, rotate60ccw(getIndexDigit(h, r)))

		if !foundFirstNonZero && getIndexDigit(h, r) != centerDigit {
			foundFirstNonZero = true

			if h3LeadingNonZeroDigit(h) == kAxesDigit {
				h = h3Rotate60ccw(h)
			}
		}
	}

	return h
}

// --- FaceIJK to H3 index ---

// faceIjkToH3 encodes a face-centered IJK coordinate at the given resolution
// into an H3 cell index. It sets the mode and resolution, walks from the finest
// resolution up to the base cell (deriving each resolution digit from the offset
// between a cell and its parent's center), then applies the base cell's
// canonical rotations — with the extra pentagon handling for pentagon base
// cells. It returns ErrFailed if the coordinate is out of the encodable range.
func faceIjkToH3(fijk faceIJK, res int) (Cell, error) {
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

		if (r+1)%2 == 1 { // class III
			upAp7(ijk)
			lastCenter = *ijk
			downAp7(&lastCenter)
		} else {
			upAp7r(ijk)
			lastCenter = *ijk
			downAp7r(&lastCenter)
		}

		diff := coordIJK{
			i: lastIJK.i - lastCenter.i,
			j: lastIJK.j - lastCenter.j,
			k: lastIJK.k - lastCenter.k,
		}
		ijkNormalize(&diff)
		h = setIndexDigit(h, r+1, unitIjkToDigit(diff))
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
		if h3LeadingNonZeroDigit(h) == kAxesDigit {
			offsets := baseCellCWOffsetPent[baseCell]
			if offsets[0] == fijkBC.face || offsets[1] == fijkBC.face {
				h = h3Rotate60cw(h)
			} else {
				h = h3Rotate60ccw(h)
			}
		}

		for range numRots {
			h = h3RotatePent60ccw(h)
		}
	} else {
		for range numRots {
			h = h3Rotate60ccw(h)
		}
	}

	return h, nil
}
