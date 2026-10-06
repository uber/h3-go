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
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"strconv"

	"github.com/uber/h3-go/v4/x/h3go"
)

// FormatVersion identifies the suite layout and record semantics described in
// README.md. A consumer that does not recognize the version must not run the
// suite.
const FormatVersion = "h3-conformance/1"

// ManifestFile is the name of the manifest at the root of a suite directory.
const ManifestFile = "manifest.json"

// Manifest pins a suite to an H3 version and lists every file in it. It is
// the only file a consumer reads by name; everything else is discovered
// through Files.
//
//nolint:govet // field order is the JSON key order given in README.md
type Manifest struct {
	// Format is FormatVersion.
	Format string `json:"format"`
	// H3Version is the H3 release the expected values were produced by,
	// without a leading "v".
	H3Version string `json:"h3Version"`
	// Generator records how the inputs were chosen.
	Generator Generator `json:"generator"`
	// Tolerances are the comparison bounds for floating-point outputs. They
	// apply to every file in the suite.
	Tolerances Tolerances `json:"tolerances"`
	// Files maps each record file, as a slash-separated path relative to the
	// manifest, to its size and hash.
	Files map[string]FileInfo `json:"files"`
}

// Generator names the program that produced the suite and the seed its
// sampler ran from, so another generator can reproduce the same inputs.
type Generator struct {
	Name string `json:"name"`
	Seed uint64 `json:"seed"`
}

// Tolerances are the bounds within which a floating-point output conforms.
// Coordinates are compared by angular distance on the sphere, in degrees of
// arc; lengths and areas by relative error.
type Tolerances struct {
	AngularDeg float64 `json:"angularDeg"`
	Relative   float64 `json:"relative"`
}

// FileInfo describes one record file.
//
//nolint:govet // field order is the JSON key order given in README.md
type FileInfo struct {
	// Records is the number of lines in the file.
	Records int `json:"records"`
	// SHA256 is the lowercase hex SHA-256 of the file's bytes.
	SHA256 string `json:"sha256"`
}

// Result is a function output that is either a value or an H3 error. It
// encodes as the value itself on success and as {"err": "<H3Error name>"}
// on failure.
type Result[T any] struct {
	Value T
	Err   string
}

// errorObject is the JSON shape of a failed Result.
type errorObject struct {
	Err string `json:"err"`
}

// MarshalJSON encodes the value, or the error object when Err is set.
func (r Result[T]) MarshalJSON() ([]byte, error) {
	if r.Err != "" {
		return json.Marshal(errorObject{Err: r.Err})
	}

	return json.Marshal(r.Value)
}

