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

import "errors"

// DirectedEdge is an H3 index that identifies a directed edge from an origin
// cell to one of its neighbors. The reserved field of the index holds the
// neighbor direction.
type DirectedEdge int64

// DirectedEdge returns the directed edge from c to other. The cells must be
// neighbors at the same resolution, otherwise it returns ErrNotNeighbors.
func (c Cell) DirectedEdge(other Cell) (DirectedEdge, error) {
	direction := c.directionForNeighbor(other)
	if direction == invalidDigit {
		return 0, ErrNotNeighbors
	}

	edge := c.setMode(directedEdgeMode).setReservedBits(direction)

	return DirectedEdge(edge), nil
}

// DirectedEdges returns the six directed edges with c as the origin. For a
// pentagon, the edge in the deleted k direction is omitted.
func (c Cell) DirectedEdges() ([]DirectedEdge, error) {
	isPent := c.IsPentagon()
	out := make([]DirectedEdge, 0, numCellEdges)

	for i := range numCellEdges {
		if isPent && i == 0 {
			continue
		}

		edge := c.setMode(directedEdgeMode).setReservedBits(i + 1)
		out = append(out, DirectedEdge(edge))
	}

	return out, nil
}

// IsValid reports whether the index is a valid H3 directed edge.
func (e DirectedEdge) IsValid() bool {
	neighborDirection := Cell(e).reservedBits()
	if neighborDirection <= centerDigit || neighborDirection >= numDigits {
		return false
	}

	origin, err := e.Origin()
	if err != nil {
		return false
	}

	if origin.IsPentagon() && neighborDirection == kAxesDigit {
		return false
	}

	return origin.IsValid()
}

// Origin returns the origin cell of the directed edge.
func (e DirectedEdge) Origin() (Cell, error) {
	if Cell(e).mode() != directedEdgeMode {
		return 0, ErrDirectedEdgeInvalid
	}

	return Cell(e).setMode(cellMode).setReservedBits(0), nil
}

// Destination returns the destination cell of the directed edge.
func (e DirectedEdge) Destination() (Cell, error) {
	origin, err := e.Origin()
	if err != nil {
		return 0, err
	}

	direction := Cell(e).reservedBits()

	destination, _, err := origin.neighborRotations(direction, 0)

	return destination, err
}

// Reverse returns the directed edge from this edge's destination back to its
// origin.
func (e DirectedEdge) Reverse() (DirectedEdge, error) {
	origin, err := e.Origin()
	if err != nil {
		return 0, err
	}

	destination, err := e.Destination()
	if err != nil {
		return 0, err
	}

	return destination.DirectedEdge(origin)
}

// Cells returns the origin and destination cells of the directed edge, in that
// order.
func (e DirectedEdge) Cells() ([]Cell, error) {
	origin, err := e.Origin()
	if err != nil {
		return nil, err
	}

	destination, err := e.Destination()
	if err != nil {
		return nil, err
	}

	return []Cell{origin, destination}, nil
}

// Boundary returns the coordinates of the directed edge: the geographic line
// from the center-relative start vertex to the end vertex of the origin cell.
// The boundary may contain an extra vertex where the edge crosses an
// icosahedron face boundary.
func (e DirectedEdge) Boundary() (CellBoundary, error) {
	direction := Cell(e).reservedBits()

	origin, err := e.Origin()
	if err != nil {
		return nil, err
	}

	fijk, err := origin.toFaceIjk()
	if err != nil {
		return nil, err
	}

	startVertex := origin.vertexNumForDirection(direction)
	if startVertex == invalidVertexNum {
		return nil, ErrDirectedEdgeInvalid
	}

	res := origin.Resolution()
	if origin.IsPentagon() {
		return fijk.pentToCellBoundary(res, startVertex, numEdgeCells), nil
	}

	return fijk.toCellBoundary(res, startVertex, numEdgeCells), nil
}

// Resolution returns the resolution of the directed edge.
func (e DirectedEdge) Resolution() int {
	return Cell(e).Resolution()
}

// IndexDigit returns the indexing digit of the directed edge at res, for res in
// [1, maxResolution].
func (e DirectedEdge) IndexDigit(res int) (int, error) {
	return Cell(e).IndexDigit(res)
}

// DirectedEdgeFromString returns a DirectedEdge parsed from its hexadecimal
// string representation. Callers should validate it with DirectedEdge.IsValid
// before use.
func DirectedEdgeFromString(s string) DirectedEdge {
	//nolint:gosec // an H3 index is a 64-bit value; uint64->int64 is a lossless reinterpretation.
	return DirectedEdge(IndexFromString(s))
}

// String returns the hexadecimal string representation of the directed edge.
func (e DirectedEdge) String() string {
	//nolint:gosec // an H3 index is a 64-bit value; int64->uint64 is a lossless reinterpretation.
	return IndexToString(uint64(e))
}

// MarshalText implements the encoding.TextMarshaler interface.
func (e DirectedEdge) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (e *DirectedEdge) UnmarshalText(text []byte) error {
	*e = DirectedEdgeFromString(string(text))
	if !e.IsValid() {
		return errors.New("invalid directed edge index")
	}

	return nil
}
