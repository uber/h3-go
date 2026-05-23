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
package h3

import (
	"strconv"
	"testing"
)

var sink int

func BenchmarkNumCells(b *testing.B) {
	for r := 0; r <= MaxResolution; r++ {
		b.Run(strconv.Itoa(r), func(b *testing.B) {
			var n int
			for b.Loop() {
				n = NumCells(r)
			}
			sink = n
		})
	}
}
