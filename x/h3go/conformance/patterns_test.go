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
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// patternsMaxCountEnv names the environment variable that raises the largest
// pattern tier the validity digest test recomputes. The file goes to ten
// million patterns; the default stops at one million.
const patternsMaxCountEnv = "H3_CONFORMANCE_PATTERNS_MAXCOUNT"

// defaultPatternsMaxCount is the largest tier checked without the
// environment variable.
const defaultPatternsMaxCount = 1_000_000

// Index layout and draw ranges, as README.md "Index construction" and
// "digests/patterns.jsonl" state them. These are deliberately the spec's
// numbers rather than h3go's internal constants: the sampler must draw the
// words the generator drew regardless of what the library under test
// believes the layout is, and a port in another language copies this block.
const (
	modeOffset     = 59
	reservedOffset = 56
	resOffset      = 52
	baseCellOffset = 45
	digitBits      = 3
	modeBits       = 4
	reservedBits   = 3
	baseCellBits   = 7
	numModes       = 16
	numDigits      = 7
	numDigitValues = 8
	patternKinds   = 8
	highBit        = uint64(1) << 63
	cellMode       = 1
	edgeMode       = 2
	vertexMode     = 4
)

// splitmix64 is the README's pseudo-random source.
type splitmix64 struct {
	state uint64
}

// draw returns the next 64 bits.
func (rng *splitmix64) draw() uint64 {
	rng.state += 0x9e3779b97f4a7c15
	z := rng.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb

	return z ^ (z >> 31)
}

// drawN returns one draw reduced modulo n.
func (rng *splitmix64) drawN(n int) int {
	return int(rng.draw() % uint64(n))
}

// digitShift returns the bit offset of the digit slot for resolution r.
func digitShift(r int) uint {
	return uint((h3go.MaxResolution - r) * digitBits)
}

// setField replaces width bits at offset with value.
func setField(index uint64, offset, width uint, value uint64) uint64 {
	mask := ((uint64(1) << width) - 1) << offset

	return (index &^ mask) | ((value << offset) & mask)
}

// drawCell is the README's "draw a cell at res": a base cell, res digits,
// and the remaining slots set to 7. The result may be invalid.
func (rng *splitmix64) drawCell(res int) uint64 {
	index := uint64(cellMode)<<modeOffset | uint64(res)<<resOffset
	index |= uint64(rng.drawN(h3go.NumBaseCells)) << baseCellOffset

	for r := 1; r <= h3go.MaxResolution; r++ {
		digit := uint64(numDigits)
		if r <= res {
			digit = uint64(rng.drawN(numDigits))
		}

		index |= digit << digitShift(r)
	}

	return index
}

// drawValidCell draws cells at res until isValidCell accepts one.
func (rng *splitmix64) drawValidCell(res int) uint64 {
	for {
		index := rng.drawCell(res)
		if h3go.Cell(int64(index)).IsValid() {
			return index
		}
	}
}

// drawPattern is one step of README.md "digests/patterns.jsonl": a uniform
// word, or a valid cell with one field mutated.
func (rng *splitmix64) drawPattern() uint64 {
	kind := rng.drawN(patternKinds)
	if kind == 0 {
		return rng.draw()
	}

	cell := rng.drawValidCell(rng.drawN(h3go.MaxResolution + 1))

	switch kind {
	case 1:
		return setField(cell, modeOffset, modeBits, uint64(rng.drawN(numModes)))
	case 2:
		return setField(cell, reservedOffset, reservedBits, uint64(rng.drawN(numDigitValues)))
	case 3:
		return setField(cell, resOffset, modeBits, uint64(rng.drawN(h3go.MaxResolution+1)))
	case 4:
		return setField(cell, baseCellOffset, baseCellBits, uint64(rng.drawN(1<<baseCellBits)))
	case 5:
		position := 1 + rng.drawN(h3go.MaxResolution)

		return setField(cell, digitShift(position), digitBits, uint64(rng.drawN(numDigitValues)))
	case 6:
		return cell | highBit
	default:
		mode := uint64(edgeMode)
		if rng.drawN(2) == 1 {
			mode = vertexMode
		}

		cell = setField(cell, modeOffset, modeBits, mode)

		return setField(cell, reservedOffset, reservedBits, uint64(rng.drawN(numDigitValues)))
	}
}

// patternsMaxCount returns the largest tier to recompute.
func patternsMaxCount(t *testing.T) int64 {
	t.Helper()

	value := os.Getenv(patternsMaxCountEnv)
	if value == "" {
		return defaultPatternsMaxCount
	}

	count, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		t.Fatalf("%s=%q: %v", patternsMaxCountEnv, value, err)
	}

	return count
}

// validityLine formats one pattern's line: the word in lowercase hex, then
// isValidCell, isValidIndex, isValidDirectedEdge and isValidVertex as 0 or 1.
func validityLine(word uint64) string {
	signed := int64(word)

	return strconv.FormatUint(word, 16) +
		" " + strconv.Itoa(boolInt(h3go.Cell(signed).IsValid())) +
		" " + strconv.Itoa(boolInt(h3go.IsValidIndex(h3go.Cell(signed)))) +
		" " + strconv.Itoa(boolInt(h3go.DirectedEdge(signed).IsValid())) +
		" " + strconv.Itoa(boolInt(h3go.Vertex(signed).IsValid())) + "\n"
}

// TestPatternDigests recomputes the validity stream over the first Count
// patterns of each tier and compares its digest.
func TestPatternDigests(t *testing.T) {
	t.Parallel()

	seed := loadSuite(t).Generator.Seed
	maxCount := patternsMaxCount(t)

	runGroup(t, "digests/patterns", func(t *testing.T, record PatternsDigestsRecord) {
		t.Helper()

		if record.Count > maxCount {
			t.Skipf("count %d is above %s=%d", record.Count, patternsMaxCountEnv, maxCount)
		}

		if len(record.Digests) != 1 {
			t.Fatalf("record has %d streams, want 1", len(record.Digests))
		}

		rng := &splitmix64{state: seed}
		digest := sha256.New()

		for range record.Count {
			digest.Write([]byte(validityLine(rng.drawPattern())))
		}

		if got := hex.EncodeToString(digest.Sum(nil)); got != record.Digests["validity"] {
			t.Errorf("validity digest = %s, want %s", got, record.Digests["validity"])
		}
	})
}
