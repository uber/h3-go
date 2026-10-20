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
	"fmt"
	"hash"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/uber/h3-go/v4/x/h3go"
)

// digestsMaxResEnv names the environment variable that raises the finest
// resolution the digest tests enumerate. The files go to resolution 7, which
// takes tens of minutes, so the default stops where a race-detector run
// still takes seconds.
const digestsMaxResEnv = "H3_CONFORMANCE_DIGESTS_MAXRES"

// defaultDigestsMaxRes is the finest resolution checked without the
// environment variable.
const defaultDigestsMaxRes = 3

// polygonsMaxRes is the finest resolution that has the polygons stream, and
// polygonsFillStep is how many resolutions finer each polygon is filled, as
// README.md fixes them.
const (
	polygonsMaxRes   = 2
	polygonsFillStep = 2
)

// polygonFactors scale a cell boundary about its center for the polygons
// stream, and containmentModes are the fills taken of each, in line order.
var (
	polygonFactors   = []float64{0.8, 1.2}     //nolint:gochecknoglobals // fixed by the format
	containmentModes = []h3go.ContainmentMode{ //nolint:gochecknoglobals // fixed by the format
		h3go.ContainmentCenter, h3go.ContainmentFull,
		h3go.ContainmentOverlapping, h3go.ContainmentOverlappingBbox,
	}
)

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

// baseCellDigests is one base cell's share of a resolution: its cell count
// and its stream digests.
type baseCellDigests struct {
	count   int64
	digests map[string]string
}

// digestBaseCell enumerates the cells of res under base, sorted, and runs
// every stream over them.
func digestBaseCell(base h3go.Cell, res int) (baseCellDigests, error) {
	cells, err := base.Children(res)
	if err != nil {
		return baseCellDigests{}, fmt.Errorf("Children(%s, %d): %w", base, res, err)
	}

	slices.Sort(cells)

	digests, err := digestStreams(cells, res)
	if err != nil {
		return baseCellDigests{}, err
	}

	return baseCellDigests{count: int64(len(cells)), digests: digests}, nil
}

// TestResolutionDigests derives each resolution's digests from its 122 base
// cells, computed in parallel, and compares them with the record.
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

		parts := make([]baseCellDigests, len(res0))
		errs := make([]error, len(res0))
		slots := make(chan struct{}, runtime.GOMAXPROCS(0))

		var wait sync.WaitGroup

		for i, base := range res0 {
			wait.Add(1)

			go func() {
				defer wait.Done()

				slots <- struct{}{}
				defer func() { <-slots }()

				parts[i], errs[i] = digestBaseCell(base, record.Res)
			}()
		}

		wait.Wait()

		for i, err := range errs {
			if err != nil {
				t.Fatalf("base cell %d: %v", i, err)
			}
		}

		count, got := combineBaseCells(parts, record.Res)
		compareDigests(t, count, got, record.Count, record.Digests)
	})
}

// combineBaseCells derives a resolution's count and digests from its base
// cells in order: per stream, the SHA-256 of the 122 hex digests
// concatenated.
func combineBaseCells(parts []baseCellDigests, res int) (int64, map[string]string) {
	var count int64

	hashes := map[string]hash.Hash{}
	for _, name := range streamsAt(res) {
		hashes[name] = sha256.New()
	}

	for _, part := range parts {
		count += part.count

		for name, h := range hashes {
			h.Write([]byte(part.digests[name]))
		}
	}

	out := make(map[string]string, len(hashes))
	for name, h := range hashes {
		out[name] = hex.EncodeToString(h.Sum(nil))
	}

	return count, out
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

		part, err := digestBaseCell(base, record.Res)
		if err != nil {
			t.Fatal(err)
		}

		compareDigests(t, part.count, part.digests, record.Count, record.Digests)
	})
}

// compareDigests compares a computed count and digests with a record's.
func compareDigests(t *testing.T, count int64, got map[string]string, wantCount int64, want map[string]string) {
	t.Helper()

	if count != wantCount {
		t.Errorf("enumerated %d cells, want %d", count, wantCount)
	}

	if len(got) != len(want) {
		t.Errorf("record has %d streams, runner computes %d", len(want), len(got))
	}

	for name, digest := range got {
		if digest != want[name] {
			t.Errorf("%s digest = %s, want %s", name, digest, want[name])
		}
	}
}

// streams lists the digest streams in key order; polygons exists only up
// to polygonsMaxRes.
var streams = []string{ //nolint:gochecknoglobals // fixed by the format
	"cells", "children", "compact", "disks", "distances", "edges",
	"inspection", "localIj", "parents", "polygons", "rings", "roundTrip", "vertexes",
}

// streamsAt returns the stream names that exist at a resolution.
func streamsAt(res int) []string {
	if res <= polygonsMaxRes {
		return streams
	}

	return slices.DeleteFunc(slices.Clone(streams), func(name string) bool { return name == "polygons" })
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
// digests by stream name. A call the stream definition does not allow to
// fail is reported as an error rather than a panic.
func digestStreams(cells []h3go.Cell, res int) (digests map[string]string, err error) {
	names := streamsAt(res)

	writer := &lineWriter{hashes: map[string]hash.Hash{}}
	for _, name := range names {
		writer.hashes[name] = sha256.New()
	}

	defer func() {
		if failure := recover(); failure != nil {
			digests, err = nil, fmt.Errorf("stream computation failed: %v", failure)
		}
	}()

	for _, cell := range cells {
		digestCell(writer, cell, res)
	}

	out := make(map[string]string, len(names))
	for name, h := range writer.hashes {
		out[name] = hex.EncodeToString(h.Sum(nil))
	}

	return out, nil
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

	if res <= polygonsMaxRes {
		w.start(cell)

		for factorIndex, factor := range polygonFactors {
			polygon := scaledPolygon(cell, factor)

			for modeIndex, mode := range containmentModes {
				if factorIndex > 0 || modeIndex > 0 {
					w.field("|")
				}

				w.cellSet(must(h3go.PolygonToCellsExperimental(polygon, res+polygonsFillStep, mode)))
			}
		}

		w.finish("polygons")
	}
}

// scaledPolygon returns the cell's boundary scaled about its center by
// factor, computed in degrees as README.md "regions/polygons.jsonl" states.
func scaledPolygon(cell h3go.Cell, factor float64) h3go.GeoPolygon {
	center := must(cell.LatLng())
	boundary := must(cell.Boundary())
	loop := make(h3go.GeoLoop, len(boundary))

	for i, vertex := range boundary {
		loop[i] = h3go.LatLng{
			Lat: center.Lat + factor*(vertex.Lat-center.Lat),
			Lng: wrapDegrees(center.Lng + factor*wrapDegrees(vertex.Lng-center.Lng)),
		}
	}

	return h3go.GeoPolygon{GeoLoop: loop}
}

// wrapDegrees brings a longitude difference back into [-180, 180].
func wrapDegrees(lng float64) float64 {
	switch {
	case lng > 180:
		return lng - 360
	case lng < -180:
		return lng + 360
	default:
		return lng
	}
}

// must panics on an error from a call that the stream definition does not
// allow to fail; digestStreams turns the panic into an error.
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
