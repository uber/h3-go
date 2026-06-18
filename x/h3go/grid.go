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
	"errors"
)

// nextRingDirection is the direction stepped to move from one ring to the next
// larger ring before tracing it.
const nextRingDirection = iAxesDigit

var (
	// directions is the ordered set of the six neighbor directions used when
	// tracing a ring counterclockwise, one side of the ring per entry.
	directions = [6]int{
		jAxesDigit, jkAxesDigit, kAxesDigit, ikAxesDigit, iAxesDigit, ijAxesDigit,
	}

	// newDigitII maps current digit and direction to the new digit on a Class II grid.
	newDigitII = [7][7]int{
		{centerDigit, kAxesDigit, jAxesDigit, jkAxesDigit, iAxesDigit,
			ikAxesDigit, ijAxesDigit},
		{kAxesDigit, iAxesDigit, jkAxesDigit, ijAxesDigit, ikAxesDigit,
			jAxesDigit, centerDigit},
		{jAxesDigit, jkAxesDigit, kAxesDigit, iAxesDigit, ijAxesDigit,
			centerDigit, ikAxesDigit},
		{jkAxesDigit, ijAxesDigit, iAxesDigit, ikAxesDigit, centerDigit,
			kAxesDigit, jAxesDigit},
		{iAxesDigit, ikAxesDigit, ijAxesDigit, centerDigit, jAxesDigit,
			jkAxesDigit, kAxesDigit},
		{ikAxesDigit, jAxesDigit, centerDigit, kAxesDigit, jkAxesDigit,
			ijAxesDigit, iAxesDigit},
		{ijAxesDigit, centerDigit, ikAxesDigit, jAxesDigit, kAxesDigit,
			iAxesDigit, jkAxesDigit}}

	// newAdjustmentII maps current digit and direction to the coarser ap7 move on a Class II grid.
	newAdjustmentII = [7][7]int{
		{centerDigit, centerDigit, centerDigit, centerDigit, centerDigit,
			centerDigit, centerDigit},
		{centerDigit, kAxesDigit, centerDigit, kAxesDigit, centerDigit,
			ikAxesDigit, centerDigit},
		{centerDigit, centerDigit, jAxesDigit, jkAxesDigit, centerDigit,
			centerDigit, jAxesDigit},
		{centerDigit, kAxesDigit, jkAxesDigit, jkAxesDigit, centerDigit,
			centerDigit, centerDigit},
		{centerDigit, centerDigit, centerDigit, centerDigit, iAxesDigit,
			iAxesDigit, ijAxesDigit},
		{centerDigit, ikAxesDigit, centerDigit, centerDigit, iAxesDigit,
			ikAxesDigit, centerDigit},
		{centerDigit, centerDigit, jAxesDigit, centerDigit, ijAxesDigit,
			centerDigit, ijAxesDigit}}

	// newDigitIII maps current digit and direction to the new digit on a Class III grid.
	newDigitIII = [7][7]int{
		{centerDigit, kAxesDigit, jAxesDigit, jkAxesDigit, iAxesDigit,
			ikAxesDigit, ijAxesDigit},
		{kAxesDigit, jAxesDigit, jkAxesDigit, iAxesDigit, ikAxesDigit,
			ijAxesDigit, centerDigit},
		{jAxesDigit, jkAxesDigit, iAxesDigit, ikAxesDigit, ijAxesDigit,
			centerDigit, kAxesDigit},
		{jkAxesDigit, iAxesDigit, ikAxesDigit, ijAxesDigit, centerDigit,
			kAxesDigit, jAxesDigit},
		{iAxesDigit, ikAxesDigit, ijAxesDigit, centerDigit, kAxesDigit,
			jAxesDigit, jkAxesDigit},
		{ikAxesDigit, ijAxesDigit, centerDigit, kAxesDigit, jAxesDigit,
			jkAxesDigit, iAxesDigit},
		{ijAxesDigit, centerDigit, kAxesDigit, jAxesDigit, jkAxesDigit,
			iAxesDigit, ikAxesDigit}}

	// newAdjustmentIII maps current digit and direction to the coarser ap7 move on a Class III grid.
	newAdjustmentIII = [7][7]int{
		{centerDigit, centerDigit, centerDigit, centerDigit, centerDigit,
			centerDigit, centerDigit},
		{centerDigit, kAxesDigit, centerDigit, jkAxesDigit, centerDigit,
			kAxesDigit, centerDigit},
		{centerDigit, centerDigit, jAxesDigit, jAxesDigit, centerDigit,
			centerDigit, ijAxesDigit},
		{centerDigit, jkAxesDigit, jAxesDigit, jkAxesDigit, centerDigit,
			centerDigit, centerDigit},
		{centerDigit, centerDigit, centerDigit, centerDigit, iAxesDigit,
			ikAxesDigit, iAxesDigit},
		{centerDigit, kAxesDigit, centerDigit, centerDigit, ikAxesDigit,
			ikAxesDigit, centerDigit},
		{centerDigit, centerDigit, ijAxesDigit, centerDigit, iAxesDigit,
			centerDigit, ijAxesDigit}}
)

