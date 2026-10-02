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
	"hash"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// digestsMaxResEnv names the environment variable that raises the finest
// resolution the digest tests enumerate. The files go to resolution 6, which
// takes minutes, so the default stops where a race-detector run still takes
// seconds.
const digestsMaxResEnv = "H3_CONFORMANCE_DIGESTS_MAXRES"

// defaultDigestsMaxRes is the finest resolution checked without the
// environment variable.
const defaultDigestsMaxRes = 3

// digestsMaxRes returns the finest resolution to enumerate.
func digestsMaxRes(t *testing.T) int {
	t.Helper()

	value := os.Getenv(digestsMaxResEnv)
	if value == "" {
		return defaultDigestsMaxRes
	}

	res, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("%s=%q: %v", digestsMaxResEnv, value, err)
	}

	return res
}

// TestResolutionDigests enumerates every cell at each resolution up to the
// limit and compares every stream digest.
func TestResolutionDigests(t *testing.T) {
	t.Parallel()

	maxRes := digestsMaxRes(t)

	runGroup(t, "digests/resolutions", func(t *testing.T, record ResolutionDigestsRecord) {
		t.Helper()

		if record.Res > maxRes {
			t.Skipf("resolution %d is above %s=%d", record.Res, digestsMaxResEnv, maxRes)
		}

		res0, err := h3go.Res0Cells()
		if err != nil {
			t.Fatalf("Res0Cells(): %v", err)
		}

		cells, err := h3go.UncompactCells(res0, record.Res)
		if err != nil {
			t.Fatalf("UncompactCells(%d): %v", record.Res, err)
		}

		checkDigests(t, cells, record.Res, record.Count, record.Digests)
	})
}

// TestBaseCellDigests enumerates the cells under each base cell at each
// resolution up to the limit and compares every stream digest.
func TestBaseCellDigests(t *testing.T) {
	t.Parallel()

	maxRes := digestsMaxRes(t)

	runGroup(t, "digests/baseCells", func(t *testing.T, record BaseCellDigestsRecord) {
		t.Helper()

		if record.Res > maxRes {
			t.Skipf("resolution %d is above %s=%d", record.Res, digestsMaxResEnv, maxRes)
		}

		base, err := h3go.ConstructCell(0, record.BaseCell, nil)
		if err != nil {
			t.Fatalf("ConstructCell(0, %d): %v", record.BaseCell, err)
		}

		cells, err := base.Children(record.Res)
		if err != nil {
			t.Fatalf("Children(%s, %d): %v", base, record.Res, err)
		}

		checkDigests(t, cells, record.Res, record.Count, record.Digests)
	})
}

// checkDigests sorts the cells, runs every stream over them, and compares the
// count and each digest with the record.
func checkDigests(t *testing.T, cells []h3go.Cell, res int, count int64, want map[string]string) {
	t.Helper()

	slices.Sort(cells)

	if int64(len(cells)) != count {
		t.Errorf("enumerated %d cells, want %d", len(cells), count)
	}

	got := digestStreams(t, cells, res)

	if len(got) != len(want) {
		t.Errorf("record has %d streams, runner computes %d", len(want), len(got))
	}

	for name, digest := range got {
		if digest != want[name] {
			t.Errorf("%s digest = %s, want %s", name, digest, want[name])
		}
	}
}

// streams lists the digest streams in key order.
var streams = []string{ //nolint:gochecknoglobals // fixed by the format
	"cells", "children", "compact", "disks", "distances", "edges",
	"inspection", "localIj", "parents", "rings", "roundTrip", "vertexes",
}

// lineWriter builds one text line and feeds it to a stream's hash.
type lineWriter struct {
	hashes map[string]hash.Hash
	line   strings.Builder
}

// start begins a line with the subject cell.
func (w *lineWriter) start(cell h3go.Cell) {
	w.line.Reset()
	w.line.WriteString(cell.String())
}

// field appends one space-separated field.
func (w *lineWriter) field(text string) {
	w.line.WriteByte(' ')
	w.line.WriteString(text)
}

// index appends an index's canonical string.
func (w *lineWriter) index(index interface{ String() string }) {
	w.field(index.String())
}

// integer appends a decimal integer.
func (w *lineWriter) integer(value int) {
	w.field(strconv.Itoa(value))
}

// result appends a decimal integer, or the error's name.
func (w *lineWriter) result(value int, err error) {
	if err != nil {
		w.field(ErrorName(err))

		return
	}

	w.integer(value)
}

