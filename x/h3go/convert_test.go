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

import "testing"

// TestIndexToString covers IndexToString hex formatting.
func TestIndexToString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveIndex  uint64
		wantString string
	}{
		"small": {giveIndex: 0xcafe, wantString: "cafe"},
		"max":   {giveIndex: 0xffffffffffffffff, wantString: "ffffffffffffffff"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IndexToString(tt.giveIndex); got != tt.wantString {
				t.Fatalf("IndexToString(%x) = %q, want %q", tt.giveIndex, got, tt.wantString)
			}
		})
	}
}

// TestIndexFromString covers IndexFromString, including junk/empty input (which
// yields 0, matching the cgo binding that discards the parse error) and the
// "0x" prefix.
func TestIndexFromString(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		giveString string
		wantIndex  uint64
	}{
		"max":        {giveString: "ffffffffffffffff", wantIndex: 0xffffffffffffffff},
		"empty":      {giveString: "", wantIndex: 0},
		"junk":       {giveString: "**", wantIndex: 0},
		"hex_prefix": {giveString: "0xcafe", wantIndex: 0xcafe},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := IndexFromString(tt.giveString); got != tt.wantIndex {
				t.Fatalf("IndexFromString(%q) = %x, want %x", tt.giveString, got, tt.wantIndex)
			}
		})
	}
}

// TestCellTextMarshaling exercises MarshalText / UnmarshalText, including the
// invalid-input error path.
func TestCellTextMarshaling(t *testing.T) {
	t.Parallel()

	t.Run("round_trip", func(t *testing.T) {
		t.Parallel()

		ref, err := LatLngToCell(LatLng{Lat: 37.7749, Lng: -122.4194}, 9)
		if err != nil {
			t.Fatalf("LatLngToCell: %v", err)
		}

		text, err := ref.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText: %v", err)
		}

		var got Cell
		if err := got.UnmarshalText(text); err != nil {
			t.Fatalf("UnmarshalText: %v", err)
		}

		if got != ref {
			t.Fatalf("UnmarshalText round-trip: got %015x, want %015x", uint64(got), uint64(ref))
		}
	})

	t.Run("invalid_input", func(t *testing.T) {
		t.Parallel()

		var bad Cell
		if err := bad.UnmarshalText([]byte("not-a-cell")); err == nil {
			t.Fatal("UnmarshalText should fail on invalid input")
		}
	})
}
