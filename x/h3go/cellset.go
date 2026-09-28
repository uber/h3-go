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

const (
	// cellSetLoadFactor is the denominator of the maximum fill ratio for
	// cellSet: the table doubles once it is more than half full so linear
	// probing stays short.
	cellSetLoadFactor = 2

	// cellSetMinSize is the table size a zero cellSet allocates on first insert.
	cellSetMinSize = 64
)

// cellSet is an open-addressing hash set of cells with linear probing. It
// replaces map[Cell]bool on hot paths: one flat allocation, no per-entry
// overhead, and no hashing beyond a modulo. The zero Cell marks an empty slot,
// which is safe because a zero index is never a valid cell. The zero cellSet is
// ready to use and allocates on first insert.
type cellSet struct {
	slots []Cell
	count int
}

// newCellSet returns a set sized to hold expected cells without growing.
func newCellSet(expected int) cellSet {
	size := max(expected*cellSetLoadFactor, cellSetMinSize)

	return cellSet{slots: make([]Cell, size)}
}

// hashSlot returns the starting probe slot for c in an open-addressing set of
// the given size.
func hashSlot(c Cell, size int) int {
	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	return int(uint64(c) % uint64(size))
}

// find returns the slot holding c, or the empty slot where c would go. The set
// must be non-empty.
func (s *cellSet) find(c Cell) int {
	slot := hashSlot(c, len(s.slots))
	for s.slots[slot] != 0 && s.slots[slot] != c {
		slot++
		if slot == len(s.slots) {
			slot = 0
		}
	}

	return slot
}

// contains reports whether c is in the set.
func (s *cellSet) contains(c Cell) bool {
	if len(s.slots) == 0 {
		return false
	}

	return s.slots[s.find(c)] == c
}

// insert adds c and reports whether it was newly added.
func (s *cellSet) insert(c Cell) bool {
	if len(s.slots) == 0 {
		s.slots = make([]Cell, cellSetMinSize)
	}

	slot := s.find(c)
	if s.slots[slot] == c {
		return false
	}

	s.slots[slot] = c
	s.count++

	if s.count*cellSetLoadFactor > len(s.slots) {
		s.grow()
	}

	return true
}

// grow doubles the table and reinserts every cell.
func (s *cellSet) grow() {
	old := s.slots
	s.slots = make([]Cell, 2*len(old))

	for _, c := range old {
		if c != 0 {
			s.slots[s.find(c)] = c
		}
	}
}
