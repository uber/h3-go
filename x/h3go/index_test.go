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
	"testing"
)

// TestIsValidCellBitPatterns covers the bit-pattern cases for cell validation:
// mode, reserved bits, high bit, an out-of-range base cell, a bad digit, and a
// pentagon deleted subsequence. They manipulate the index format directly.
func TestIsValidCellBitPatterns(t *testing.T) {
	t.Parallel()

	t.Run("mode", func(t *testing.T) {
		t.Parallel()

		for m := 0; m <= 0xf; m++ {
			h := Cell(h3Init) | Cell(m)<<modeOffset
			if got, want := h.IsValid(), m == int(cellMode); got != want {
				t.Fatalf("mode %d: isValidCell = %v, want %v", m, got, want)
			}
		}
	})

	t.Run("reserved_bits", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < 8; i++ {
			h := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(i)<<reservedOffset
			if got, want := h.IsValid(), i == 0; got != want {
				t.Fatalf("reserved bits %d: isValidCell = %v, want %v", i, got, want)
			}
		}
	})

	t.Run("high_bit", func(t *testing.T) {
		t.Parallel()

		var highBit Cell = 1
		highBit <<= bitSize - 1

		h := Cell(h3Init) | Cell(cellMode)<<modeOffset | highBit
		if h.IsValid() {
			t.Fatal("isValidCell should fail when the high bit is set")
		}
	})

	t.Run("base_cell_out_of_range", func(t *testing.T) {
		t.Parallel()

		h := setH3Index(0, NumBaseCells, centerDigit)
		if h.IsValid() {
			t.Fatal("isValidCell should fail for an out-of-range base cell")
		}
	})

	t.Run("bad_digit", func(t *testing.T) {
		t.Parallel()

		// res 1 with the default all-7 digits: digit 1 is invalid.
		h := (Cell(h3Init) | Cell(cellMode)<<modeOffset).setResolution(1)
		if h.IsValid() {
			t.Fatal("isValidCell should fail when a used digit is 7")
		}
	})

	t.Run("deleted_subsequence", func(t *testing.T) {
		t.Parallel()

		// A cell in a deleted subsequence of pentagon base cell 4.
		h := setH3Index(1, 4, kAxesDigit)
		if h.IsValid() {
			t.Fatal("isValidCell should fail on a pentagon deleted subsequence")
		}
	})

	t.Run("more_deleted_subsequence", func(t *testing.T) {
		t.Parallel()

		for res := 1; res <= MaxResolution; res++ {
			pent := setH3Index(res, 4, centerDigit) // pentagon center child
			if !pent.IsValid() {
				t.Fatalf("res %d: pentagon center child should be valid", res)
			}

			for d := 0; d <= 6; d++ {
				h := pent.setIndexDigit(res, d)
				if got, want := h.IsValid(), d != kAxesDigit; got != want {
					t.Fatalf("res %d digit %d: isValidCell = %v, want %v", res, d, got, want)
				}
			}
		}
	})

	t.Run("unused_digit_not_7", func(t *testing.T) {
		t.Parallel()

		// A res-0 cell with an unused digit slot set to something other than 7
		// is invalid: every digit beyond the resolution must be 7.
		h := setH3Index(0, 0, centerDigit).setIndexDigit(5, centerDigit)
		if h.IsValid() {
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
		if res < 0 || res > MaxResolution {
			t.Fatalf("cell %015x: resolution %d out of range", uint64(c), res)
		}

		if c.IsResClassIII() != (res%2 == 1) {
			t.Fatalf("cell %015x: IsResClassIII inconsistent with resolution %d", uint64(c), res)
		}

		if c.BaseCellNumber() != BaseCellNumber(c) {
			t.Fatalf("cell %015x: BaseCellNumber method and func disagree", uint64(c))
		}

		if bc := c.BaseCellNumber(); bc < 0 || bc >= NumBaseCells {
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

// TestConstructCell covers the domain checks, the pentagon deleted-subsequence
// rule, and round trips against Resolution, BaseCellNumber and IndexDigit.
func TestConstructCell(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveRes      int
		giveBaseCell int
		giveDigits   []int
		wantErr      error
		wantCell     Cell
	}{
		"res_below_zero":                       {giveRes: -1, giveBaseCell: 0, giveDigits: nil, wantErr: ErrResolutionDomain},
		"res_above_max":                        {giveRes: MaxResolution + 1, giveBaseCell: 0, giveDigits: make([]int, 16), wantErr: ErrResolutionDomain},
		"base_cell_negative":                   {giveRes: 1, giveBaseCell: -1, giveDigits: []int{0}, wantErr: ErrBaseCellDomain},
		"base_cell_at_count":                   {giveRes: 1, giveBaseCell: NumBaseCells, giveDigits: []int{0}, wantErr: ErrBaseCellDomain},
		"digits_shorter_than_res_pads_centers": {giveRes: 3, giveBaseCell: 0, giveDigits: []int{0, 0}, wantCell: setH3Index(3, 0, 0)},
		"digits_empty_pads_centers":            {giveRes: 2, giveBaseCell: 0, giveDigits: nil, wantCell: setH3Index(2, 0, 0)},
		"digit_negative":                       {giveRes: 1, giveBaseCell: 0, giveDigits: []int{-1}, wantErr: ErrDigitDomain},
		"digit_is_invalid":                     {giveRes: 1, giveBaseCell: 0, giveDigits: []int{invalidDigit}, wantErr: ErrDigitDomain},
		"pentagon_deleted_k_axis":              {giveRes: 1, giveBaseCell: 4, giveDigits: []int{kAxesDigit}, wantErr: ErrDeletedDigit},
		"pentagon_deleted_below_centers":       {giveRes: 3, giveBaseCell: 4, giveDigits: []int{0, 0, kAxesDigit}, wantErr: ErrDeletedDigit},
		"res_zero_ignores_nil_digits":          {giveRes: 0, giveBaseCell: 14, giveDigits: nil, wantCell: setH3Index(0, 14, 0)},
		"hexagon_all_center_digits":            {giveRes: 2, giveBaseCell: 0, giveDigits: []int{0, 0}, wantCell: setH3Index(2, 0, 0)},
		"pentagon_all_center_digits":           {giveRes: 2, giveBaseCell: 4, giveDigits: []int{0, 0}, wantCell: setH3Index(2, 4, 0)},
		"extra_digits_ignored":                 {giveRes: 1, giveBaseCell: 0, giveDigits: []int{0, 5, 5}, wantCell: setH3Index(1, 0, 0)},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ConstructCell(tt.giveRes, tt.giveBaseCell, tt.giveDigits)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ConstructCell(%d, %d, %v) error = %v, want %v", tt.giveRes, tt.giveBaseCell, tt.giveDigits, err, tt.wantErr)
			}

			if tt.wantErr == nil && got != tt.wantCell {
				t.Fatalf("ConstructCell(%d, %d, %v) = %015x, want %015x", tt.giveRes, tt.giveBaseCell, tt.giveDigits, uint64(got), uint64(tt.wantCell))
			}
		})
	}
}