// isBaseCellPolarPentagon reports whether a base cell is one of the two polar
// pentagons, which have all-i neighbors and therefore distort differently.
func isBaseCellPolarPentagon(baseCell int) bool {
	return baseCell == 4 || baseCell == 117
}

// baseCellIsCwOffset reports whether testFace is one of the clockwise-offset
// faces of a pentagon base cell.
func baseCellIsCwOffset(baseCell, testFace int) bool {
	offsets, ok := baseCellCWOffsetPent[baseCell]

	return ok && (offsets[0] == testFace || offsets[1] == testFace)
}

// neighborRotations returns the cell adjacent to c in direction dir, applying
// rotations counterclockwise 60° turns to dir first. It also returns the number
// of rotations a caller should carry into the next step, which changes whenever
// the move crosses an icosahedron face edge or pentagon distortion. It fails
// with ErrPentagon when the move enters a pentagon's deleted k subsequence in an
// undefined way.
func (c Cell) neighborRotations(dir, rotations int) (Cell, int, error) {
	current := c

	if dir < centerDigit || dir >= invalidDigit {
		return 0, 0, ErrFailed
	}

	rotations %= 6
	for range rotations {
		dir = rotate60ccw(dir)
	}

	newRotations := 0

	oldBaseCell := current.BaseCellNumber()
	if oldBaseCell < 0 || oldBaseCell >= NumBaseCells {
		return 0, 0, ErrCellInvalid
	}

	oldLeadingDigit := current.leadingNonZeroDigit()

	res := current.Resolution() - 1
	for {
		if res == -1 {
			current = current.setBaseCell(baseCellNeighbors[oldBaseCell][dir])
			newRotations = baseCellNeighbor60CCWRots[oldBaseCell][dir]

			if current.BaseCellNumber() == invalidBaseCell {
				// Adjust for the deleted k vertex at the base cell level. This
				// edge actually borders a different neighbor.
				current = current.setBaseCell(baseCellNeighbors[oldBaseCell][ikAxesDigit])
				newRotations = baseCellNeighbor60CCWRots[oldBaseCell][ikAxesDigit]

				// Perform the adjustment for the k-subsequence we're skipping.
				current = current.rotate60ccw()
				rotations++
			}

			break
		}

		oldDigit := indexDigit(current, res+1)
		if oldDigit == invalidDigit {
			return 0, 0, ErrCellInvalid
		}

		var nextDir int

		if isResClassIII(res + 1) {
			current = current.setIndexDigit(res+1, newDigitII[oldDigit][dir])
			nextDir = newAdjustmentII[oldDigit][dir]
		} else {
			current = current.setIndexDigit(res+1, newDigitIII[oldDigit][dir])
			nextDir = newAdjustmentIII[oldDigit][dir]
		}

		if nextDir == centerDigit {
			break
		}

		dir = nextDir
		res--
	}

	newBaseCell := current.BaseCellNumber()
	if isBaseCellPentagon[newBaseCell] {
		alreadyAdjustedKSubsequence := false

		// Force rotation out of the missing k-axes sub-sequence.
		if current.leadingNonZeroDigit() == kAxesDigit {
			if oldBaseCell != newBaseCell {
				// We traversed into the deleted k subsequence of a different
				// pentagon base cell. The shared edge is always that base cell's
				// clockwise-offset face, so the rotation is clockwise.
				if baseCellIsCwOffset(newBaseCell, baseCellHomeFijk[oldBaseCell].face) {
					current = current.rotate60cw()
				}

				alreadyAdjustedKSubsequence = true
			} else {
				// We traversed into the deleted k subsequence from within the
				// same pentagon base cell.
				switch oldLeadingDigit {
				case centerDigit:
					// Undefined: the k direction is deleted from here.
					return 0, 0, ErrPentagon
				case jkAxesDigit:
					current = current.rotate60ccw()
					rotations++
				case ikAxesDigit:
					current = current.rotate60cw()
					rotations += 5
				default:
					return 0, 0, ErrFailed
				}
			}
		}

		for range newRotations {
			current = current.rotatePent60ccw()
		}

		// Account for differing orientation of the base cells.
		if oldBaseCell != newBaseCell {
			if isBaseCellPolarPentagon(newBaseCell) {
				// Polar base cells behave differently because they have all i
				// neighbors.
				if oldBaseCell != 118 && oldBaseCell != 8 &&
					current.leadingNonZeroDigit() != jkAxesDigit {
					rotations++
				}
			} else if current.leadingNonZeroDigit() == ikAxesDigit && !alreadyAdjustedKSubsequence {
				// Account for distortion introduced to the 5 neighbor by the
				// deleted k subsequence.
				rotations++
			}
		}
	} else {
		for range newRotations {
			current = current.rotate60ccw()
		}
	}

	rotations = (rotations + newRotations) % 6

	return current, rotations, nil
}

