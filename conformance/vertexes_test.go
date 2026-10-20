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

package conformance

import (
	"slices"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// TestVertexes runs every vertexes record in the suite against h3go.
func TestVertexes(t *testing.T) {
	t.Parallel()

	runGroup(t, "vertexes/", checkVertexes)
}

// checkVertexes asserts every field of one vertexes record against h3go.
func checkVertexes(t *testing.T, record VertexesRecord) {
	t.Helper()

	cell := h3go.CellFromString(record.Index)

	vertexes, err := cell.Vertexes()
	if err != nil {
		t.Fatalf("Vertexes(): %v", err)
	}

	slices.Sort(vertexes)

	got := make([]string, len(vertexes))
	for i, vertex := range vertexes {
		got[i] = vertex.String()

		if !vertex.IsValid() {
			t.Errorf("IsValid(%s) = false", vertex)
		}
	}

	if !slices.Equal(got, record.Vertexes) {
		t.Errorf("Vertexes() = %v, want %v", got, record.Vertexes)
	}

	if len(record.ByNumber) != 6 {
		t.Fatalf("record has %d byNumber entries, want 6", len(record.ByNumber))
	}

	for number, want := range record.ByNumber {
		vertex, err := cell.Vertex(number)

		result := Result[string]{Err: ErrorName(err)}
		if err == nil {
			result.Value = vertex.String()
		}

		if !equalResults(result, want, equalValues) {
			t.Errorf("Vertex(%d) = %s, want %s", number, formatResult(result), formatResult(want))
		}
	}
}
