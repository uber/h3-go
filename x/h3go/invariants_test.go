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

// This file checks properties that hold for every cell of a resolution and
// need no reference implementation to state: the cells of a resolution
// partition the sphere, neighbouring cells agree on the vertices and edges
// they share, and grid distance behaves as a metric. Each test enumerates
// every cell up to a resolution, so a failure names the cell and nothing is
// inferred from a sample.

import (
	"errors"
	"math"
	"os"
	"strconv"
	"testing"
)

const (
	// invariantsMaxResEnv names the environment variable that raises the
	// finest resolution the invariant tests enumerate.
	invariantsMaxResEnv = "H3GO_INVARIANTS_MAXRES"

	// invariantsDefaultMaxRes is the finest resolution enumerated without the
	// environment variable: 5,882 cells, about a second under the race
	// detector on two cores. Each resolution finer costs seven times as
	// much, and the shared CI runners are an order of magnitude slower than
	// a workstation with the whole package running in parallel.
	invariantsDefaultMaxRes = 2

	// sphereAreaRads2 is the area of the unit sphere, which the cells of any
	// one resolution tile exactly.
	sphereAreaRads2 = 4 * math.Pi

	// areaPartitionTolerance bounds the relative error of the summed cell
	// areas against the sphere. The sum is compensated, so this measures the
	// areas and not the adder.
	areaPartitionTolerance = 1e-9

	// sharedPointBoundDeg bounds, in degrees of arc, how far two cells may
	// place a point they share. A face crossing between them evaluates the
	// same point through two different face charts, which is where the last
	// bits differ.
	sharedPointBoundDeg = 1e-11

	// lengthRelativeTolerance bounds the relative difference between two
	// evaluations of the same length; lengthFloorRads is the absolute floor
	// below which the relative bound is not applied, the shared-point bound
	// in radians.
	lengthRelativeTolerance = 1e-9
	lengthFloorRads         = sharedPointBoundDeg * DegsToRads

	// gridMetricK is the disk radius the grid-metric test checks around every
	// cell.
	gridMetricK = 3
)

// invariantsMaxRes returns the finest resolution to enumerate.
func invariantsMaxRes(t *testing.T) int {
	t.Helper()

	raw := os.Getenv(invariantsMaxResEnv)
	if raw == "" {
		return invariantsDefaultMaxRes
	}

	maxRes, err := strconv.Atoi(raw)
	if err != nil || maxRes < 0 || maxRes > MaxResolution {
		t.Fatalf("%s = %q, want 0..%d", invariantsMaxResEnv, raw, MaxResolution)
	}

	return maxRes
}

// forEachResolution runs check once per resolution from 0 to the enumerated
// maximum, as a parallel subtest named by the resolution.
func forEachResolution(t *testing.T, check func(t *testing.T, res int)) {
	t.Helper()

	maxRes := invariantsMaxRes(t)

	for res := 0; res <= maxRes; res++ {
		t.Run("res"+strconv.Itoa(res), func(t *testing.T) {
			t.Parallel()
			check(t, res)
		})
	}
}

// cellsOfResolution returns every cell at res, grouped by base cell.
func cellsOfResolution(t *testing.T, res int) [][]Cell {
	t.Helper()

	baseCells, err := Res0Cells()
	if err != nil {
		t.Fatal(err)
	}

	groups := make([][]Cell, len(baseCells))

	for i, base := range baseCells {
		groups[i], err = base.Children(res)
		if err != nil {
			t.Fatal(err)
		}
	}

	return groups
}

// compensatedSum accumulates float64 values with Neumaier's correction, so
// that summing millions of small areas loses nothing to the running total's
// rounding.
type compensatedSum struct {
	sum, correction float64
}

// add accumulates one value.
func (s *compensatedSum) add(value float64) {
	total := s.sum + value

	if math.Abs(s.sum) >= math.Abs(value) {
		s.correction += (s.sum - total) + value
	} else {
		s.correction += (value - total) + s.sum
	}

	s.sum = total
}

// total returns the corrected sum.
func (s *compensatedSum) total() float64 {
	return s.sum + s.correction
}

// TestAreaPartition sums the area of every cell at each resolution and
// requires the total to be the area of the sphere. Children do not tile their
// parent, so there is no finer closed form than the whole sphere; the
// per-cell areas are compared with the reference by the conformance suite.
func TestAreaPartition(t *testing.T) {
	t.Parallel()

	forEachResolution(t, func(t *testing.T, res int) {
		t.Helper()

		var total compensatedSum

		for _, cells := range cellsOfResolution(t, res) {
			var partial compensatedSum

			for _, cell := range cells {
				area, err := CellAreaRads2(cell)
				if err != nil {
					t.Fatalf("CellAreaRads2(%s): %v", cell, err)
				}

				partial.add(area)
			}

			total.add(partial.total())
		}

		relative := math.Abs(total.total()-sphereAreaRads2) / sphereAreaRads2
		t.Logf("sum of cell areas differs from 4π by %.3g relative", relative)

		if relative > areaPartitionTolerance {
			t.Errorf("sum of cell areas at res %d = %.17g, want 4π within %.1g relative",
				res, total.total(), areaPartitionTolerance)
		}
	})
}

