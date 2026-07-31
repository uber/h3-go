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
	"math/rand"
	"testing"
)

// maxCellBoundaryVerts bounds how many vertices a cell boundary can have: each
// topological vertex plus at most one distortion vertex per edge.
const maxCellBoundaryVerts = 2 * numHexVerts

// reverseCorpus builds a rich set of cells for the reverse-projection white-box
// tests. It combines the shared corpus, the descendants of every pentagon (which
// reach the pentagon rotation and overage branches), and a large deterministic
// random sweep that drives cells through every Class III edge-crossing path.
func reverseCorpus(t *testing.T) []Cell {
	t.Helper()

	cells := corpus(t)

	pents, err := Pentagons(0)
	if err != nil {
		t.Fatalf("Pentagons(0): %v", err)
	}

	for _, pent := range pents {
		for res := 1; res <= 3; res++ {
			children, err := pent.Children(res)
			if err != nil {
				t.Fatalf("Children(%015x, %d): %v", uint64(pent), res, err)
			}

			cells = append(cells, children...)
		}
	}

	const (
		seed       = 2
		iterations = 60000
		latSpan    = 180.0
		latOffset  = 90.0
		lngSpan    = 360.0
		lngOffset  = 180.0
		resCount   = MaxResolution + 1
	)

	rng := rand.New(rand.NewSource(seed))
	for range iterations {
		lat := rng.Float64()*latSpan - latOffset
		lng := rng.Float64()*lngSpan - lngOffset

		c, err := LatLngToCell(LatLng{Lat: lat, Lng: lng}, rng.Intn(resCount))
		if err != nil {
			t.Fatalf("LatLngToCell(%v, %v): %v", lat, lng, err)
		}

		cells = append(cells, c)
	}

	return cells
}

// TestCellToBoundarySweep exercises the boundary builders over the reverse
// corpus and checks each boundary is well-formed: a sane vertex count and every
// vertex in geographic range. Exact-value correctness is covered by the parity
// tests.
func TestCellToBoundarySweep(t *testing.T) {
	t.Parallel()

	for _, c := range reverseCorpus(t) {
		boundary, err := CellToBoundary(c)
		if err != nil {
			t.Fatalf("CellToBoundary(%015x): %v", uint64(c), err)
		}

		minVerts := numHexVerts
		if c.IsPentagon() {
			minVerts = numPentVerts
		}

		if len(boundary) < minVerts || len(boundary) > maxCellBoundaryVerts {
			t.Fatalf("CellToBoundary(%015x): %d vertices, want %d..%d",
				uint64(c), len(boundary), minVerts, maxCellBoundaryVerts)
		}

		for i, vert := range boundary {
			if math.IsNaN(vert.Lat) || math.IsNaN(vert.Lng) ||
				vert.Lat < -90 || vert.Lat > 90 || vert.Lng < -180 || vert.Lng > 180 {
				t.Fatalf("CellToBoundary(%015x) vertex %d = %+v: out of range", uint64(c), i, vert)
			}
		}
	}
}

// TestCellToBoundaryInvalidBaseCell covers the invalid-base-cell error branch of
// CellToBoundary, which is unreachable from a validly constructed cell.
func TestCellToBoundaryInvalidBaseCell(t *testing.T) {
	t.Parallel()

	bad := Cell(h3Init) | Cell(cellMode)<<modeOffset | Cell(NumBaseCells)<<baseCellOffset

	if _, err := CellToBoundary(bad); err == nil {
		t.Fatalf("CellToBoundary(%015x): got nil error, want ErrCellInvalid", uint64(bad))
	}
}

// TestCellToBoundaryClassIIIEdgeVertex is the regression for uber/h3#45: each of
// these Class III cells has a distortion vertex on a Class III edge and so must
// report seven boundary vertices.
func TestCellToBoundaryClassIIIEdgeVertex(t *testing.T) {
	t.Parallel()

	hexes := []string{
		"894cc5349b7ffff", "894cc534d97ffff", "894cc53682bffff",
		"894cc536b17ffff", "894cc53688bffff", "894cead92cbffff",
		"894cc536537ffff", "894cc5acbabffff", "894cc536597ffff",
	}

	for _, hex := range hexes {
		t.Run(hex, func(t *testing.T) {
			t.Parallel()

			boundary, err := CellToBoundary(CellFromString(hex))
			if err != nil {
				t.Fatalf("CellToBoundary(%s): %v", hex, err)
			}

			if len(boundary) != 7 {
				t.Fatalf("CellToBoundary(%s): %d vertices, want 7", hex, len(boundary))
			}
		})
	}
}

