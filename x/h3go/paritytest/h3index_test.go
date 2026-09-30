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

package paritytest

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// TestIntrospectionMatchesCgo sweeps the introspection helpers over the
// reference corpus and asserts the pure-Go output matches the cgo reference.
func TestIntrospectionMatchesCgo(t *testing.T) {
	t.Parallel()
	corpus := referenceCorpus(t)

	t.Run("Resolution", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).Resolution(), ref.Resolution(); got != want {
				t.Fatalf("Resolution(%015x): got %d, want %d", uint64(ref), got, want)
			}
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).IsValid(), ref.IsValid(); got != want {
				t.Fatalf("IsValid(%015x): got %v, want %v", uint64(ref), got, want)
			}
		}
	})

	t.Run("IsPentagon", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).IsPentagon(), ref.IsPentagon(); got != want {
				t.Fatalf("IsPentagon(%015x): got %v, want %v", uint64(ref), got, want)
			}
		}
	})

	t.Run("IsResClassIII", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			if got, want := h3goCell(ref).IsResClassIII(), ref.IsResClassIII(); got != want {
				t.Fatalf("IsResClassIII(%015x): got %v, want %v", uint64(ref), got, want)
			}
		}
	})

	t.Run("BaseCellNumber", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			c := h3goCell(ref)
			if got, want := c.BaseCellNumber(), ref.BaseCellNumber(); got != want {
				t.Fatalf("BaseCellNumber(%015x): got %d, want %d", uint64(ref), got, want)
			}

			if got, want := h3go.BaseCellNumber(c), h3.BaseCellNumber(ref); got != want {
				t.Fatalf("BaseCellNumber func(%015x): got %d, want %d", uint64(ref), got, want)
			}
		}
	})

	t.Run("IndexDigit", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			assertIndexDigitsMatch(t, h3goCell(ref), ref)
		}
	})
}

func assertIndexDigitsMatch(t *testing.T, c h3go.Cell, ref h3.Cell) {
	t.Helper()

	for res := 1; res <= h3.MaxResolution; res++ {
		got, gErr := c.IndexDigit(res)
		want, wErr := ref.IndexDigit(res)

		if !bothErr(gErr, wErr) {
			t.Fatalf("IndexDigit(%015x, %d): err got=%v want=%v", uint64(ref), res, gErr, wErr)
		}

		if wErr == nil && got != want {
			t.Fatalf("IndexDigit(%015x, %d): got %d, want %d", uint64(ref), res, got, want)
		}
	}
}

// TestNumCellsMatchesCgo checks NumCells across all resolutions against the cgo
// reference.
func TestNumCellsMatchesCgo(t *testing.T) {
	t.Parallel()

	for res := 0; res <= h3.MaxResolution; res++ {
		if got, want := h3go.NumCells(res), h3.NumCells(res); got != want {
			t.Fatalf("NumCells(%d): got %d, want %d", res, got, want)
		}
	}
}

// TestConstructCellMatchesCgo checks ConstructCell against the cgo reference:
// that every corpus cell round trips through its own digits, that the domain
// and deleted-digit errors agree, and that a seeded sweep of digit sequences
// agrees on both the cell and the error.
func TestConstructCellMatchesCgo(t *testing.T) {
	t.Parallel()

	corpus := referenceCorpus(t)

	t.Run("round_trips_corpus_cells", func(t *testing.T) {
		t.Parallel()

		for _, ref := range corpus {
			res := ref.Resolution()

			digits := make([]int, res)
			for i := range res {
				digit, err := ref.IndexDigit(i + 1)
				if err != nil {
					t.Fatalf("IndexDigit(%015x, %d): %v", uint64(ref), i+1, err)
				}

				digits[i] = digit
			}

			assertSameConstructed(t, res, ref.BaseCellNumber(), digits)
		}
	})

	t.Run("domain_and_deleted_digit_errors", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			giveRes      int
			giveBaseCell int
			giveDigits   []int
		}{
			"res_below_zero":        {giveRes: -1, giveBaseCell: 0, giveDigits: nil},
			"res_above_max":         {giveRes: h3.MaxResolution + 1, giveBaseCell: 0, giveDigits: make([]int, 16)},
			"base_cell_negative":    {giveRes: 1, giveBaseCell: -1, giveDigits: []int{0}},
			"base_cell_too_large":   {giveRes: 1, giveBaseCell: 122, giveDigits: []int{0}},
			"digit_negative":        {giveRes: 1, giveBaseCell: 0, giveDigits: []int{-1}},
			"digit_is_seven":        {giveRes: 1, giveBaseCell: 0, giveDigits: []int{7}},
			"deleted_subsequence":   {giveRes: 1, giveBaseCell: 4, giveDigits: []int{1}},
			"deleted_below_pent":    {giveRes: 2, giveBaseCell: 4, giveDigits: []int{0, 1}},
			"pentagon_resolved":     {giveRes: 2, giveBaseCell: 4, giveDigits: []int{2, 1}},
			"res_zero_no_digits":    {giveRes: 0, giveBaseCell: 14, giveDigits: nil},
			"short_digits_hexagon":  {giveRes: 5, giveBaseCell: 73, giveDigits: []int{1, 2}},
			"short_digits_pentagon": {giveRes: 4, giveBaseCell: 4, giveDigits: []int{0, 2}},
			"empty_digits_at_res_3": {giveRes: 3, giveBaseCell: 4, giveDigits: nil},
			"extra_digits_ignored":  {giveRes: 1, giveBaseCell: 0, giveDigits: []int{0, 5, 5}},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				assertSameConstructed(t, tt.giveRes, tt.giveBaseCell, tt.giveDigits)
			})
		}
	})

	t.Run("seeded_digit_sweep", func(t *testing.T) {
		t.Parallel()

		const (
			seed       = 1
			iterations = 20000
		)

		rng := rand.New(rand.NewSource(seed))

		for range iterations {
			res := rng.Intn(h3.MaxResolution + 1)
			baseCell := rng.Intn(h3.NumBaseCells)

			// Vary the slice length around res so the padding rule is
			// exercised in both directions, not just the exact-length case.
			length := rng.Intn(h3.MaxResolution + 1)

			digits := make([]int, length)
			for i := range digits {
				// Reach past the valid range so the digit-domain branch is hit.
				digits[i] = rng.Intn(8)
			}

			assertSameConstructed(t, res, baseCell, digits)
		}
	})
}