// separationDegLatLng returns the angular separation of two points in
// degrees of arc.
func separationDegLatLng(a, b LatLng) float64 {
	return GreatCircleDistanceRads(a, b) * RadsToDegs
}

// nearestBoundaryPointDeg returns the smallest angular separation, in
// degrees of arc, between point and any point of boundary.
func nearestBoundaryPointDeg(point LatLng, boundary CellBoundary) float64 {
	nearest := math.Inf(1)

	for _, candidate := range boundary {
		if sep := separationDegLatLng(point, candidate); sep < nearest {
			nearest = sep
		}
	}

	return nearest
}

// TestSharedVertices requires every cell's boundary to pass through the
// point of each vertex it has. A vertex's point comes from its owning cell's
// boundary, so for the two or three other cells that meet there this is the
// check that neighbours agree on where their common corner is.
func TestSharedVertices(t *testing.T) {
	t.Parallel()

	forEachResolution(t, func(t *testing.T, res int) {
		t.Helper()

		worst := 0.0
		worstCell := Cell(0)

		for _, cells := range cellsOfResolution(t, res) {
			for _, cell := range cells {
				boundary, err := cell.Boundary()
				if err != nil {
					t.Fatalf("Boundary(%s): %v", cell, err)
				}

				vertexes, err := cell.Vertexes()
				if err != nil {
					t.Fatalf("Vertexes(%s): %v", cell, err)
				}

				if want := vertexCount(cell); len(vertexes) != want {
					t.Fatalf("Vertexes(%s) has %d vertexes, want %d", cell, len(vertexes), want)
				}

				for _, vertex := range vertexes {
					point, err := VertexToLatLng(vertex)
					if err != nil {
						t.Fatalf("VertexToLatLng(%s): %v", vertex, err)
					}

					if sep := nearestBoundaryPointDeg(point, boundary); sep > worst {
						worst, worstCell = sep, cell
					}
				}
			}
		}

		t.Logf("worst vertex separation %.3g deg at %s", worst, worstCell)

		if worst > sharedPointBoundDeg {
			t.Errorf("a vertex of %s is %.3g deg from its boundary, bound %.1g",
				worstCell, worst, sharedPointBoundDeg)
		}
	})
}

// vertexCount is the number of vertices a cell has.
func vertexCount(cell Cell) int {
	if cell.IsPentagon() {
		return numPentVerts
	}

	return numHexVerts
}

// lengthsAgree reports whether two evaluations of one length in radians are
// within the relative tolerance, or within the absolute floor.
func lengthsAgree(a, b float64) bool {
	diff := math.Abs(a - b)

	return diff <= lengthFloorRads || diff <= lengthRelativeTolerance*math.Max(math.Abs(a), math.Abs(b))
}

// perimeterRads is the length of a closed boundary in radians.
func perimeterRads(boundary CellBoundary) float64 {
	var perimeter compensatedSum

	for i := range boundary {
		perimeter.add(GreatCircleDistanceRads(boundary[i], boundary[(i+1)%len(boundary)]))
	}

	return perimeter.total()
}

// TestEdgeConsistency requires every directed edge to have the length of its
// reverse, which the neighbour computes from its own boundary, and the
// lengths of a cell's edges to sum to the length of its boundary.
func TestEdgeConsistency(t *testing.T) {
	t.Parallel()

	forEachResolution(t, func(t *testing.T, res int) {
		t.Helper()

		for _, cells := range cellsOfResolution(t, res) {
			for _, cell := range cells {
				edges, err := cell.DirectedEdges()
				if err != nil {
					t.Fatalf("DirectedEdges(%s): %v", cell, err)
				}

				var sum compensatedSum

				for _, edge := range edges {
					length, err := EdgeLengthRads(edge)
					if err != nil {
						t.Fatalf("EdgeLengthRads(%s): %v", edge, err)
					}

					reverse, err := edge.Reverse()
					if err != nil {
						t.Fatalf("Reverse(%s): %v", edge, err)
					}

					reverseLength, err := EdgeLengthRads(reverse)
					if err != nil {
						t.Fatalf("EdgeLengthRads(%s): %v", reverse, err)
					}

					if !lengthsAgree(length, reverseLength) {
						t.Errorf("EdgeLengthRads(%s) = %.17g, reverse %s = %.17g", edge, length, reverse, reverseLength)
					}

					sum.add(length)
				}

				boundary, err := cell.Boundary()
				if err != nil {
					t.Fatalf("Boundary(%s): %v", cell, err)
				}

				if perimeter := perimeterRads(boundary); !lengthsAgree(sum.total(), perimeter) {
					t.Errorf("edges of %s sum to %.17g rad, boundary perimeter %.17g", cell, sum.total(), perimeter)
				}
			}
		}
	})
}

