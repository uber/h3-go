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

// Package h3 is a pure-Go implementation of Uber's H3 Geo Index system.
//
// It needs no C toolchain and no cgo. The API follows the H3 C library with
// the differences listed in README.md: LatLng holds degrees, the C "get"
// prefixes are dropped, and most functions are also methods on the type that
// is their main argument.
//
// The implementation is held to the H3 reference by the conformance suite in
// the conformance subdirectory, whose README states the contract: discrete
// outputs (indexes, counts, booleans, errors, sets and sequences of indexes)
// identical to the reference; coordinates within 1e-12 degrees of arc;
// measures within 1e-9 relative; and knife-edge inputs, such as points on
// cell edges and polygon vertices on cell edges, outside the contract. The
// README also records where this implementation is deliberately allowed to
// differ from the reference.
package h3