// GridDisk returns the cells within grid distance k of the origin cell. The
// origin is included. Output ordering is not significant. It falls back to the
// safe traversal when the fast one meets pentagon distortion.
func (c Cell) GridDisk(k int) ([]Cell, error) {
	rings, err := c.GridDiskDistances(k)
	if err != nil {
		return nil, err
	}

	var out []Cell
	for _, ring := range rings {
		out = append(out, ring...)
	}

	return out, nil
}

// GridDisk returns the cells within grid distance k of the origin cell.
func GridDisk(origin Cell, k int) ([]Cell, error) {
	return origin.GridDisk(k)
}

// GridDisksUnsafe returns, for each origin, the cells within grid distance k of
// that origin. The outer slice matches the order of origins; inner ordering is
// not significant. It fails if any disk encounters pentagon distortion.
func GridDisksUnsafe(origins []Cell, k int) ([][]Cell, error) {
	if len(origins) == 0 {
		return nil, nil
	}

	out := make([][]Cell, len(origins))
	for i, origin := range origins {
		rings, err := origin.GridDiskDistancesUnsafe(k)
		if err != nil {
			return nil, err
		}

		var flat []Cell
		for _, ring := range rings {
			flat = append(flat, ring...)
		}

		out[i] = flat
	}

	return out, nil
}

// GridDiskDistances returns the cells within grid distance k of the origin,
// grouped by ring: index 0 is the origin, index d holds the cells at grid
// distance d. It optimistically tries the fast traversal, falling back to the
// safe one on pentagon distortion.
func (c Cell) GridDiskDistances(k int) ([][]Cell, error) {
	rings, err := c.GridDiskDistancesUnsafe(k)
	if err != nil {
		return c.GridDiskDistancesSafe(k)
	}

	return rings, nil
}

// GridDiskDistances returns the cells within grid distance k of the origin,
// grouped by ring.
func GridDiskDistances(origin Cell, k int) ([][]Cell, error) {
	return origin.GridDiskDistances(k)
}

// GridDiskDistancesUnsafe returns the cells within grid distance k of the
// origin, grouped by ring, using a fast spiral traversal. It fails with
// ErrPentagon if the spiral meets a pentagon or pentagon distortion.
func (c Cell) GridDiskDistancesUnsafe(k int) ([][]Cell, error) {
	if k < 0 {
		return nil, ErrDomain
	}

	rings := make([][]Cell, k+1)

	cursor := c
	rings[0] = append(rings[0], cursor)

	if cursor.IsPentagon() {
		return nil, ErrPentagon
	}

	ring := 1
	direction := 0
	i := 0
	rotations := 0

	for ring <= k {
		if direction == 0 && i == 0 {
			next, rot, err := cursor.neighborRotations(nextRingDirection, rotations)
			if err != nil {
				return nil, err
			}

			cursor, rotations = next, rot
			if cursor.IsPentagon() {
				return nil, ErrPentagon
			}
		}

		next, rot, err := cursor.neighborRotations(directions[direction], rotations)
		if err != nil {
			return nil, err
		}

		cursor, rotations = next, rot
		rings[ring] = append(rings[ring], cursor)

		i++
		if i == ring {
			i = 0
			direction++

			if direction == 6 {
				direction = 0
				ring++
			}
		}

		if cursor.IsPentagon() {
			return nil, ErrPentagon
		}
	}

	return rings, nil
}