// cellSet appends distinct cells in ascending order and returns them.
func (w *lineWriter) cellSet(cells []h3go.Cell) []h3go.Cell {
	slices.Sort(cells)
	cells = slices.Compact(cells)

	for _, cell := range cells {
		w.index(cell)
	}

	return cells
}

// finish terminates the line and feeds it to the named stream.
func (w *lineWriter) finish(stream string) {
	w.line.WriteByte('\n')
	w.hashes[stream].Write([]byte(w.line.String()))
}

// digestStreams runs every stream over the sorted cells and returns the hex
// digests by stream name.
func digestStreams(t *testing.T, cells []h3go.Cell, res int) map[string]string {
	t.Helper()

	writer := &lineWriter{hashes: map[string]hash.Hash{}}
	for _, name := range streams {
		writer.hashes[name] = sha256.New()
	}

	defer func() {
		if failure := recover(); failure != nil {
			t.Fatalf("stream computation failed: %v", failure)
		}
	}()

	for _, cell := range cells {
		digestCell(writer, cell, res)
	}

	out := make(map[string]string, len(streams))
	for name, h := range writer.hashes {
		out[name] = hex.EncodeToString(h.Sum(nil))
	}

	return out
}

// digestCell appends one cell's line to every stream.
func digestCell(w *lineWriter, cell h3go.Cell, res int) {
	w.start(cell)
	w.finish("cells")

	parent0 := must(cell.Parent(0))

	w.start(cell)

	for parentRes := res - 1; parentRes >= 0; parentRes-- {
		w.index(must(cell.Parent(parentRes)))
	}

	w.finish("parents")

	w.start(cell)
	w.index(must(cell.CenterChild(res + 1)))

	children := must(cell.Children(res + 1))
	w.integer(len(children))

	pos := must(cell.ChildPos(0))
	w.integer(pos)
	w.index(must(parent0.ChildPosToCell(pos, res)))
	w.finish("children")

	w.start(cell)
	w.integer(cell.BaseCellNumber())
	w.integer(boolInt(cell.IsPentagon()))
	w.integer(boolInt(cell.IsResClassIII()))

	faces := must(cell.IcosahedronFaces())
	slices.Sort(faces)

	for _, face := range faces {
		w.integer(face)
	}

	w.finish("inspection")

	var disk3 []h3go.Cell

	w.start(cell)

	for k := 1; k <= 3; k++ {
		if k > 1 {
			w.field("|")
		}

		disk := w.cellSet(must(cell.GridDisk(k)))
		if k == 3 {
			disk3 = disk
		}
	}

	w.finish("disks")

	w.start(cell)

	for k := 1; k <= 3; k++ {
		if k > 1 {
			w.field("|")
		}

		w.cellSet(must(cell.GridRing(k)))
	}

	w.finish("rings")

	w.start(cell)

	for _, other := range disk3 {
		w.result(cell.GridDistance(other))
	}

	w.finish("distances")

	w.start(cell)

	edges := must(cell.DirectedEdges())
	slices.Sort(edges)

	for _, edge := range edges {
		w.index(edge)
		w.index(must(edge.Destination()))
	}

	w.field("|")

	disk1 := must(cell.GridDisk(1))
	slices.Sort(disk1)

	for _, other := range slices.Compact(disk1) {
		neighbor, err := cell.IsNeighbor(other)
		w.result(boolInt(neighbor), err)
	}

	w.finish("edges")

	w.start(cell)

	vertexes := must(cell.Vertexes())
	slices.Sort(vertexes)

	for _, vertex := range vertexes {
		w.index(vertex)
	}

	w.finish("vertexes")

	w.start(cell)

	origin := parent0
	if res > 0 {
		origin = must(parent0.CenterChild(res))
	}

	if ij, err := h3go.CellToLocalIJ(origin, cell); err != nil {
		w.field(ErrorName(err))
	} else {
		w.integer(ij.I)
		w.integer(ij.J)
	}

	w.finish("localIj")

	w.start(cell)
	slices.Sort(children)
	w.cellSet(must(h3go.CompactCells(children[:len(children)-1])))
	w.finish("compact")

	w.start(cell)
	w.index(must(h3go.LatLngToCell(must(cell.LatLng()), res)))
	w.finish("roundTrip")
}

// must panics on an error from a call that the stream definition does not
// allow to fail; digestStreams turns the panic into a test failure.
func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

// boolInt is 1 for true and 0 for false, as C returns booleans.
func boolInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
