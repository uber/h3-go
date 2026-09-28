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
	// flatTableLoadFactor is the denominator of the maximum fill ratio for
	// flatTable: the table doubles once it is more than half full so linear
	// probing stays short.
	flatTableLoadFactor = 2

	// flatTableMinSize is the table size a zero flatTable allocates on first
	// insert.
	flatTableMinSize = 64
)

// flatEntry is one slot of a flatTable: a key and its value, stored inline so
// the whole table is a single allocation. The value comes first because Go pads
// a trailing zero-size field, which would double the entry size of a set.
type flatEntry[K Index, V any] struct {
	value V
	key   K
}

// flatTable is an open-addressing hash table with linear probing, keyed by an
// H3 index. It replaces Go maps on hot paths: one flat allocation, no per-entry
// overhead, and no hashing beyond a modulo, which is enough because the low
// bits of an H3 index carry the finest resolution digits. The zero flatTable is
// ready to use and allocates on first insert.
type flatTable[K Index, V any] struct {
	entries []flatEntry[K, V]
	count   int
}

// newFlatTable returns a table sized to hold expected entries without growing.
func newFlatTable[K Index, V any](expected int) flatTable[K, V] {
	size := max(expected*flatTableLoadFactor, flatTableMinSize)

	return flatTable[K, V]{entries: make([]flatEntry[K, V], size)}
}

// hashSlot returns the starting probe slot for key in an open-addressing table
// of the given size.
func hashSlot[K Index](key K, size int) int {
	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	return int(uint64(key) % uint64(size))
}

// find returns the slot holding key, or the empty slot where key would go. The
// table must be non-empty.
func (t *flatTable[K, V]) find(key K) int {
	slot := hashSlot(key, len(t.entries))
	for t.entries[slot].key != 0 && t.entries[slot].key != key {
		slot++
		if slot == len(t.entries) {
			slot = 0
		}
	}

	return slot
}

// contains reports whether key is in the table.
func (t *flatTable[K, V]) contains(key K) bool {
	if len(t.entries) == 0 {
		return false
	}

	return t.entries[t.find(key)].key == key
}

// lookup returns the value stored for key and whether key is present.
func (t *flatTable[K, V]) lookup(key K) (V, bool) {
	if len(t.entries) == 0 {
		var zero V

		return zero, false
	}

	entry := &t.entries[t.find(key)]

	return entry.value, entry.key == key
}

// insert stores value under key and reports whether key was newly added. An
// existing key keeps its original value.
func (t *flatTable[K, V]) insert(key K, value V) bool {
	if len(t.entries) == 0 {
		t.entries = make([]flatEntry[K, V], flatTableMinSize)
	}

	slot := t.find(key)
	if t.entries[slot].key == key {
		return false
	}

	t.entries[slot] = flatEntry[K, V]{key: key, value: value}
	t.count++

	if t.count*flatTableLoadFactor > len(t.entries) {
		t.grow()
	}

	return true
}

// grow doubles the table and reinserts every entry.
func (t *flatTable[K, V]) grow() {
	old := t.entries
	t.entries = make([]flatEntry[K, V], 2*len(old))

	for _, entry := range old {
		if entry.key != 0 {
			t.entries[t.find(entry.key)] = entry
		}
	}
}

// cellSet is a flatTable used as a set of cells. The wrapper hides the empty
// value so call sites read as a set.
type cellSet struct {
	table flatTable[Cell, struct{}]
}

// newCellSet returns a set sized to hold expected cells without growing.
func newCellSet(expected int) cellSet {
	return cellSet{table: newFlatTable[Cell, struct{}](expected)}
}

// contains reports whether c is in the set.
func (s *cellSet) contains(c Cell) bool {
	return s.table.contains(c)
}

// insert adds c and reports whether it was newly added.
func (s *cellSet) insert(c Cell) bool {
	return s.table.insert(c, struct{}{})
}
