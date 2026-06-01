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

// Package h3core holds the cgo-free H3 index-format primitives shared by the
// cgo-backed h3 package and the pure-Go x/h3go package: the bit-layout
// constants and the pentagon base-cell table. Centralizing them here gives both
// packages a single source of truth without x/h3go taking on a cgo
// dependency. The values mirror the H3 C library (h3_constants.h and
// h3_h3Index.h); the x/h3go compatibility tests exercise the full pipeline
// against the C implementation, so any divergence is caught there.
package h3core

// H3 index modes, matching the H3_*_MODE values from h3_constants.h. The values
// are sequential (not bit flags), so iota tracks them directly.
const (
	CellMode         = iota + 1 // 1: H3_CELL_MODE
	DirectedEdgeMode            // 2: H3_DIRECTEDEDGE_MODE
	EdgeMode                    // 3: H3_EDGE_MODE (undirected edge)
	VertexMode                  // 4: H3_VERTEX_MODE
)

// H3 index bit-layout offsets and masks, matching h3_h3Index.h.
const (
	ModeOffset       = 59  // H3_MODE_OFFSET
	ResolutionOffset = 52  // H3_RES_OFFSET
	BaseCellOffset   = 45  // H3_BC_OFFSET
	PerDigitOffset   = 3   // H3_PER_DIGIT_OFFSET
	DigitMask        = 7   // H3_DIGIT_MASK
	ResolutionMask   = 0xF // low 4 bits of the resolution field

	// MaxResolution is the maximum H3 resolution (MAX_H3_RES).
	MaxResolution = 15
)

// IsBaseCellPentagon maps base cell number to whether it is a pentagon. There
// are exactly 12 pentagons at every resolution, one for each vertex of the
// icosahedron.
var IsBaseCellPentagon = [128]bool{
	4: true, 14: true, 24: true, 38: true, 49: true, 58: true,
	63: true, 72: true, 83: true, 97: true, 107: true, 117: true,
}