// TestCellToBoundaryKnownValues pins boundaries to exact values for cells from
// the reported bugs uber/h3#45 (a Class III distortion vertex) and uber/h3#212
// (a cos-longitude constraint near a face edge).
func TestCellToBoundaryKnownValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveCell string
		wantLoop CellBoundary
	}{
		"class_iii_edge_vertex_exact": {
			giveCell: "894cc536537ffff",
			wantLoop: CellBoundary{
				{18.043333154, -66.27836523500002},
				{18.042238363, -66.27929062800001},
				{18.040818259, -66.27854193899998},
				{18.040492975, -66.27686786700002},
				{18.041040385, -66.27640518300001},
				{18.041757122, -66.27596711500001},
				{18.043007860, -66.27669118199998},
			},
		},
		"coslng_constrain": {
			giveCell: "87dc6d364ffffff",
			wantLoop: CellBoundary{
				{-52.0130533678236091, -34.6232931343713091},
				{-52.0041156384652012, -34.6096733160584549},
				{-51.9929610229502472, -34.6165157145896387},
				{-51.9907410568096608, -34.6369680004259877},
				{-51.9996738734672377, -34.6505896528323660},
				{-52.0108315681413629, -34.6437571897165668},
			},
		},
	}

	const tolDegs = 1e-7

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := CellToBoundary(CellFromString(tt.giveCell))
			if err != nil {
				t.Fatalf("CellToBoundary(%s): %v", tt.giveCell, err)
			}

			if len(got) != len(tt.wantLoop) {
				t.Fatalf("CellToBoundary(%s): %d vertices, want %d", tt.giveCell, len(got), len(tt.wantLoop))
			}

			for i := range tt.wantLoop {
				if math.Abs(got[i].Lat-tt.wantLoop[i].Lat) > tolDegs ||
					math.Abs(got[i].Lng-tt.wantLoop[i].Lng) > tolDegs {
					t.Fatalf("CellToBoundary(%s) vertex %d = %+v, want %+v", tt.giveCell, i, got[i], tt.wantLoop[i])
				}
			}
		})
	}
}

// TestCellToBoundaryDoublePrecisionVertex is the regression for the double-
// precision edge-intersection case: a res-1 pentagon whose distortion vertices
// shift between float and double precision. The boundary must be geometrically
// consistent with indexing — a point that indexes to the cell must fall inside
// the boundary, and a point that does not must fall outside.
func TestCellToBoundaryDoublePrecisionVertex(t *testing.T) {
	t.Parallel()

	const cellHex = "81083ffffffffff"

	point := LatLng{Lat: 61.890838431, Lng: 8.644221328}
	cell := CellFromString(cellHex)

	boundary, err := CellToBoundary(cell)
	if err != nil {
		t.Fatalf("CellToBoundary(%s): %v", cellHex, err)
	}

	indexed, err := LatLngToCell(point, 1)
	if err != nil {
		t.Fatalf("LatLngToCell(%+v, 1): %v", point, err)
	}

	inside := pointInLoop(boundary, point)
	if indexed == cell && !inside {
		t.Fatalf("point %+v indexes to %s but lies outside its boundary", point, cellHex)
	}

	if indexed != cell && inside {
		t.Fatalf("point %+v indexes to %015x but lies inside %s's boundary", point, uint64(indexed), cellHex)
	}
}

// pointInLoop reports whether a geographic point lies inside the polygon
// described by loop, using even-odd ray casting in lat/lng. It is sufficient for
// loops that do not cross the antimeridian.
func pointInLoop(loop CellBoundary, point LatLng) bool {
	inside := false

	for i, j := 0, len(loop)-1; i < len(loop); j, i = i, i+1 {
		yi, xi := loop[i].Lat, loop[i].Lng
		yj, xj := loop[j].Lat, loop[j].Lng

		if (yi > point.Lat) != (yj > point.Lat) &&
			point.Lng < (xj-xi)*(point.Lat-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}

	return inside
}