// assertSameConstructed checks that both implementations agree on the cell and
// the error produced for one (resolution, base cell, digits) triple.
func assertSameConstructed(t *testing.T, res, baseCell int, digits []int) {
	t.Helper()

	want, wErr := h3.ConstructCell(res, baseCell, digits)
	got, gErr := h3go.ConstructCell(res, baseCell, digits)

	if !bothErr(wErr, gErr) {
		t.Fatalf("ConstructCell(%d, %d, %v): err cgo=%v h3go=%v", res, baseCell, digits, wErr, gErr)
	}

	if wErr == nil && uint64(got) != uint64(want) {
		t.Fatalf("ConstructCell(%d, %d, %v): cgo=%015x h3go=%015x", res, baseCell, digits, uint64(want), uint64(got))
	}
}

// upstreamDigits returns the given digits in a 15-length zero-padded slice,
// matching the fixed `int digits[15]` member of the test-case struct in the C
// suite. Entries past the resolution under test are never read, and padding
// them is what lets the table below be transcribed unchanged.
func upstreamDigits(digits ...int) []int {
	padded := make([]int, h3.MaxResolution)
	copy(padded, digits)

	return padded
}

// TestConstructCellUpstreamTable replays the test table from the C suite's
// testConstructCell.c, asserting both implementations produce the cell or the
// error that upstream declares. It transcribes the C cases verbatim, including
// the zero padding, so drift from the reference contract shows up here.
func TestConstructCellUpstreamTable(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveRes      int
		giveBaseCell int
		giveDigits   []int
		wantCell     uint64
		wantErr      error // expected cgo (h3) error
		wantGoErr    error // expected pure-Go (h3go) error
	}{
		// Valid constructions.
		"res0_base_cell_0":   {giveRes: 0, giveBaseCell: 0, giveDigits: upstreamDigits(), wantCell: 0x8001fffffffffff},
		"res0_base_cell_1":   {giveRes: 0, giveBaseCell: 1, giveDigits: upstreamDigits(), wantCell: 0x8003fffffffffff},
		"res0_base_cell_121": {giveRes: 0, giveBaseCell: 121, giveDigits: upstreamDigits(), wantCell: 0x80f3fffffffffff},
		"res3_base_cell_73":  {giveRes: 3, giveBaseCell: 73, giveDigits: upstreamDigits(1, 2, 3), wantCell: 0x839253fffffffff},
		"res2_base_cell_15":  {giveRes: 2, giveBaseCell: 15, giveDigits: upstreamDigits(5, 4), wantCell: 0x821f67fffffffff},
		"res1_base_cell_42":  {giveRes: 1, giveBaseCell: 42, giveDigits: upstreamDigits(6), wantCell: 0x8155bffffffffff},
		"res15_base_cell_58": {giveRes: 15, giveBaseCell: 58, giveDigits: upstreamDigits(5, 1, 6, 3, 1, 1, 1, 4, 4, 5, 5, 3, 3, 3, 0), wantCell: 0x8f754e64992d6d8},

		// Resolution domain.
		"res_16":           {giveRes: 16, giveBaseCell: 0, giveDigits: upstreamDigits(), wantErr: h3.ErrResolutionDomain, wantGoErr: h3go.ErrResolutionDomain},
		"res_18":           {giveRes: 18, giveBaseCell: 0, giveDigits: upstreamDigits(), wantErr: h3.ErrResolutionDomain, wantGoErr: h3go.ErrResolutionDomain},
		"res_negative_one": {giveRes: -1, giveBaseCell: 0, giveDigits: upstreamDigits(), wantErr: h3.ErrResolutionDomain, wantGoErr: h3go.ErrResolutionDomain},

		// Base cell domain.
		"base_cell_122":          {giveRes: 0, giveBaseCell: 122, giveDigits: upstreamDigits(), wantErr: h3.ErrBaseCellDomain, wantGoErr: h3go.ErrBaseCellDomain},
		"base_cell_negative_one": {giveRes: 0, giveBaseCell: -1, giveDigits: upstreamDigits(), wantErr: h3.ErrBaseCellDomain, wantGoErr: h3go.ErrBaseCellDomain},
		"base_cell_259":          {giveRes: 0, giveBaseCell: 259, giveDigits: upstreamDigits(), wantErr: h3.ErrBaseCellDomain, wantGoErr: h3go.ErrBaseCellDomain},
		"base_cell_122_at_res2":  {giveRes: 2, giveBaseCell: 122, giveDigits: upstreamDigits(1, 0), wantErr: h3.ErrBaseCellDomain, wantGoErr: h3go.ErrBaseCellDomain},

		// Digit domain.
		"digit_negative_one": {giveRes: 1, giveBaseCell: 40, giveDigits: upstreamDigits(-1), wantErr: h3.ErrDigitDomain, wantGoErr: h3go.ErrDigitDomain},
		"digit_seven":        {giveRes: 1, giveBaseCell: 40, giveDigits: upstreamDigits(7), wantErr: h3.ErrDigitDomain, wantGoErr: h3go.ErrDigitDomain},
		"digit_eight":        {giveRes: 1, giveBaseCell: 40, giveDigits: upstreamDigits(8), wantErr: h3.ErrDigitDomain, wantGoErr: h3go.ErrDigitDomain},
		"digit_seventeen":    {giveRes: 1, giveBaseCell: 40, giveDigits: upstreamDigits(17), wantErr: h3.ErrDigitDomain, wantGoErr: h3go.ErrDigitDomain},

		// Deleted subsequence, on pentagon base cell 4.
		"pentagon_digits_000":         {giveRes: 3, giveBaseCell: 4, giveDigits: upstreamDigits(0, 0, 0), wantCell: 0x830800fffffffff},
		"pentagon_digits_001_deleted": {giveRes: 3, giveBaseCell: 4, giveDigits: upstreamDigits(0, 0, 1), wantErr: h3.ErrDeletedDigit, wantGoErr: h3go.ErrDeletedDigit},
		"pentagon_digits_002":         {giveRes: 3, giveBaseCell: 4, giveDigits: upstreamDigits(0, 0, 2), wantCell: 0x830802fffffffff},

		// The same digits on hexagon base cell 5 are all valid.
		"hexagon_digits_000": {giveRes: 3, giveBaseCell: 5, giveDigits: upstreamDigits(0, 0, 0), wantCell: 0x830a00fffffffff},
		"hexagon_digits_001": {giveRes: 3, giveBaseCell: 5, giveDigits: upstreamDigits(0, 0, 1), wantCell: 0x830a01fffffffff},
		"hexagon_digits_002": {giveRes: 3, giveBaseCell: 5, giveDigits: upstreamDigits(0, 0, 2), wantCell: 0x830a02fffffffff},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotRef, refErr := h3.ConstructCell(tt.giveRes, tt.giveBaseCell, tt.giveDigits)
			if !errors.Is(refErr, tt.wantErr) {
				t.Fatalf("cgo ConstructCell(%d, %d) error = %v, want %v", tt.giveRes, tt.giveBaseCell, refErr, tt.wantErr)
			}

			if uint64(gotRef) != tt.wantCell {
				t.Fatalf("cgo ConstructCell(%d, %d) = %015x, want %015x", tt.giveRes, tt.giveBaseCell, uint64(gotRef), tt.wantCell)
			}

			got, err := h3go.ConstructCell(tt.giveRes, tt.giveBaseCell, tt.giveDigits)
			if !errors.Is(err, tt.wantGoErr) {
				t.Fatalf("h3go ConstructCell(%d, %d) error = %v, want %v", tt.giveRes, tt.giveBaseCell, err, tt.wantGoErr)
			}

			if uint64(got) != tt.wantCell {
				t.Fatalf("h3go ConstructCell(%d, %d) = %015x, want %015x", tt.giveRes, tt.giveBaseCell, uint64(got), tt.wantCell)
			}
		})
	}
}
