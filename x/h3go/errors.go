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

// Error codes. Messages mirror the H3 C library's error strings.
var (
	ErrFailed                = errors.New("the operation failed")
	ErrDomain                = errors.New("argument was outside of acceptable range")
	ErrLatLngDomain          = errors.New("latitude or longitude arguments were outside of acceptable range")
	ErrResolutionDomain      = errors.New("resolution argument was outside of acceptable range")
	ErrResolutionMismatch    = errors.New("H3Index cell arguments had incompatible resolutions")
	ErrCellInvalid           = errors.New("H3Index cell argument was not valid")
	ErrDirectedEdgeInvalid   = errors.New("H3Index directed edge argument was not valid")
	ErrNotNeighbors          = errors.New("H3Index cell arguments were not neighbors")
	ErrDuplicateInput        = errors.New("duplicate input was encountered in the arguments")
	ErrPentagon              = errors.New("pentagon distortion was encountered")
	ErrMemoryAlloc           = errors.New("necessary memory allocation failed")
	ErrMemoryBounds          = errors.New("bounds of provided memory were not large enough")
	ErrOptionInvalid         = errors.New("mode or flags argument was not valid")
	ErrUndirectedEdgeInvalid = errors.New("H3Index undirected edge argument was not valid")
	ErrVertexInvalid         = errors.New("H3Index vertex argument was not valid")
	ErrIndexInvalid          = errors.New("index argument was not valid")
	ErrBaseCellDomain        = errors.New("base cell number was outside of acceptable range")
	ErrDigitDomain           = errors.New("child digits invalid")
	ErrDeletedDigit          = errors.New("deleted subsequence indicates invalid index")
)