// UnmarshalJSON decodes either an error object or a value.
func (r *Result[T]) UnmarshalJSON(data []byte) error {
	var obj errorObject
	if err := json.Unmarshal(data, &obj); err == nil && obj.Err != "" {
		*r = Result[T]{Err: obj.Err}

		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	*r = Result[T]{Value: value}

	return nil
}

// InspectionRecord holds the inspection-function outputs for one 64-bit index.
// The index may be a cell, a directed edge, a vertex, or an invalid bit
// pattern; every field is defined for all of them.
//
//nolint:govet // field order is the JSON key order given in README.md
type InspectionRecord struct {
	// Index is the canonical string form of the index: lowercase hex with no
	// leading zeros, as h3ToString produces it.
	Index string `json:"index"`
	// Res is the resolution field.
	Res int `json:"res"`
	// BaseCell is the base cell field.
	BaseCell int `json:"baseCell"`
	// ValidCell is isValidCell.
	ValidCell bool `json:"validCell"`
	// ValidIndex is isValidIndex.
	ValidIndex bool `json:"validIndex"`
	// ResClassIII is isResClassIII.
	ResClassIII bool `json:"resClassIII"`
	// Pentagon is isPentagon.
	Pentagon bool `json:"pentagon"`
	// Faces is getIcosahedronFaces as a set, stored in ascending order.
	Faces Result[[]int] `json:"faces"`
	// Digits is getIndexDigit for every resolution from 1 to 15, including
	// the digits beyond Res.
	Digits []int `json:"digits"`
	// Construct is constructCell called with Res, BaseCell and the first Res
	// entries of Digits, as a canonical index string.
	Construct Result[string] `json:"construct"`
}

// HierarchyRecord holds the parent and child relations of one valid cell.
//
//nolint:govet // field order is the JSON key order given in README.md
type HierarchyRecord struct {
	// Index is the subject cell.
	Index string `json:"index"`
	// Parents is cellToParent for every resolution from 0 to the cell's
	// resolution minus one, in that order.
	Parents []string `json:"parents"`
	// CenterChild is cellToCenterChild one resolution finer.
	CenterChild Result[string] `json:"centerChild"`
	// Children is cellToChildren one resolution finer, as a set.
	Children Result[[]string] `json:"children"`
	// ChildPos is cellToChildPos against the resolution-0 ancestor.
	ChildPos Result[int64] `json:"childPos"`
}

// TraversalRecord holds the grid-traversal outputs of one valid cell and of
// the cell paired with a target drawn from its 3-disk.
//
//nolint:govet // field order is the JSON key order given in README.md
type TraversalRecord struct {
	// Index is the subject cell.
	Index string `json:"index"`
	// Disk1, Disk2 and Disk3 are gridDisk for k 1 to 3, as sets.
	Disk1 []string `json:"disk1"`
	Disk2 []string `json:"disk2"`
	Disk3 []string `json:"disk3"`
	// Ring1, Ring2 and Ring3 are gridRing for k 1 to 3, as sets.
	Ring1 []string `json:"ring1"`
	Ring2 []string `json:"ring2"`
	Ring3 []string `json:"ring3"`
	// DiskDistances2 is gridDiskDistances for k 2, one set per distance from
	// 0 to 2.
	DiskDistances2 [][]string `json:"diskDistances2"`
	// Target is the second cell of the pair.
	Target string `json:"target"`
	// Distance is gridDistance(index, target).
	Distance Result[int64] `json:"distance"`
	// Path is gridPathCells(index, target), as a sequence.
	Path Result[[]string] `json:"path"`
	// Neighbor is areNeighborCells(index, target).
	Neighbor Result[bool] `json:"neighbor"`
}

// EdgesRecord holds the directed-edge outputs of one valid cell.
//
//nolint:govet // field order is the JSON key order given in README.md
type EdgesRecord struct {
	// Index is the subject cell.
	Index string `json:"index"`
	// Edges is originToDirectedEdges, as a set.
	Edges []string `json:"edges"`
	// Destinations is getDirectedEdgeDestination for each entry of Edges in
	// its written (ascending) order.
	Destinations []string `json:"destinations"`
	// Target is a cell drawn from the subject's 1-disk.
	Target string `json:"target"`
	// Edge is cellsToDirectedEdge(index, target).
	Edge Result[string] `json:"edge"`
}

// VertexesRecord holds the vertex outputs of one valid cell.
type VertexesRecord struct {
	// Index is the subject cell.
	Index string `json:"index"`
	// Vertexes is cellToVertexes, as a set.
	Vertexes []string `json:"vertexes"`
	// ByNumber is cellToVertex for vertex numbers 0 to 5, in that order.
	ByNumber []Result[string] `json:"byNumber"`
}

// CoordIJ is a local IJ coordinate pair.
type CoordIJ struct {
	I int `json:"i"`
	J int `json:"j"`
}

// LocalIJRecord holds the local IJ outputs of one origin and target pair.
type LocalIJRecord struct {
	// Index is the origin cell.
	Index string `json:"index"`
	// Target is the cell being located.
	Target string `json:"target"`
	// IJ is cellToLocalIj(index, target).
	IJ Result[CoordIJ] `json:"ij"`
	// Cell is localIjToCell(index, ij) when IJ succeeded, else IJ's error.
	Cell Result[string] `json:"cell"`
}

// SetsRecord holds compactCells and uncompactCells for one input set.
//
//nolint:govet // field order is the JSON key order given in README.md
type SetsRecord struct {
	// ID names the set: its kind and the cell it was built from.
	ID string `json:"id"`
	// Input is the set passed to both functions, in order, duplicates and
	// invalid entries included.
	Input []string `json:"input"`
	// Compact is compactCells(input), as a set.
	Compact Result[[]string] `json:"compact"`
	// UncompactRes is the resolution passed to uncompactCells.
	UncompactRes int `json:"uncompactRes"`
	// Uncompact is uncompactCells(input, uncompactRes), as a set.
	Uncompact Result[[]string] `json:"uncompact"`
}

// GeoCoord is a point as a [latitude, longitude] pair in degrees.
type GeoCoord [2]float64

// Polygon is an outer loop with zero or more holes, each a sequence of points.
type Polygon struct {
	Outer []GeoCoord   `json:"outer"`
	Holes [][]GeoCoord `json:"holes"`
}

// RegionsRecord holds the polyfill outputs for one polygon at one resolution
// and the outline of the classic fill.
//
//nolint:govet // field order is the JSON key order given in README.md
type RegionsRecord struct {
	// ID names the polygon: the cell whose boundary it is, the fill
	// resolution, and ":hole" when the center child is cut out.
	ID string `json:"id"`
	// Polygon is the input.
	Polygon Polygon `json:"polygon"`
	// Res is the fill resolution.
	Res int `json:"res"`
	// Cells is polygonToCells with flags 0, as a set.
	Cells Result[[]string] `json:"cells"`
	// Center, Full, Overlapping and OverlappingBbox are
	// polygonToCellsExperimental in containment modes 0 to 3, as sets.
	Center          Result[[]string] `json:"center"`
	Full            Result[[]string] `json:"full"`
	Overlapping     Result[[]string] `json:"overlapping"`
	OverlappingBbox Result[[]string] `json:"overlappingBbox"`
	// MultiPolygon is cellsToLinkedMultiPolygon(cells), or Cells' error.
	MultiPolygon Result[[]Polygon] `json:"multiPolygon"`
}

// ResolutionDigestsRecord holds the digests over every cell of one
// resolution, keyed by stream name as README.md defines them.
//
//nolint:govet // field order is the JSON key order given in README.md
type ResolutionDigestsRecord struct {
	// Res is the resolution whose cells are enumerated.
	Res int `json:"res"`
	// Count is the number of cells at Res.
	Count int64 `json:"count"`
	// Digests maps each stream to its lowercase hex SHA-256.
	Digests map[string]string `json:"digests"`
}

// BaseCellDigestsRecord holds the same digests restricted to the cells of
// one base cell, so a mismatch is localised to that subtree.
//
//nolint:govet // field order is the JSON key order given in README.md
type BaseCellDigestsRecord struct {
	// Res is the resolution whose cells are enumerated.
	Res int `json:"res"`
	// BaseCell is the base cell whose descendants are enumerated.
	BaseCell int `json:"baseCell"`
	// Count is the number of cells at Res under BaseCell.
	Count int64 `json:"count"`
	// Digests maps each stream to its lowercase hex SHA-256.
	Digests map[string]string `json:"digests"`
}

// PatternsDigestsRecord holds the validity digest over the first Count
// patterns of the bit-pattern stream.
//
//nolint:govet // field order is the JSON key order given in README.md
type PatternsDigestsRecord struct {
	// Count is how many patterns the digest covers.
	Count int64 `json:"count"`
	// Digests maps the single stream, "validity", to its lowercase hex SHA-256.
	Digests map[string]string `json:"digests"`
}

// subject names the record by its pattern count.
func (r PatternsDigestsRecord) subject() string { return strconv.FormatInt(r.Count, 10) }

// FloatCellRecord holds the coordinate and area outputs of one valid cell,
// plus latLngToCell on points inside it.
//
//nolint:govet // field order is the JSON key order given in README.md
type FloatCellRecord struct {
	// Index is the subject cell.
	Index string `json:"index"`
	// Center is cellToLatLng.
	Center GeoCoord `json:"center"`
	// Boundary is cellToBoundary, as a sequence.
	Boundary []GeoCoord `json:"boundary"`
	// AreaRads2, AreaKm2 and AreaM2 are the cellArea functions.
	AreaRads2 float64 `json:"areaRads2"`
	AreaKm2   float64 `json:"areaKm2"`
	AreaM2    float64 `json:"areaM2"`
	// InteriorRes is the resolution the interior points are mapped at.
	InteriorRes int `json:"interiorRes"`
	// Interior holds one point halfway from the center to each boundary
	// vertex.
	Interior []GeoCoord `json:"interior"`
	// InteriorCells is latLngToCell(point, interiorRes) for each point.
	InteriorCells []string `json:"interiorCells"`
}

// FloatEdgeRecord holds the length and boundary of one directed edge.
//
//nolint:govet // field order is the JSON key order given in README.md
type FloatEdgeRecord struct {
	// Index is the subject edge.
	Index string `json:"index"`
	// LengthRads, LengthKm and LengthM are the edgeLength functions.
	LengthRads float64 `json:"lengthRads"`
	LengthKm   float64 `json:"lengthKm"`
	LengthM    float64 `json:"lengthM"`
	// Boundary is directedEdgeToBoundary, as a sequence.
	Boundary []GeoCoord `json:"boundary"`
}

// FloatVertexRecord holds the position of one vertex.
type FloatVertexRecord struct {
	// Index is the subject vertex.
	Index string `json:"index"`
	// LatLng is vertexToLatLng.
	LatLng GeoCoord `json:"latLng"`
}

// DistanceRecord holds the great-circle distance between two cell centers.
type DistanceRecord struct {
	// Index and Target are the cells whose centers are A and B.
	Index  string `json:"index"`
	Target string `json:"target"`
	// A and B are the points the distance is measured between.
	A GeoCoord `json:"a"`
	B GeoCoord `json:"b"`
	// Rads, Km and M are the greatCircleDistance functions of A and B.
	Rads float64 `json:"rads"`
	Km   float64 `json:"km"`
	M    float64 `json:"m"`
}

// FloatResolutionRecord holds the average-measure functions of one
// resolution.
//
//nolint:govet // field order is the JSON key order given in README.md
type FloatResolutionRecord struct {
	// Res is the resolution; 16 is the error row.
	Res int `json:"res"`
	// HexagonAreaKm2 and HexagonAreaM2 are getHexagonAreaAvg.
	HexagonAreaKm2 Result[float64] `json:"hexagonAreaKm2"`
	HexagonAreaM2  Result[float64] `json:"hexagonAreaM2"`
	// EdgeLengthKm and EdgeLengthM are getHexagonEdgeLengthAvg.
	EdgeLengthKm Result[float64] `json:"edgeLengthKm"`
	EdgeLengthM  Result[float64] `json:"edgeLengthM"`
}

// subject is implemented by every record type and names the record's input,
// which identifies the record within its file.
type subject interface {
	subject() string
}

// subject returns the record's index.
func (r InspectionRecord) subject() string { return r.Index }

// subject returns the record's cell.
func (r HierarchyRecord) subject() string { return r.Index }

// subject returns the record's cell and target.
func (r TraversalRecord) subject() string { return r.Index + "/" + r.Target }

// subject returns the record's cell.
func (r EdgesRecord) subject() string { return r.Index }

// subject returns the record's cell.
func (r VertexesRecord) subject() string { return r.Index }

// subject returns the record's origin and target.
func (r LocalIJRecord) subject() string { return r.Index + "/" + r.Target }

// subject returns the set's id.
func (r SetsRecord) subject() string { return r.ID }

// subject returns the polygon's id.
func (r RegionsRecord) subject() string { return r.ID }

// subject returns the resolution.
func (r ResolutionDigestsRecord) subject() string { return strconv.Itoa(r.Res) }

// subject returns the record's cell.
func (r FloatCellRecord) subject() string { return r.Index }

// subject returns the record's edge.
func (r FloatEdgeRecord) subject() string { return r.Index }

// subject returns the record's vertex.
func (r FloatVertexRecord) subject() string { return r.Index }

// subject returns the record's origin and target.
func (r DistanceRecord) subject() string { return r.Index + "/" + r.Target }

// subject returns the resolution.
func (r FloatResolutionRecord) subject() string { return strconv.Itoa(r.Res) }

// subject returns the resolution and base cell.
func (r BaseCellDigestsRecord) subject() string {
	return strconv.Itoa(r.Res) + "/" + strconv.Itoa(r.BaseCell)
}

// errorNames maps the h3go errors to the H3Error enum names used in record
// files.
var errorNames = map[error]string{
	h3go.ErrFailed:                "E_FAILED",
	h3go.ErrDomain:                "E_DOMAIN",
	h3go.ErrLatLngDomain:          "E_LATLNG_DOMAIN",
	h3go.ErrResolutionDomain:      "E_RES_DOMAIN",
	h3go.ErrCellInvalid:           "E_CELL_INVALID",
	h3go.ErrDirectedEdgeInvalid:   "E_DIR_EDGE_INVALID",
	h3go.ErrUndirectedEdgeInvalid: "E_UNDIR_EDGE_INVALID",
	h3go.ErrVertexInvalid:         "E_VERTEX_INVALID",
	h3go.ErrPentagon:              "E_PENTAGON",
	h3go.ErrDuplicateInput:        "E_DUPLICATE_INPUT",
	h3go.ErrNotNeighbors:          "E_NOT_NEIGHBORS",
	h3go.ErrResolutionMismatch:    "E_RES_MISMATCH",
	h3go.ErrMemoryAlloc:           "E_MEMORY_ALLOC",
	h3go.ErrMemoryBounds:          "E_MEMORY_BOUNDS",
	h3go.ErrOptionInvalid:         "E_OPTION_INVALID",
	h3go.ErrIndexInvalid:          "E_INDEX_INVALID",
	h3go.ErrBaseCellDomain:        "E_BASE_CELL_DOMAIN",
	h3go.ErrDigitDomain:           "E_DIGIT_DOMAIN",
	h3go.ErrDeletedDigit:          "E_DELETED_DIGIT",
}

// ErrorName returns the H3Error enum name of an h3go error, "" for nil, and
// "E_FAILED" for an error the package does not define.
func ErrorName(err error) string {
	if err == nil {
		return ""
	}

	for known, name := range errorNames {
		if errors.Is(err, known) {
			return name
		}
	}

	return errorNames[h3go.ErrFailed]
}

// Records decodes a newline-delimited JSON stream one record at a time. The
// sequence yields a non-nil error, with the 1-based line number in its
// message, for the first line that fails to decode and then stops.
func Records[T any](r io.Reader) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(nil, maxLineBytes)

		for line := 1; scanner.Scan(); line++ {
			var record T
			if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
				var zero T

				yield(zero, fmt.Errorf("line %d: %w", line, err))

				return
			}

			if !yield(record, nil) {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			var zero T

			yield(zero, err)
		}
	}
}

