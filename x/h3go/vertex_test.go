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

// TestVertexRoundTrip checks that every vertex of every corpus cell round trips
// through LatLng and reconstructs as a valid, canonical vertex.
func TestVertexRoundTrip(t *testing.T) {
	t.Parallel()

	for _, origin := range gridCorpus(t) {
		vertexes, err := origin.Vertexes()
		if err != nil {
			t.Fatalf("Vertexes(%015x): %v", uint64(origin), err)
		}

		expected := numHexVerts
		if origin.IsPentagon() {
			expected = numPentVerts
		}

		if len(vertexes) != expected {
			t.Fatalf("Vertexes(%015x): got %d, want %d", uint64(origin), len(vertexes), expected)
		}

		for _, vertex := range vertexes {
			if !vertex.IsValid() {
				t.Fatalf("vertex %015x of %015x is not valid", uint64(vertex), uint64(origin))
			}

			if vertex.Resolution() != origin.Resolution() {
				t.Fatalf("vertex %015x resolution mismatch", uint64(vertex))
			}

			if _, err := vertex.LatLng(); err != nil {
				t.Fatalf("LatLng(%015x): %v", uint64(vertex), err)
			}
		}
	}
}

// TestCellToVertexFreeFunction checks the free function delegates to the method.
func TestCellToVertexFreeFunction(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	method, err := origin.Vertex(0)
	if err != nil {
		t.Fatalf("Vertex: %v", err)
	}

	free, err := CellToVertex(origin, 0)
	if err != nil || free != method {
		t.Fatalf("CellToVertex: got %015x (%v), want %015x", uint64(free), err, uint64(method))
	}

	if latLng, err := VertexToLatLng(method); err != nil || latLng == (LatLng{}) {
		t.Fatalf("VertexToLatLng: got %v (%v)", latLng, err)
	}

	all, err := CellToVertexes(origin)
	if err != nil || len(all) != numHexVerts {
		t.Fatalf("CellToVertexes: got %d (%v), want %d", len(all), err, numHexVerts)
	}
}

// TestVertexNumOutOfRange covers the domain check for invalid vertex numbers on
// both hexagons and pentagons.
func TestVertexNumOutOfRange(t *testing.T) {
	t.Parallel()

	hexagon := CellFromString("8928308280fffff")
	if _, err := hexagon.Vertex(numHexVerts); !errors.Is(err, ErrDomain) {
		t.Fatalf("Vertex(6) on hexagon: got %v, want ErrDomain", err)
	}

	if _, err := hexagon.Vertex(-1); !errors.Is(err, ErrDomain) {
		t.Fatalf("Vertex(-1): got %v, want ErrDomain", err)
	}

	pents, err := Pentagons(2)
	if err != nil {
		t.Fatalf("Pentagons: %v", err)
	}

	if _, err := pents[0].Vertex(numPentVerts); !errors.Is(err, ErrDomain) {
		t.Fatalf("Vertex(5) on pentagon: got %v, want ErrDomain", err)
	}
}

// TestVertexNumForDirectionInvalid covers the invalid-direction guards.
func TestVertexNumForDirectionInvalid(t *testing.T) {
	t.Parallel()

	hexagon := CellFromString("8928308280fffff")
	if got := hexagon.vertexNumForDirection(centerDigit); got != invalidVertexNum {
		t.Fatalf("vertexNumForDirection(center): got %d, want %d", got, invalidVertexNum)
	}

	if got := hexagon.vertexNumForDirection(invalidDigit); got != invalidVertexNum {
		t.Fatalf("vertexNumForDirection(invalid): got %d, want %d", got, invalidVertexNum)
	}

	pents, err := Pentagons(2)
	if err != nil {
		t.Fatalf("Pentagons: %v", err)
	}

	if got := pents[0].vertexNumForDirection(kAxesDigit); got != invalidVertexNum {
		t.Fatalf("vertexNumForDirection(k) on pentagon: got %d, want %d", got, invalidVertexNum)
	}
}

// TestDirectionForVertexNumInvalid covers the out-of-range guard.
func TestDirectionForVertexNumInvalid(t *testing.T) {
	t.Parallel()

	hexagon := CellFromString("8928308280fffff")
	if got := hexagon.directionForVertexNum(-1); got != invalidDigit {
		t.Fatalf("directionForVertexNum(-1): got %d, want %d", got, invalidDigit)
	}

	if got := hexagon.directionForVertexNum(numHexVerts); got != invalidDigit {
		t.Fatalf("directionForVertexNum(6): got %d, want %d", got, invalidDigit)
	}
}