// GridDiskDistancesUnsafe returns the cells within grid distance k of the
// origin, grouped by ring, using a fast spiral traversal.
func GridDiskDistancesUnsafe(origin Cell, k int) ([][]Cell, error) {
	return origin.GridDiskDistancesUnsafe(k)
}

// GridDiskDistancesSafe returns the cells within grid distance k of the origin,
// grouped by ring, using a breadth-first traversal that tolerates pentagon
// distortion. It is slower than the unsafe variant but always correct.
func (c Cell) GridDiskDistancesSafe(k int) ([][]Cell, error) {
	if k < 0 {
		return nil, ErrDomain
	}

	rings := make([][]Cell, k+1)
	seen := map[Cell]int{c: 0}
	rings[0] = append(rings[0], c)

	type queued struct {
		cell Cell
		dist int
	}

	queue := []queued{{c, 0}}
	for len(queue) > 0 {
		head := queue[0]
		queue = queue[1:]

		if head.dist >= k {
			continue
		}

		for _, dir := range directions {
			neighbor, _, err := head.cell.neighborRotations(dir, 0)
			if err != nil {
				if errors.Is(err, ErrPentagon) {
					continue
				}

				return nil, err
			}

			dist := head.dist + 1
			if prev, ok := seen[neighbor]; ok && prev <= dist {
				continue
			}

			seen[neighbor] = dist
			rings[dist] = append(rings[dist], neighbor)
			queue = append(queue, queued{neighbor, dist})
		}
	}

	return rings, nil
}

// GridDiskDistancesSafe returns the cells within grid distance k of the origin,
// grouped by ring, using a breadth-first traversal that tolerates pentagon
// distortion.
func GridDiskDistancesSafe(origin Cell, k int) ([][]Cell, error) {
	return origin.GridDiskDistancesSafe(k)
}

// GridRingUnsafe returns the hollow ring of cells at exactly grid distance k
// from the origin, using a fast single-loop traversal. k of 0 returns just the
// origin. It fails with ErrPentagon if the ring meets a pentagon or pentagon
// distortion.
func (c Cell) GridRingUnsafe(k int) ([]Cell, error) {
	if k < 0 {
		return nil, ErrDomain
	}

	if k == 0 {
		return []Cell{c}, nil
	}

	cursor := c
	rotations := 0

	if cursor.IsPentagon() {
		return nil, ErrPentagon
	}

	for range k {
		next, rot, err := cursor.neighborRotations(nextRingDirection, rotations)
		if err != nil {
			return nil, err
		}

		cursor, rotations = next, rot
		if cursor.IsPentagon() {
			return nil, ErrPentagon
		}
	}

	lastIndex := cursor
	out := []Cell{cursor}

	for direction := 0; direction < 6; direction++ {
		for pos := 0; pos < k; pos++ {
			next, rot, err := cursor.neighborRotations(directions[direction], rotations)
			if err != nil {
				return nil, err
			}

			cursor, rotations = next, rot

			// Skip the very last index; it equals the first one already added.
			// We still traverse to it for the pentagon-distortion check below.
			if pos != k-1 || direction != 5 {
				out = append(out, cursor)

				if cursor.IsPentagon() {
					return nil, ErrPentagon
				}
			}
		}
	}

	// If the loop didn't return to its start, pentagon distortion occurred.
	if lastIndex != cursor {
		return nil, ErrPentagon
	}

	return out, nil
}

// GridRingUnsafe returns the hollow ring of cells at exactly grid distance k
// from the origin, using a fast single-loop traversal.
func GridRingUnsafe(origin Cell, k int) ([]Cell, error) {
	return origin.GridRingUnsafe(k)
}