// maxLineBytes bounds a single record line. Set-valued outputs at high k or
// resolution can run to tens of thousands of indexes.
const maxLineBytes = 16 << 20

// WriteRecords encodes records as newline-delimited JSON, one per line, every
// line terminated, with no indentation.
func WriteRecords[T any](w io.Writer, records []T) error {
	encoder := json.NewEncoder(w)

	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	return nil
}

// DescribeFile counts the lines of a record file and hashes its bytes.
func DescribeFile(path string) (FileInfo, error) {
	data, err := os.ReadFile(path) //nolint:gosec // paths come from the suite manifest or the generator
	if err != nil {
		return FileInfo{}, err
	}

	sum := sha256.Sum256(data)

	return FileInfo{
		Records: bytes.Count(data, []byte{'\n'}),
		SHA256:  hex.EncodeToString(sum[:]),
	}, nil
}

// LoadManifest reads and validates the manifest in dir. It rejects an
// unrecognized format version and any listed file whose size or hash does not
// match the bytes on disk.
func LoadManifest(dir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestFile)) //nolint:gosec // dir is the suite directory
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("%s: %w", ManifestFile, err)
	}

	if manifest.Format != FormatVersion {
		return Manifest{}, fmt.Errorf("%s: format %q, want %q", ManifestFile, manifest.Format, FormatVersion)
	}

	for name, want := range manifest.Files {
		got, err := DescribeFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			return Manifest{}, err
		}

		if got != want {
			return Manifest{}, fmt.Errorf("%s: %d records, sha256 %s; manifest says %d, %s", name, got.Records, got.SHA256, want.Records, want.SHA256)
		}
	}

	return manifest, nil
}

// WriteManifest writes the manifest to dir with stable, indented formatting
// and a trailing newline.
func WriteManifest(dir string, manifest Manifest) error {
	file, err := os.Create(filepath.Join(dir, ManifestFile)) //nolint:gosec // dir is the suite directory
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return errors.Join(encoder.Encode(manifest), file.Close())
}
