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