// TestConstructCellResolvesPentagon checks that a non-center, non-k digit
// clears the pentagon state so a later k-axis digit is accepted.
func TestConstructCellResolvesPentagon(t *testing.T) {
	t.Parallel()

	got, err := ConstructCell(2, 4, []int{2, kAxesDigit})
	if err != nil {
		t.Fatalf("ConstructCell(2, 4, [2 1]): %v", err)
	}

	if got.IsPentagon() {
		t.Fatalf("ConstructCell(2, 4, [2 1]) = %015x, want a non-pentagon cell", uint64(got))
	}
}

// TestConstructCellRoundTripsCorpus checks every corpus cell rebuilds from its
// own resolution, base cell and digits.
func TestConstructCellRoundTripsCorpus(t *testing.T) {
	t.Parallel()

	for _, cell := range corpus(t) {
		res := cell.Resolution()

		digits := make([]int, res)
		for i := range res {
			digit, err := cell.IndexDigit(i + 1)
			if err != nil {
				t.Fatalf("IndexDigit(%015x, %d): %v", uint64(cell), i+1, err)
			}

			digits[i] = digit
		}

		got, err := ConstructCell(res, cell.BaseCellNumber(), digits)
		if err != nil {
			t.Fatalf("ConstructCell(%015x): %v", uint64(cell), err)
		}

		if got != cell {
			t.Fatalf("ConstructCell round trip = %015x, want %015x", uint64(got), uint64(cell))
		}
	}
}

// TestConstructCellShortDigitsPadCenters checks that omitting trailing digits
// is the same as passing them as center digits, for a non-zero prefix and for
// a pentagon base cell where the padding must not disturb pentagon state.
func TestConstructCellShortDigitsPadCenters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveRes      int
		giveBaseCell int
		giveShort    []int
		givePadded   []int
	}{
		"hexagon_prefix":  {giveRes: 5, giveBaseCell: 73, giveShort: []int{1, 2}, givePadded: []int{1, 2, 0, 0, 0}},
		"pentagon_prefix": {giveRes: 4, giveBaseCell: 4, giveShort: []int{0, 2}, givePadded: []int{0, 2, 0, 0}},
		"pentagon_empty":  {giveRes: 3, giveBaseCell: 4, giveShort: nil, givePadded: []int{0, 0, 0}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			short, err := ConstructCell(tt.giveRes, tt.giveBaseCell, tt.giveShort)
			if err != nil {
				t.Fatalf("ConstructCell(%d, %d, %v): %v", tt.giveRes, tt.giveBaseCell, tt.giveShort, err)
			}

			padded, err := ConstructCell(tt.giveRes, tt.giveBaseCell, tt.givePadded)
			if err != nil {
				t.Fatalf("ConstructCell(%d, %d, %v): %v", tt.giveRes, tt.giveBaseCell, tt.givePadded, err)
			}

			if short != padded {
				t.Fatalf("ConstructCell(%d, %d, %v) = %015x, want %015x from the padded form", tt.giveRes, tt.giveBaseCell, tt.giveShort, uint64(short), uint64(padded))
			}
		})
	}
}