// gridDistanceOrDeclined returns the grid distance between two cells, or -1
// when the library declines the pair with ErrFailed, which the reference
// documents for cells too far apart or separated by pentagon distortion.
// Any other error is a failure.
func gridDistanceOrDeclined(t *testing.T, a, b Cell) int {
	t.Helper()

	distance, err := a.GridDistance(b)
	if errors.Is(err, ErrFailed) {
		return -1
	}

	if err != nil {
		t.Fatalf("GridDistance(%s, %s): %v", a, b, err)
	}

	return distance
}

// checkGridPath requires the path from origin to target to have one more
// cell than the distance, to start and end at its endpoints, and to step
// between neighbours. The distance succeeded, so the path must too.
func checkGridPath(t *testing.T, origin, target Cell, distance int) {
	t.Helper()

	path, err := origin.GridPath(target)
	if err != nil {
		t.Fatalf("GridPath(%s, %s): %v", origin, target, err)
	}

	if len(path) != distance+1 || path[0] != origin || path[len(path)-1] != target {
		t.Errorf("GridPath(%s, %s) = %v, want %d cells from origin to target", origin, target, path, distance+1)

		return
	}

	for i := 1; i < len(path); i++ {
		neighbor, err := path[i-1].IsNeighbor(path[i])
		if err != nil || !neighbor {
			t.Errorf("GridPath(%s, %s) steps from %s to %s, which are not neighbours", origin, target, path[i-1], path[i])
		}
	}
}

// gridMetricDeclinedMinRes is the first resolution at which the library is
// expected to decline a pair only for pentagon distortion. At resolutions 0
// and 1 a 3-disk spans several icosahedron faces and pairs are also declined
// as too far apart.
const gridMetricDeclinedMinRes = 2

// TestGridMetric requires, around every cell, that grid distance to each
// member of its 3-disk equals the ring the member was found on, that the
// pair is declined in both directions or neither, that a path realises every
// distance, and that within the 1-disk distance is zero exactly on the
// diagonal and never exceeds two. From resolution 2 a declined pair must
// have a pentagon within the disk.
func TestGridMetric(t *testing.T) {
	t.Parallel()

	forEachResolution(t, func(t *testing.T, res int) {
		t.Helper()

		declined := 0

		for _, cells := range cellsOfResolution(t, res) {
			for _, origin := range cells {
				declined += checkGridMetricAround(t, origin, res >= gridMetricDeclinedMinRes)
			}
		}

		t.Logf("%d pairs declined", declined)
	})
}

// checkGridMetricAround runs the grid-metric checks for one origin and
// returns the number of pairs the library declined. When strict, a declined
// pair must have a pentagon in the origin's disk.
func checkGridMetricAround(t *testing.T, origin Cell, strict bool) int {
	t.Helper()

	rings, err := origin.GridDiskDistances(gridMetricK)
	if err != nil {
		t.Fatalf("GridDiskDistances(%s, %d): %v", origin, gridMetricK, err)
	}

	nearPentagon := false

	for _, ring := range rings {
		for _, member := range ring {
			nearPentagon = nearPentagon || member.IsPentagon()
		}
	}

	declined := 0

	distance := func(a, b Cell) int {
		forward := gridDistanceOrDeclined(t, a, b)
		backward := gridDistanceOrDeclined(t, b, a)

		if forward != backward {
			t.Errorf("GridDistance(%s, %s) = %d but GridDistance(%s, %s) = %d", a, b, forward, b, a, backward)
		}

		if forward < 0 {
			declined++

			if strict && !nearPentagon {
				t.Errorf("GridDistance(%s, %s) declined with no pentagon within %d of %s", a, b, gridMetricK, origin)
			}
		}

		return forward
	}

	for k, ring := range rings {
		for _, member := range ring {
			found := distance(origin, member)
			if found < 0 {
				continue
			}

			if found != k {
				t.Errorf("GridDistance(%s, %s) = %d, found on ring %d", origin, member, found, k)
			}

			checkGridPath(t, origin, member, found)
		}
	}

	disk1 := append(append([]Cell{}, rings[0]...), rings[1]...)

	for i, a := range disk1 {
		for _, b := range disk1[i:] {
			found := distance(a, b)
			if found < 0 {
				continue
			}

			if (found == 0) != (a == b) || found > 2 {
				t.Errorf("GridDistance(%s, %s) = %d, both within 1 of %s", a, b, found, origin)
			}
		}
	}

	return declined
}
