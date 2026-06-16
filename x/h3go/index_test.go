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

import "testing"

// TestIsValidCellBitPatterns covers the bit-pattern cases for cell validation:
// mode, reserved bits, high bit, an out-of-range base cell, a bad digit, and a
// pentagon deleted subsequence. They manipulate the index format directly.
func TestIsValidCellBitPatterns(t *testing.T) {
	t.Parallel()

	t.Run("mode", func(t *testing.T) {
		t.Parallel()

		for m := 0; m <= 0xf; m++ {
			h := Cell(h3Init) | Cell(m)<<modeOffset
			if got, want := isValidCell(h), m == cellMode; got != want {
				t.Fatalf("mode %d: isValidCell = %v, want %v", m, got, want)
			}
		}
	})

	t.Run("reserved_bits", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < 8; i++ {
			h := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(i)<<reservedOffset
			if got, want := isValidCell(h), i == 0; got != want {
				t.Fatalf("reserved bits %d: isValidCell = %v, want %v", i, got, want)
			}
		}
	})

	t.Run("high_bit", func(t *testing.T) {
		t.Parallel()

		var highBit Cell = 1
		highBit <<= bitSize - 1

		h := Cell(h3Init) | Cell(cellMode)<<modeOffset | highBit
		if isValidCell(h) {
			t.Fatal("isValidCell should fail when the high bit is set")
		}
	})

	t.Run("base_cell_out_of_range", func(t *testing.T) {
		t.Parallel()

		h := setH3Index(0, numBaseCells, centerDigit)
		if isValidCell(h) {
			t.Fatal("isValidCell should fail for an out-of-range base cell")
		}
	})

	t.Run("bad_digit", func(t *testing.T) {
		t.Parallel()

		// res 1 with the default all-7 digits: digit 1 is invalid.
		h := setResolution(Cell(h3Init)|Cell(cellMode)<<modeOffset, 1)
		if isValidCell(h) {
			t.Fatal("isValidCell should fail when a used digit is 7")
		}
	})

	t.Run("deleted_subsequence", func(t *testing.T) {
		t.Parallel()

		// A cell in a deleted subsequence of pentagon base cell 4.
		h := setH3Index(1, 4, kAxesDigit)
		if isValidCell(h) {
			t.Fatal("isValidCell should fail on a pentagon deleted subsequence")
		}
	})

	t.Run("more_deleted_subsequence", func(t *testing.T) {
		t.Parallel()

		for res := 1; res <= maxResolution; res++ {
			pent := setH3Index(res, 4, centerDigit) // pentagon center child
			if !isValidCell(pent) {
				t.Fatalf("res %d: pentagon center child should be valid", res)
			}

			for d := 0; d <= 6; d++ {
				h := setIndexDigit(pent, res, d)
				if got, want := isValidCell(h), d != kAxesDigit; got != want {
					t.Fatalf("res %d digit %d: isValidCell = %v, want %v", res, d, got, want)
				}
			}
		}
	})

	t.Run("unused_digit_not_7", func(t *testing.T) {
		t.Parallel()

		// A res-0 cell with an unused digit slot set to something other than 7
		// is invalid: every digit beyond the resolution must be 7.
		h := setIndexDigit(setH3Index(0, 0, centerDigit), 5, centerDigit)
		if isValidCell(h) {
			t.Fatal("isValidCell should fail when an unused digit is not 7")
		}
	})
}

// TestIndexDigitErrors covers the resolution-domain error cases for
// Cell.IndexDigit.
func TestIndexDigitErrors(t *testing.T) {
	t.Parallel()

	c, err := LatLngToCell(LatLng{Lat: 0, Lng: 0}, 9)
	if err != nil {
		t.Fatalf("LatLngToCell: %v", err)
	}

	tests := map[string]struct {
		giveRes int
	}{
		"negative_resolution": {giveRes: -1},
		"zero_resolution":     {giveRes: 0},
		"too_high_resolution": {giveRes: 16},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := c.IndexDigit(tt.giveRes); err == nil {
				t.Fatalf("IndexDigit(res=%d): expected error", tt.giveRes)
			}
		})
	}
}

// TestNumCellsKnownValues checks the documented NumCells values.
func TestNumCellsKnownValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveRes   int
		wantCount int
	}{
		"res_0":               {giveRes: 0, wantCount: 122},
		"res_1":               {giveRes: 1, wantCount: 842},
		"negative_resolution": {giveRes: -1, wantCount: 0},
		"too_high_resolution": {giveRes: 16, wantCount: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := NumCells(tt.giveRes); got != tt.wantCount {
				t.Fatalf("NumCells(%d): got %d, want %d", tt.giveRes, got, tt.wantCount)
			}
		})
	}
}

// TestInvalidPentagons checks that malformed indexes are not reported as
// pentagons.
func TestInvalidPentagons(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveCell     Cell
		wantPentagon bool
	}{
		"zero":             {giveCell: 0, wantPentagon: false},
		"all_but_high_bit": {giveCell: 0x7fffffffffffffff, wantPentagon: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := tt.giveCell.IsPentagon(); got != tt.wantPentagon {
				t.Fatalf("IsPentagon(%015x): got %v, want %v", uint64(tt.giveCell), got, tt.wantPentagon)
			}
		})
	}
}

// TestIntrospectionCorpus exercises the introspection accessors over the corpus
// and asserts self-consistent results.
func TestIntrospectionCorpus(t *testing.T) {
	t.Parallel()

	for _, c := range corpus(t) {
		if !c.IsValid() {
			t.Fatalf("corpus cell %015x is not valid", uint64(c))
		}

		res := c.Resolution()
		if res < 0 || res > maxResolution {
			t.Fatalf("cell %015x: resolution %d out of range", uint64(c), res)
		}

		if c.IsResClassIII() != (res%2 == 1) {
			t.Fatalf("cell %015x: IsResClassIII inconsistent with resolution %d", uint64(c), res)
		}

		if c.BaseCellNumber() != BaseCellNumber(c) {
			t.Fatalf("cell %015x: BaseCellNumber method and func disagree", uint64(c))
		}

		if bc := c.BaseCellNumber(); bc < 0 || bc >= numBaseCells {
			t.Fatalf("cell %015x: base cell %d out of range", uint64(c), bc)
		}

		// Exercise IsPentagon on both pentagon and non-pentagon cells.
		_ = c.IsPentagon()

		for r := 1; r <= res; r++ {
			digit, err := c.IndexDigit(r)
			if err != nil {
				t.Fatalf("IndexDigit(%015x, %d): %v", uint64(c), r, err)
			}

			if digit < 0 || digit > 6 {
				t.Fatalf("IndexDigit(%015x, %d) = %d out of range", uint64(c), r, digit)
			}
		}
	}
}
