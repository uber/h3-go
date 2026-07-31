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
	"cmp"
	"math"

	"github.com/uber/h3-go/v4/internal/h3core"
)

// CoordIJ holds IJ hexagon coordinates anchored at an origin cell. The two axes
// are spaced 120° apart. Coordinates are only comparable when produced from the
// same origin, and the encoding is not guaranteed stable across H3 versions.
type CoordIJ struct {
	I, J int
}

// getBaseCellDirection returns the digit direction from one base cell to a
// neighboring base cell, or invalidDigit if they are not neighbors.
func getBaseCellDirection(originBaseCell, neighborBaseCell int) int {
	for dir := centerDigit; dir < numDigits; dir++ {
		if baseCellNeighbors[originBaseCell][dir] == neighborBaseCell {
			return dir
		}
	}

	return invalidDigit
}

// cubeRound rounds floating-point cube coordinates to the nearest valid integer
// cube coordinate (where i+j+k == 0), per the standard hex rounding algorithm.
func cubeRound(iCoord, jCoord, kCoord float64) coordIJK {
	roundI := int(math.Round(iCoord))
	roundJ := int(math.Round(jCoord))
	roundK := int(math.Round(kCoord))

	iDiff := math.Abs(float64(roundI) - iCoord)
	jDiff := math.Abs(float64(roundJ) - jCoord)
	kDiff := math.Abs(float64(roundK) - kCoord)

	switch {
	case iDiff > jDiff && iDiff > kDiff:
		roundI = -roundJ - roundK
	case jDiff > kDiff:
		roundJ = -roundI - roundK
	default:
		roundK = -roundI - roundJ
	}

	return coordIJK{i: roundI, j: roundJ, k: roundK}
}

// cellToLocalIjk returns the IJK coordinates of target in the coordinate system
// anchored at c (the origin). The coordinate space may have deleted regions or
// pentagon warping, so it can fail when target is too far from the origin or on
// the far side of a pentagon.
func (c Cell) cellToLocalIjk(target Cell) (coordIJK, error) {
	res := c.Resolution()
	if res != target.Resolution() {
		return coordIJK{}, ErrResolutionMismatch
	}

	originBaseCell := c.BaseCellNumber()
	baseCell := target.BaseCellNumber()

	if originBaseCell >= NumBaseCells || baseCell >= NumBaseCells {
		return coordIJK{}, ErrCellInvalid
	}

	// Direction from origin base cell to the target base cell (and back).
	dir := centerDigit
	revDir := centerDigit

	if originBaseCell != baseCell {
		dir = getBaseCellDirection(originBaseCell, baseCell)
		if dir == invalidDigit {
			return coordIJK{}, ErrFailed
		}

		revDir = getBaseCellDirection(baseCell, originBaseCell)
	}

	originOnPent := h3core.IsBaseCellPentagon[originBaseCell]
	indexOnPent := h3core.IsBaseCellPentagon[baseCell]

	index := target

	if dir != centerDigit {
		// Rotate the target into the origin base cell's orientation, undoing the
		// rotation into that base cell (hence clockwise).
		baseCellRotations := baseCellNeighbor60CCWRots[originBaseCell][dir]
		if indexOnPent {
			for range baseCellRotations {
				index = index.rotatePent60cw()

				revDir = rotate60cw(revDir)
				if revDir == kAxesDigit {
					revDir = rotate60cw(revDir)
				}
			}
		} else {
			for range baseCellRotations {
				index = index.rotate60cw()
				revDir = rotate60cw(revDir)
			}
		}
	}

	// Face is unused: this produces coordinates in base cell coordinate space.
	fijk, _ := index.toFaceIjkWithInitializedFijk(faceIJK{})
	coord := fijk.coord

	switch {
	case dir != centerDigit:
		pentagonRots := 0
		directionRots := 0

		if originOnPent {
			originLeadingDigit := c.leadingNonZeroDigit()
			if originLeadingDigit == invalidDigit {
				return coordIJK{}, ErrCellInvalid
			}

			if failedDirections[originLeadingDigit][dir] {
				return coordIJK{}, ErrFailed
			}

			directionRots = pentagonRotations[originLeadingDigit][dir]
			pentagonRots = directionRots
		} else if indexOnPent {
			indexLeadingDigit := index.leadingNonZeroDigit()
			if indexLeadingDigit == invalidDigit {
				return coordIJK{}, ErrCellInvalid
			}

			if failedDirections[indexLeadingDigit][revDir] {
				return coordIJK{}, ErrFailed
			}

			pentagonRots = pentagonRotations[revDir][indexLeadingDigit]
		}

		if pentagonRots < 0 || directionRots < 0 {
			return coordIJK{}, ErrCellInvalid
		}

		for range pentagonRots {
			coord = coord.rotate60cw()
		}

		offset := coordIJK{}.neighbor(dir)
		// Scale the offset to the index resolution.
		for r := res - 1; r >= 0; r-- {
			if isResClassIII(r + 1) {
				offset.downAp7()
			} else {
				offset.downAp7r()
			}
		}

		for range directionRots {
			offset = offset.rotate60cw()
		}

		coord = coord.add(offset)
		coord.normalize()
	case originOnPent && indexOnPent:
		// Same pentagon base cell.
		originLeadingDigit := c.leadingNonZeroDigit()
		indexLeadingDigit := index.leadingNonZeroDigit()

		if originLeadingDigit == invalidDigit || indexLeadingDigit == invalidDigit {
			return coordIJK{}, ErrCellInvalid
		}

		if failedDirections[originLeadingDigit][indexLeadingDigit] {
			return coordIJK{}, ErrFailed
		}

		withinPentagonRots := pentagonRotations[originLeadingDigit][indexLeadingDigit]
		for range withinPentagonRots {
			coord = coord.rotate60cw()
		}
	}

	return coord, nil
}