// TestVertexRotationsError covers the projection failure path of vertexRotations
// and the helpers that depend on it, using a corrupt base cell.
func TestVertexRotationsError(t *testing.T) {
	t.Parallel()

	corrupt := CellFromString("8928308280fffff").setBaseCell(NumBaseCells)

	if _, err := corrupt.vertexRotations(); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("vertexRotations(corrupt): got %v, want ErrCellInvalid", err)
	}

	if got := corrupt.vertexNumForDirection(jAxesDigit); got != invalidVertexNum {
		t.Fatalf("vertexNumForDirection(corrupt): got %d, want %d", got, invalidVertexNum)
	}

	if got := corrupt.directionForVertexNum(0); got != invalidDigit {
		t.Fatalf("directionForVertexNum(corrupt): got %d, want %d", got, invalidDigit)
	}
}

// TestVertexOwnerFails covers the failed-owner path of Vertex, reached when the
// vertex direction cannot be resolved because the cell does not project.
func TestVertexOwnerFails(t *testing.T) {
	t.Parallel()

	// A res-0 cell with an out-of-range base cell enters the owner search (res 0)
	// but cannot determine its vertex direction, so the lookup fails.
	corrupt := Cell(h3Init).setMode(cellMode).setBaseCell(NumBaseCells)

	if _, err := corrupt.Vertex(0); !errors.Is(err, ErrFailed) {
		t.Fatalf("Vertex(corrupt res0): got %v, want ErrFailed", err)
	}
}

// TestVertexCenterChildShortcut covers the branch where a center child is its own
// owner, skipping the neighbor search.
func TestVertexCenterChildShortcut(t *testing.T) {
	t.Parallel()

	// A cell whose finest digit is the center digit owns its vertexes directly.
	origin := CellFromString("8928308280fffff")

	center := origin.setIndexDigit(origin.Resolution(), centerDigit)
	if !center.IsValid() {
		t.Skip("constructed center child is not valid")
	}

	vertex, err := center.Vertex(0)
	if err != nil {
		t.Fatalf("Vertex(center child): %v", err)
	}

	owner := Cell(vertex).setMode(cellMode).setReservedBits(0)
	if owner != center {
		t.Fatalf("center child should own its vertex: got owner %015x, want %015x", uint64(owner), uint64(center))
	}
}

// TestVertexIsValidPaths covers the invalid-mode, invalid-owner and
// non-canonical branches of Vertex.IsValid.
func TestVertexIsValidPaths(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	// A plain cell has cell mode, not vertex mode.
	if Vertex(origin).IsValid() {
		t.Fatal("cell-mode index should not be a valid vertex")
	}

	// A vertex over an invalid owner is invalid.
	badOwner := Cell(0).setMode(vertexMode)
	if Vertex(badOwner).IsValid() {
		t.Fatal("vertex with zero owner should be invalid")
	}

	// A vertex whose number is out of range for its (valid) owner is invalid:
	// reconstruction returns a domain error.
	outOfRange := Vertex(origin.setMode(vertexMode).setReservedBits(numHexVerts))
	if outOfRange.IsValid() {
		t.Fatal("vertex with out-of-range number should be invalid")
	}

	// A non-canonical vertex: find a cell that is not the owner of one of its
	// vertexes, then build a vertex index anchored at that cell. Reconstruction
	// resolves to the real (lower-index) owner, so the index is non-canonical.
	found := false

	for _, cell := range gridCorpus(t) {
		numVerts := numHexVerts
		if cell.IsPentagon() {
			numVerts = numPentVerts
		}

		for vertexNum := range numVerts {
			vertex, err := cell.Vertex(vertexNum)
			if err != nil {
				continue
			}

			owner := Cell(vertex).setMode(cellMode).setReservedBits(0)
			if owner == cell {
				continue
			}

			nonCanonical := Vertex(cell.setMode(vertexMode).setReservedBits(vertexNum))
			if nonCanonical.IsValid() {
				t.Fatalf("non-canonical vertex %015x should be invalid", uint64(nonCanonical))
			}

			found = true

			break
		}

		if found {
			break
		}
	}

	if !found {
		t.Fatal("no non-owned vertex found in corpus")
	}
}

// TestVertexStringRoundTrip covers the string, marshal and unmarshal helpers.
func TestVertexStringRoundTrip(t *testing.T) {
	t.Parallel()

	origin := CellFromString("8928308280fffff")

	vertex, err := origin.Vertex(0)
	if err != nil {
		t.Fatalf("Vertex: %v", err)
	}

	parsed := VertexFromString(vertex.String())
	if parsed != vertex {
		t.Fatalf("VertexFromString round trip: got %015x, want %015x", uint64(parsed), uint64(vertex))
	}

	text, err := vertex.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}

	var unmarshaled Vertex
	if err := unmarshaled.UnmarshalText(text); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}

	if unmarshaled != vertex {
		t.Fatalf("UnmarshalText: got %015x, want %015x", uint64(unmarshaled), uint64(vertex))
	}

	if err := unmarshaled.UnmarshalText([]byte("not-a-vertex")); err == nil {
		t.Fatal("UnmarshalText(invalid): got nil error, want failure")
	}

	if digit, err := vertex.IndexDigit(1); err != nil {
		t.Fatalf("IndexDigit(1): %v, digit=%d", err, digit)
	}

	if _, err := vertex.IndexDigit(0); !errors.Is(err, ErrResolutionDomain) {
		t.Fatalf("IndexDigit(0): want ErrResolutionDomain")
	}
}

