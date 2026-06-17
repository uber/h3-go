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
	"sort"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// TestIcosahedronFacesMatchesCgo asserts Cell.IcosahedronFaces matches the cgo
// reference across the corpus, as a sorted set.
func TestIcosahedronFacesMatchesCgo(t *testing.T) {
	t.Parallel()

	for _, cell := range referenceCorpus(t) {
		want, wantErr := cell.IcosahedronFaces()
		got, gotErr := h3go.Cell(cell).IcosahedronFaces()

		if !bothErr(wantErr, gotErr) {
			t.Fatalf("IcosahedronFaces(%015x) error: cgo=%v h3go=%v", uint64(cell), wantErr, gotErr)
		}

		if wantErr != nil {
			continue
		}

		sort.Ints(want)
		sort.Ints(got)

		if len(want) != len(got) {
			t.Fatalf("IcosahedronFaces(%015x): cgo=%v h3go=%v", uint64(cell), want, got)
		}

		for i := range want {
			if want[i] != got[i] {
				t.Fatalf("IcosahedronFaces(%015x): cgo=%v h3go=%v", uint64(cell), want, got)
			}
		}
	}
}
