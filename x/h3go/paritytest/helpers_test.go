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

// Package paritytest holds the cross-implementation parity tests that compare
// the pure-Go x/h3go package against the cgo-backed github.com/uber/h3-go/v4
// reference. They live in their own package so the h3go package's own tests can
// stay pure (white-box, no cgo dependency).
package paritytest

import (
	"slices"
	"testing"

	"github.com/uber/h3-go/v4"
	"github.com/uber/h3-go/v4/x/h3go"
)

// corpusPoints is a spread of coordinates including the poles and antimeridian,
// reused to derive reference cells across the parity tests.
var corpusPoints = []h3.LatLng{
	{Lat: 0, Lng: 0},
	{Lat: 37.7749, Lng: -122.4194},
	{Lat: 67.1509, Lng: -168.3908},
	{Lat: -45, Lng: 170},
	{Lat: 89.9, Lng: 0},
	{Lat: -89.9, Lng: 179.9},
	{Lat: 11.7, Lng: 13.4},
}

// referenceCorpus builds a deterministic set of reference cells exercising base
// cells, hexagons and pentagons across every resolution. Cells are returned as
// the cgo h3.Cell type so callers can compare both implementations.
func referenceCorpus(t *testing.T) []h3.Cell {
	t.Helper()
	var cells []h3.Cell

	res0, err := h3.Res0Cells()
	if err != nil {
		t.Fatalf("h3.Res0Cells: %v", err)
	}

	cells = append(cells, res0...)

	for res := 0; res <= h3.MaxResolution; res++ {
		for _, p := range corpusPoints {
			c, err := h3.LatLngToCell(p, res)
			if err == nil {
				cells = append(cells, c)
			}
		}

		pents, err := h3.Pentagons(res)
		if err != nil {
			t.Fatalf("h3.Pentagons(%d): %v", res, err)
		}

		cells = append(cells, pents...)
	}

	return cells
}

// assertSameCellSet fails if the two cell slices differ as sets (order ignored).
func assertSameCellSet[A ~int64, B ~int64](t *testing.T, got []A, want []B, msg string) {
	t.Helper()
	slices.Sort(got)
	slices.Sort(want)

	if len(got) != len(want) {
		t.Fatalf("%s: len got=%d want=%d", msg, len(got), len(want))
	}

	for i := range got {
		if int64(got[i]) != int64(want[i]) {
			t.Fatalf("%s: element %d got=%015x want=%015x", msg, i, got[i], want[i])
		}
	}
}

// bothErr reports whether two error values agree on success/failure.
func bothErr(a, b error) bool {
	return (a == nil) == (b == nil)
}

// h3goCell converts a cgo cell to the pure-Go Cell type.
func h3goCell(c h3.Cell) h3go.Cell {
	return h3go.Cell(c)
}

// toGoCells converts a slice of cgo cells to pure-Go cells.
func toGoCells(in []h3.Cell) []h3go.Cell {
	out := make([]h3go.Cell, len(in))
	for i, c := range in {
		out[i] = h3go.Cell(c)
	}

	return out
}

// res0Cell builds the resolution-0 H3 index for a base cell number. The layout
// is mode=1 (cell), resolution=0, the base cell in its field, and all 15 digit
// slots set to 7 (the canonical "unused digit" pattern).
func res0Cell(baseCell int) h3go.Cell {
	const (
		cellMode       = 1
		modeOffset     = 59
		baseCellOffset = 45
		numDigits      = 15
		digitBits      = 3
		allDigitsSet   = (1 << (numDigits * digitBits)) - 1
	)

	return h3go.Cell(cellMode<<modeOffset | baseCell<<baseCellOffset | allDigitsSet)
}