// localIjkToCell returns the cell at the given IJK coordinates in the coordinate
// system anchored at c (the origin). It can fail when the coordinates are too
// far from the origin or on the far side of a pentagon.
func (c Cell) localIjkToCell(ijk coordIJK) (Cell, error) {
	res := c.Resolution()

	originBaseCell := c.BaseCellNumber()
	if originBaseCell >= NumBaseCells {
		return 0, ErrCellInvalid
	}

	originOnPent := h3core.IsBaseCellPentagon[originBaseCell]

	out := Cell(h3Init) | Cell(cellMode)<<modeOffset
	out = out.setResolution(res)

	// res 0 / base cell case.
	if res == 0 {
		dir := ijk.unitToDigit()
		if dir == invalidDigit {
			return 0, ErrFailed
		}

		newBaseCell := baseCellNeighbors[originBaseCell][dir]
		if newBaseCell == invalidBaseCell {
			return 0, ErrFailed
		}

		return out.setBaseCell(newBaseCell), nil
	}

	// Build the index digits from finest resolution up, recovering the base
	// cell's IJK in its own coordinate system in ijkCopy.
	ijkCopy := ijk
	for r := res - 1; r >= 0; r-- {
		lastIJK := ijkCopy

		var lastCenter coordIJK

		if isResClassIII(r + 1) {
			ijkCopy.upAp7()
			lastCenter = ijkCopy
			lastCenter.downAp7()
		} else {
			ijkCopy.upAp7r()
			lastCenter = ijkCopy
			lastCenter.downAp7r()
		}

		diff := lastIJK.sub(lastCenter)
		diff.normalize()
		out = out.setIndexDigit(r+1, diff.unitToDigit())
	}

	if ijkCopy.i > 1 || ijkCopy.j > 1 || ijkCopy.k > 1 {
		return 0, ErrFailed
	}

	dir := ijkCopy.unitToDigit()
	baseCell := baseCellNeighbors[originBaseCell][dir]

	indexOnPent := false
	if baseCell != invalidBaseCell {
		indexOnPent = h3core.IsBaseCellPentagon[baseCell]
	}

	switch {
	case dir != centerDigit:
		var err error

		out, baseCell, err = c.localIjkToCellOffOrigin(out, dir, originBaseCell, baseCell, originOnPent, indexOnPent)
		if err != nil {
			return 0, err
		}
	case originOnPent && indexOnPent:
		originLeadingDigit := c.leadingNonZeroDigit()
		indexLeadingDigit := out.leadingNonZeroDigit()

		if originLeadingDigit == invalidDigit || indexLeadingDigit == invalidDigit {
			return 0, ErrCellInvalid
		}

		withinPentagonRots := pentagonRotationsReverse[originLeadingDigit][indexLeadingDigit]
		if withinPentagonRots < 0 {
			return 0, ErrCellInvalid
		}

		for range withinPentagonRots {
			out = out.rotate60ccw()
		}
	}

	if indexOnPent {
		// Fail if the recovered index lands on the deleted k subsequence.
		if out.leadingNonZeroDigit() == kAxesDigit {
			return 0, ErrPentagon
		}
	}

	return out.setBaseCell(baseCell), nil
}

// localIjkToCellOffOrigin applies the base-cell and pentagon rotations for the
// case where the recovered coordinate moves off the origin base cell (the
// direction is not the center). It returns the rotated partial cell and the
// resolved destination base cell, or an error when the move crosses pentagon
// distortion or lands on an invalid coordinate.
func (c Cell) localIjkToCellOffOrigin(out Cell, dir, originBaseCell, baseCell int, originOnPent, indexOnPent bool) (Cell, int, error) {
	pentagonRots := 0

	if originOnPent {
		originLeadingDigit := c.leadingNonZeroDigit()
		if originLeadingDigit == invalidDigit {
			return 0, 0, ErrCellInvalid
		}

		pentagonRots = pentagonRotationsReverse[originLeadingDigit][dir]
		for range pentagonRots {
			dir = rotate60ccw(dir)
		}

		// The pentagon rotations avoid the deleted direction; if we still land on
		// it the coordinate crosses pentagon distortion.
		if dir == kAxesDigit {
			return 0, 0, ErrPentagon
		}

		baseCell = baseCellNeighbors[originBaseCell][dir]
	}

	baseCellRotations := baseCellNeighbor60CCWRots[originBaseCell][dir]
	if indexOnPent {
		revDir := getBaseCellDirection(baseCell, originBaseCell)

		// Adjust for the two base cells' coordinate spaces first, so the pentagon
		// rotations use the leading digit in pentagon space.
		for range baseCellRotations {
			out = out.rotate60ccw()
		}

		indexLeadingDigit := out.leadingNonZeroDigit()
		if isBaseCellPolarPentagon(baseCell) {
			pentagonRots = pentagonRotationsReversePolar[revDir][indexLeadingDigit]
		} else {
			pentagonRots = pentagonRotationsReverseNonpolar[revDir][indexLeadingDigit]
		}

		// A negative entry would require revDir to be the k axis, which is
		// impossible from a pentagon index base cell, so a range over it simply
		// does nothing here.
		for range pentagonRots {
			out = out.rotatePent60ccw()
		}

		return out, baseCell, nil
	}

	if pentagonRots < 0 {
		return 0, 0, ErrCellInvalid
	}

	for range pentagonRots {
		out = out.rotate60ccw()
	}

	for range baseCellRotations {
		out = out.rotate60ccw()
	}

	return out, baseCell, nil
}