// GridRing returns the hollow ring of cells at exactly grid distance k from the
// origin. It tries the fast traversal first and falls back to the safe disk
// traversal when pentagon distortion is encountered.
func (c Cell) GridRing(k int) ([]Cell, error) {
	out, err := c.GridRingUnsafe(k)
	if err == nil {
		return out, nil
	}

	if k < 0 {
		return nil, ErrDomain
	}

	rings, err := c.GridDiskDistancesSafe(k)
	if err != nil {
		return nil, err
	}

	return rings[k], nil
}

// GridRing returns the hollow ring of cells at exactly grid distance k from the
// origin.
func GridRing(origin Cell, k int) ([]Cell, error) {
	return origin.GridRing(k)
}

// IsNeighbor reports whether c and other are adjacent cells at the same
// resolution.
func (c Cell) IsNeighbor(other Cell) (bool, error) {
	if modeOf(c) != cellMode || modeOf(other) != cellMode {
		return false, ErrCellInvalid
	}

	// A cell is never its own neighbor.
	if c == other {
		return false, nil
	}

	if c.Resolution() != other.Resolution() {
		return false, ErrResolutionMismatch
	}

	// Cells that share a parent are very likely neighbors; a digit lookup is a
	// cheap way to decide many cases before the general traversal.
	parentRes := c.Resolution() - 1
	if parentRes > 0 {
		originParent, originErr := c.Parent(parentRes)
		destParent, destErr := other.Parent(parentRes)

		if originErr == nil && destErr == nil && originParent == destParent {
			originResDigit := indexDigit(c, parentRes+1)
			destResDigit := indexDigit(other, parentRes+1)

			if originResDigit == centerDigit || destResDigit == centerDigit {
				return true, nil
			}

			if originResDigit >= invalidDigit {
				return false, ErrCellInvalid
			}

			if (originResDigit == kAxesDigit || destResDigit == kAxesDigit) &&
				originParent.IsPentagon() {
				// Real neighbors across a pentagon's deleted subsequence fail
				// this optimized check but are accepted by the disk check below.
				return false, ErrCellInvalid
			}

			// neighborDigits maps an indexing digit to the two adjacent digits
			// (clockwise, counterclockwise) of the cells that share this parent
			// and neighbor the origin.
			neighborDigits := [7][2]int{
				{centerDigit, centerDigit},
				{jkAxesDigit, ikAxesDigit},
				{ijAxesDigit, jkAxesDigit},
				{jAxesDigit, kAxesDigit},
				{ikAxesDigit, ijAxesDigit},
				{kAxesDigit, iAxesDigit},
				{iAxesDigit, jAxesDigit},
			}

			//nolint:gosec // originResDigit is in [1,6] here (center and out-of-range digits are handled above), so this fixed-size lookup stays in bounds.
			adjacent := neighborDigits[originResDigit]
			if adjacent[0] == destResDigit || adjacent[1] == destResDigit {
				return true, nil
			}
		}
	}

	// Otherwise determine the relationship the hard way.
	ring, err := c.GridDisk(1)
	if err != nil {
		return false, err
	}

	for _, cell := range ring {
		if cell == other {
			return true, nil
		}
	}

	return false, nil
}

// directionForNeighbor returns the digit direction from origin to destination,
// the reverse of stepping a neighbor. It returns invalidDigit when the cells are
// not neighbors. It checks each neighbor in turn, skipping the center (which is
// the origin) and the deleted k axis for pentagons.
func (c Cell) directionForNeighbor(destination Cell) int {
	firstDir := kAxesDigit
	if c.IsPentagon() {
		firstDir = jAxesDigit
	}

	for dir := firstDir; dir < numDigits; dir++ {
		neighbor, _, err := c.neighborRotations(dir, 0)
		if err == nil && neighbor == destination {
			return dir
		}
	}

	return invalidDigit
}

// baseCellToCCWrot60 returns the number of 60° counterclockwise rotations for a
// base cell's coordinate system on the given face, or invalidRotations if the
// base cell does not appear on that face.
func baseCellToCCWrot60(baseCell, face int) int {
	if face < 0 || face >= NumIcosaFaces {
		return invalidRotations
	}

	for i := range 3 {
		for j := range 3 {
			for k := range 3 {
				if faceIjkBaseCells[face][i][j][k].baseCell == baseCell {
					return faceIjkBaseCells[face][i][j][k].ccwRot60
				}
			}
		}
	}

	return invalidRotations
}
