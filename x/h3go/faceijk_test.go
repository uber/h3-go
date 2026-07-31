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
	"math"
	"testing"
)

// farCoord is far beyond maxFaceCoord and stays out of range even after the
// aperture-7 reductions, so it deterministically trips faceIjkToH3's guards.
const farCoord = 1 << 20

// TestVec3ToHex2dFaceCenter covers the r < epsilon special case: a point exactly
// at a face's center point projects to the hex2d origin of that face. Real
// geographic inputs cannot reproduce this through LatLngToCell because the
// deg->rad->vec3 round-trip never lands within epsilon of a center, so the
// branch is exercised here against the raw center vectors.
func TestVec3ToHex2dFaceCenter(t *testing.T) {
	t.Parallel()

	for f := range faceCenterPoint {
		face, v := faceCenterPoint[f].toHex2d(0)
		if face != f {
			t.Fatalf("face center %d: closest face = %d, want %d", f, face, f)
		}

		if v.x != 0 || v.y != 0 {
			t.Fatalf("face center %d: hex2d = (%v, %v), want (0, 0)", f, v.x, v.y)
		}
	}
}

// TestFaceIjkToH3OutOfRange covers the defensive overflow guards. Face IJK
// coordinates beyond maxFaceCoord cannot encode a valid cell, so faceIjkToH3
// reports ErrFailed for both the resolution-0 path and the finer-resolution
// path (after the aperture-7 reductions). These guards are unreachable from
// LatLngToCell with valid finite input.
func TestFaceIjkToH3OutOfRange(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveRes int
	}{
		"resolution_0":     {giveRes: 0},
		"finer_resolution": {giveRes: 1},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fijk := faceIJK{face: 0, coord: coordIJK{i: farCoord}}

			got, err := fijk.toH3(tt.giveRes)
			if !errors.Is(err, ErrFailed) {
				t.Fatalf("faceIjkToH3 res %d: err = %v, want ErrFailed", tt.giveRes, err)
			}

			if got != 0 {
				t.Fatalf("faceIjkToH3 res %d: cell = %#x, want 0", tt.giveRes, uint64(got))
			}
		})
	}
}

// TestIcosahedronFacesKnown ports the testGetIcosahedronFaces.c regression cases:
// single-face and multi-face hexagons, and pentagons at several resolutions.
func TestIcosahedronFacesKnown(t *testing.T) {
	t.Parallel()

	validFaces := func(t *testing.T, c Cell) []int {
		t.Helper()

		faces, err := c.IcosahedronFaces()
		if err != nil {
			t.Fatalf("IcosahedronFaces(%015x): %v", uint64(c), err)
		}

		for _, face := range faces {
			if face < 0 || face > 19 {
				t.Fatalf("face %d out of range for %015x", face, uint64(c))
			}
		}

		return faces
	}

	t.Run("single_face_hexes", func(t *testing.T) {
		t.Parallel()

		// Base cell 16 sits at the center of an icosahedron face, so all of its
		// children share that single face.
		baseCell16 := setH3Index(0, 16, centerDigit)
		for _, childRes := range []int{2, 3} {
			children, err := baseCell16.Children(childRes)
			if err != nil {
				t.Fatalf("Children(%d): %v", childRes, err)
			}

			for _, child := range children {
				if got := validFaces(t, child); len(got) != 1 {
					t.Fatalf("child %015x: got %d faces, want 1", uint64(child), len(got))
				}
			}
		}
	})

	t.Run("hexagon_with_edge_vertices", func(t *testing.T) {
		t.Parallel()
		// Class II pentagon neighbor: one face, two adjacent vertices on an edge.
		if got := validFaces(t, CellFromString("821c37fffffffff")); len(got) != 1 {
			t.Fatalf("got %d faces, want 1", len(got))
		}
	})

	t.Run("hexagon_with_distortion", func(t *testing.T) {
		t.Parallel()
		// Class III pentagon neighbor: distortion spans two faces.
		if got := validFaces(t, CellFromString("831c06fffffffff")); len(got) != 2 {
			t.Fatalf("got %d faces, want 2", len(got))
		}
	})

	t.Run("hexagon_crossing_faces", func(t *testing.T) {
		t.Parallel()
		// Class II hexagon with two vertices on an edge.
		if got := validFaces(t, CellFromString("821ce7fffffffff")); len(got) != 2 {
			t.Fatalf("got %d faces, want 2", len(got))
		}
	})

	t.Run("pentagons", func(t *testing.T) {
		t.Parallel()
		// Class III (res 1), Class II (res 2), and res 15 pentagons on base cell 4.
		for _, res := range []int{1, 2, 15} {
			pentagon := setH3Index(res, 4, centerDigit)
			if !pentagon.IsPentagon() {
				t.Fatalf("setH3Index(%d,4,0) is not a pentagon", res)
			}

			if got := validFaces(t, pentagon); len(got) != 5 {
				t.Fatalf("res %d pentagon: got %d faces, want 5", res, len(got))
			}
		}
	})

	t.Run("base_cell_hexagons", func(t *testing.T) {
		t.Parallel()

		for bc := range NumBaseCells {
			if isBaseCellPentagon[bc] {
				continue
			}

			baseCell := setH3Index(0, bc, centerDigit)
			if got := validFaces(t, baseCell); len(got) < 1 {
				t.Fatalf("base cell %d: got no faces", bc)
			}
		}
	})
}

// TestIcosahedronFacesInvalid covers the defensive error paths: an out-of-range
// base cell (projection failure) and a malformed index whose vertices span more
// faces than the maximum (the hash-set overflow guard).
func TestIcosahedronFacesInvalid(t *testing.T) {
	t.Parallel()

	corrupt := CellFromString("8928308280fffff").setBaseCell(NumBaseCells)
	if _, err := corrupt.IcosahedronFaces(); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("corrupt base cell: got %v, want ErrCellInvalid", err)
	}

	overflow := Cell(0x08191d58a34080d2)
	if _, err := overflow.IcosahedronFaces(); !errors.Is(err, ErrFailed) {
		t.Fatalf("face-count overflow: got %v, want ErrFailed", err)
	}
}

// TestToVec3SubstrateClassIII covers the substrate aperture-7 branch that only
// fires for a hand-built odd-resolution substrate grid: H3's own builders bump
// res to Class II first, so no public call reaches it. The projected point must
// still land on the unit sphere.
func TestToVec3SubstrateClassIII(t *testing.T) {
	t.Parallel()

	const classIIIRes = 1 // odd => Class III
	if !isResClassIII(classIIIRes) {
		t.Fatalf("res %d should be Class III", classIIIRes)
	}

	point := vec2d{x: 0.5, y: 0.5}.toVec3(0, classIIIRes, true)

	mag := math.Sqrt(point.x*point.x + point.y*point.y + point.z*point.z)
	if math.Abs(mag-1) > epsilon {
		t.Fatalf("substrate Class III point off the unit sphere: mag=%v", mag)
	}
}