// CellToLocalIJ returns the IJ coordinates of cell anchored by origin. The
// coordinate space may have deleted regions or pentagon warping, and is only
// comparable for cells produced from the same origin.
func CellToLocalIJ(origin, cell Cell) (CoordIJ, error) {
	ijk, err := origin.cellToLocalIjk(cell)
	if err != nil {
		return CoordIJ{}, err
	}

	return CoordIJ{I: ijk.i - ijk.k, J: ijk.j - ijk.k}, nil
}

// LocalIJToCell returns the cell at the IJ coordinates anchored by origin. It is
// the inverse of CellToLocalIJ, subject to the same pentagon and range limits.
func LocalIJToCell(origin Cell, ij CoordIJ) (Cell, error) {
	ijk := coordIJK{i: ij.I, j: ij.J, k: 0}
	ijk.normalize()

	return origin.localIjkToCell(ijk)
}

// GridDistance returns the grid distance (number of steps) between c and other,
// or an error if the distance cannot be computed, such as when the cells are far
// apart or on opposite sides of a pentagon.
func (c Cell) GridDistance(other Cell) (int, error) {
	originIjk, err := c.cellToLocalIjk(c)
	if err != nil {
		return 0, err
	}

	otherIjk, err := c.cellToLocalIjk(other)
	if err != nil {
		return 0, err
	}

	return originIjk.distance(otherIjk), nil
}

// GridDistance returns the grid distance between two cells.
func GridDistance(a, b Cell) (int, error) {
	return a.GridDistance(b)
}

// GridPath returns the line of cells from c to other inclusive. Consecutive
// cells are neighbors and the line length is GridDistance+1. It can fail when
// the distance cannot be computed or the line crosses pentagon distortion.
func (c Cell) GridPath(other Cell) ([]Cell, error) {
	distance, err := c.GridDistance(other)
	if err != nil {
		return nil, err
	}

	out := make([]Cell, distance+1)
	if distance == 0 {
		out[0] = c
		return out, nil
	}

	// Straight-line interpolation in local IJK space anchored at c.
	if gridPathInterpolate(c, other, distance, out, 0, 1) == nil {
		return out, nil
	}

	// The forward attempt can fail crossing pentagon distortion; retrying
	// anchored at the other endpoint (filling in reverse) resolves those cases
	// for any valid pair, so its result is returned directly.
	return out, gridPathInterpolate(other, c, distance, out, distance, -1)
}

// GridPath returns the line of cells between two cells inclusive.
func GridPath(a, b Cell) ([]Cell, error) {
	return a.GridPath(b)
}

// gridPathInterpolate fills out with the line from start to end by interpolating
// through the local IJK space anchored at start. out[outOffset + outStep*n]
// receives the nth cell, so outStep of -1 fills the line in reverse. It fails if
// an intermediate coordinate cannot be mapped back to a cell.
func gridPathInterpolate(start, end Cell, distance int, out []Cell, outOffset, outStep int) error {
	startIjk, startErr := start.cellToLocalIjk(start)
	endIjk, endErr := start.cellToLocalIjk(end)

	// start-to-start always succeeds; end-to-start can fail when this is the
	// reverse attempt across pentagon distortion.
	if err := cmp.Or(startErr, endErr); err != nil {
		return err
	}

	startIjk.toCube()
	endIjk.toCube()

	invDistance := 1.0 / float64(distance)
	iStep := float64(endIjk.i-startIjk.i) * invDistance
	jStep := float64(endIjk.j-startIjk.j) * invDistance
	kStep := float64(endIjk.k-startIjk.k) * invDistance

	for n := 0; n <= distance; n++ {
		current := cubeRound(
			float64(startIjk.i)+iStep*float64(n),
			float64(startIjk.j)+jStep*float64(n),
			float64(startIjk.k)+kStep*float64(n),
		)
		current.fromCube()

		cell, err := start.localIjkToCell(current)
		if err != nil {
			return err
		}

		out[outOffset+outStep*n] = cell
	}

	return nil
}