// TestBaseCellToCCWrot60Invalid covers the not-found branches of the helper.
func TestBaseCellToCCWrot60Invalid(t *testing.T) {
	t.Parallel()

	if got := baseCellToCCWrot60(0, -1); got != invalidRotations {
		t.Fatalf("baseCellToCCWrot60(0, -1): got %d, want %d", got, invalidRotations)
	}

	if got := baseCellToCCWrot60(NumBaseCells, 0); got != invalidRotations {
		t.Fatalf("baseCellToCCWrot60(out of range, 0): got %d, want %d", got, invalidRotations)
	}
}

// TestVertexRotationsIKtoJK covers the IK-to-JK pentagon-crossing rotation case
// of vertexRotations, reached by a pentagon-base-cell cell whose leading digit
// is the IK axis and which lands on the JK direction's face.
func TestVertexRotationsIKtoJK(t *testing.T) {
	t.Parallel()

	cell := CellFromString("84082b7ffffffff")

	if _, err := cell.vertexRotations(); err != nil {
		t.Fatalf("vertexRotations: %v", err)
	}

	// Exercise it through the public API as well.
	if _, err := cell.Vertexes(); err != nil {
		t.Fatalf("Vertexes: %v", err)
	}
}

// TestVertexNeighborErrors covers the left/right neighbor failure branches of the
// owner search and their propagation through Vertexes, using a cell whose
// corrupt interior digit makes neighbor stepping fail.
func TestVertexNeighborErrors(t *testing.T) {
	t.Parallel()

	// 8928308280fffff with the digit at resolution 8 set to the unused sentinel.
	corrupt := CellFromString("892830828efffff")

	if _, err := corrupt.vertexRotations(); err != nil {
		t.Fatalf("vertexRotations should still succeed on this cell: %v", err)
	}

	// Vertex 1's left neighbor cannot be stepped to.
	if _, err := corrupt.Vertex(1); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("Vertex(1): got %v, want ErrCellInvalid", err)
	}

	// Vertex 4's left neighbor succeeds but the right neighbor fails.
	if _, err := corrupt.Vertex(4); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("Vertex(4): got %v, want ErrCellInvalid", err)
	}

	// Vertexes propagates the per-vertex failure.
	if _, err := corrupt.Vertexes(); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("Vertexes: got %v, want ErrCellInvalid", err)
	}
}

// TestVertexLatLngOwnerError covers the projection-failure path of LatLng for a
// vertex anchored on a corrupt owner, and the free IsValidVertex helper.
func TestVertexLatLngOwnerError(t *testing.T) {
	t.Parallel()

	corrupt := Cell(h3Init).setMode(vertexMode).setBaseCell(NumBaseCells)

	if _, err := Vertex(corrupt).LatLng(); !errors.Is(err, ErrCellInvalid) {
		t.Fatalf("LatLng(corrupt owner): got %v, want ErrCellInvalid", err)
	}

	if IsValidVertex(Vertex(corrupt)) {
		t.Fatal("IsValidVertex(corrupt) should be false")
	}

	valid, err := CellFromString("8928308280fffff").Vertex(0)
	if err != nil {
		t.Fatalf("Vertex: %v", err)
	}

	if !IsValidVertex(valid) {
		t.Fatal("IsValidVertex(valid) should be true")
	}
}

// TestCellToVertexInvalidLiterals ports the testVertex.c cellToVertex_invalid2
// and invalid3 regressions: specific malformed indexes must fail with
// ErrCellInvalid.
func TestCellToVertexInvalidLiterals(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveCell      Cell
		giveVertexNum int
	}{
		"invalid2": {giveCell: Cell(0x685b2396e900fff9), giveVertexNum: 2},
		"invalid3": {giveCell: Cell(0x20ff20202020ff35), giveVertexNum: 0},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := tt.giveCell.Vertex(tt.giveVertexNum); !errors.Is(err, ErrCellInvalid) {
				t.Fatalf("Vertex: got %v, want ErrCellInvalid", err)
			}
		})
	}
}

// TestIsValidVertexKnownLiteral ports the testVertex.c isValidVertex_hex
// regression: a specific known-valid vertex index validates.
func TestIsValidVertexKnownLiteral(t *testing.T) {
	t.Parallel()

	if !IsValidVertex(Vertex(0x2222597fffffffff)) {
		t.Fatal("IsValidVertex(0x2222597fffffffff): got false, want true")
	}
}
