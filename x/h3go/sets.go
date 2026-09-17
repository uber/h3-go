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

// setH3Index builds a cell index from its resolution, base cell, and an initial
// digit for every digit slot up to res.
func setH3Index(res, baseCell, initDigit int) Cell {
	h := Cell(h3Init)
	h |= Cell(cellMode) << modeOffset
	h = h.setResolution(res)

	h |= Cell(baseCell) << baseCellOffset
	for r := 1; r <= res; r++ {
		h = h.setIndexDigit(r, initDigit)
	}

	return h
}

// Res0Cells returns all the cells at resolution 0 (the base cells). The error
// return is always nil; it exists to match the signature of the other cell
// enumerators.
func Res0Cells() ([]Cell, error) {
	out := make([]Cell, NumBaseCells)
	for bc := range out {
		out[bc] = setH3Index(0, bc, centerDigit)
	}

	return out, nil
}

// Pentagons returns all the pentagons at the given resolution.
func Pentagons(res int) ([]Cell, error) {
	if res < 0 || res > MaxResolution {
		return nil, ErrResolutionDomain
	}
	out := make([]Cell, 0, NumPentagons)

	for bc := range NumBaseCells {
		if isBaseCellPentagon[bc] {
			out = append(out, setH3Index(res, bc, centerDigit))
		}
	}

	return out, nil
}
