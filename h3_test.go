/*
 * Copyright 2018 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"testing"
)

const eps = 1e-4

// validH3Index resolution 5.
const (
	validCell     = Cell(0x850dab63fffffff)
	pentagonCell  = Cell(0x821c07fffffffff)
	lineStartCell = Cell(0x89283082803ffff)
	lineEndCell   = Cell(0x8929a5653c3ffff)
)

var (
	validDiskDist3_1 = [][]Cell{
		{
			validCell,
		},
		{
			0x850dab73fffffff,
			0x850dab7bfffffff,
			0x850dab6bfffffff,
			0x850dab6ffffffff,
			0x850dab67fffffff,
			0x850dab77fffffff,
		},
		{
			0x850dab0bfffffff,
			0x850dab47fffffff,
			0x850dab4ffffffff,
			0x850d8cb7fffffff,
			0x850d8ca7fffffff,
			0x850d8dd3fffffff,
			0x850d8dd7fffffff,
			0x850d8d9bfffffff,
			0x850d8d93fffffff,
			0x850dab2bfffffff,
			0x850dab3bfffffff,
			0x850dab0ffffffff,
		},
	}

	validLatLng1 = LatLng{
		Lat: 67.1509268640,
		Lng: -168.3908885810,
	}
	validLatLng2 = LatLng{
		Lat: 37.775705522929044,
		Lng: -122.41812765598296,
	}

	// validGeoLoop is the boundary of validCell_1.
	validGeoLoop = GeoLoop{
		{Lat: 67.224749856, Lng: -168.523006585},
		{Lat: 67.140938355, Lng: -168.626914333},
		{Lat: 67.067252558, Lng: -168.494913285},
		{Lat: 67.077062918, Lng: -168.259695931},
		{Lat: 67.160561948, Lng: -168.154801171},
		{Lat: 67.234563187, Lng: -168.286102782},
	}

	validHole1 = GeoLoop{
		{Lat: 67.2, Lng: -168.4},
		{Lat: 67.1, Lng: -168.4},
		{Lat: 67.1, Lng: -168.3},
		{Lat: 67.2, Lng: -168.3},
	}

	validHole2 = GeoLoop{
		{Lat: 67.21, Lng: -168.41},
		{Lat: 67.22, Lng: -168.41},
		{Lat: 67.22, Lng: -168.42},
	}

	validGeoPolygonNoHoles = GeoPolygon{GeoLoop: validGeoLoop}

	validGeoPolygonHoles = GeoPolygon{
		GeoLoop: validGeoLoop,
		Holes: []GeoLoop{
			validHole1,
			validHole2,
		},
	}

	validEdge = DirectedEdge(0x1250dab73fffffff)
)

func TestLatLngToCell(t *testing.T) {
	t.Parallel()

	c, err := LatLngToCell(validLatLng1, 5)
	assertEqual(t, validCell, c)
	assertNoErr(t, err)

	_, err = LatLngToCell(NewLatLng(0, 0), MaxResolution+1)
	assertErr(t, err)
	assertErrIs(t, err, ErrResolutionDomain)
}

func TestCellToLatLng(t *testing.T) {
	t.Parallel()

	g, err := CellToLatLng(validCell)
	assertEqualLatLng(t, validLatLng1, g)
	assertNoErr(t, err)

	_, err = CellToLatLng(-1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestToCellBoundary(t *testing.T) {
	t.Parallel()

	boundary, err := validCell.Boundary()
	assertEqualLatLngs(t, validGeoLoop[:], boundary[:])
	assertNoErr(t, err)

	c := Cell(-1)
	_, err = c.Boundary()
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestCellToLocalIJ(t *testing.T) {
	t.Parallel()

	_, err := CellToLocalIJ(validCell, validCell)
	assertNoErr(t, err)

	_, err = CellToLocalIJ(-1, -1)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestGridDisk(t *testing.T) {
	t.Parallel()

	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()

		gd, err := validCell.GridDisk(len(validDiskDist3_1) - 1)
		assertEqualDisks(t,
			flattenDisks(validDiskDist3_1),
			gd,
		)
		assertNoErr(t, err)
	})

	t.Run("pentagon ok", func(t *testing.T) {
		t.Parallel()

		assertNoPanic(t, func() {
			disk, err := GridDisk(pentagonCell, 1)
			assertEqual(t, 6, len(disk), "expected pentagon disk to have 6 cells")
			assertNoErr(t, err)
		})
	})

	t.Run("invalid cell", func(t *testing.T) {
		t.Parallel()

		c := Cell(-1)
		_, err := c.GridDisk(1)
		assertErr(t, err)
		assertErrIs(t, err, ErrCellInvalid)
	})
}

func TestGridDiskDistances(t *testing.T) {
	t.Parallel()

	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		rings, err := validCell.GridDiskDistances(len(validDiskDist3_1) - 1)
		assertNoErr(t, err)
		assertEqualDiskDistances(t, validDiskDist3_1, rings)
	})

	t.Run("pentagon centered", func(t *testing.T) {
		t.Parallel()
		assertNoPanic(t, func() {
			rings, err := GridDiskDistances(pentagonCell, 1)
			assertNoErr(t, err)
			assertEqual(t, 2, len(rings), "expected 2 rings")
			assertEqual(t, 5, len(rings[1]), "expected 5 cells in second ring")
		})
	})

	t.Run("invalid k-ring", func(t *testing.T) {
		rings, err := GridDiskDistances(pentagonCell, -1)
		assertErr(t, err)
		assertErrIs(t, err, ErrDomain)
		assertNil(t, rings)
	})
}

func TestIsValid(t *testing.T) {
	t.Parallel()
	assertTrue(t, validCell.IsValid())
	assertFalse(t, Cell(0).IsValid())
}

func TestRoundtrip(t *testing.T) {
	t.Parallel()

	t.Run("latlng", func(t *testing.T) {
		t.Parallel()
		expectedGeo := LatLng{Lat: 1, Lng: 2}
		c, _ := LatLngToCell(expectedGeo, MaxResolution)
		actualGeo, _ := CellToLatLng(c)
		assertEqualLatLng(t, expectedGeo, actualGeo)

		expectedCell, _ := expectedGeo.Cell(MaxResolution)
		expectedLatLng, _ := expectedCell.LatLng()
		assertEqualLatLng(t, expectedGeo, expectedLatLng)
	})

	t.Run("cell", func(t *testing.T) {
		t.Parallel()
		geo, _ := CellToLatLng(validCell)
		actualCell, _ := LatLngToCell(geo, validCell.Resolution())
		assertEqual(t, validCell, actualCell)
	})
}

func TestResolution(t *testing.T) {
	t.Parallel()

	for i := 1; i <= MaxResolution; i++ {
		c, _ := LatLngToCell(validLatLng1, i)
		assertEqual(t, i, c.Resolution())
	}

	for _, e := range validCell.DirectedEdges() {
		assertEqual(t, validCell.Resolution(), e.Resolution())
	}
}

func TestBaseCellNumber(t *testing.T) {
	t.Parallel()
	bcID := validCell.BaseCellNumber()
	assertEqual(t, 6, bcID)
}

func TestParent(t *testing.T) {
	t.Parallel()
	// get the index's parent by requesting that index's resolution+1
	parent := validCell.ImmediateParent()

	// get the children at the resolution of the original index
	children := parent.ImmediateChildren()

	assertCellIn(t, validCell, children)
}

func TestCompactCells(t *testing.T) {
	t.Parallel()

	in := flattenDisks(validDiskDist3_1[:2])
	t.Logf("in: %v", in)
	out := CompactCells(in)
	t.Logf("out: %v", in)
	assertEqual(t, 1, len(out))
	assertEqual(t, validDiskDist3_1[0][0].ImmediateParent(), out[0])
}

func TestUncompactCells(t *testing.T) {
	t.Parallel()
	// get the index's parent by requesting that index's resolution+1
	parent := validCell.ImmediateParent()
	out := UncompactCells([]Cell{parent}, parent.Resolution()+1)
	assertCellIn(t, validCell, out)
}

func TestChildPosToCell(t *testing.T) {
	t.Parallel()

	children := validCell.Children(6)

	assertEqual(t, children[0], validCell.ChildPosToCell(0, 6))
	assertEqual(t, children[0], ChildPosToCell(0, validCell, 6))
}

func TestChildPos(t *testing.T) {
	t.Parallel()

	children := validCell.Children(7)

	assertEqual(t, 32, children[32].ChildPos(validCell.Resolution()))
	assertEqual(t, 32, CellToChildPos(children[32], validCell.Resolution()))
}

func TestIsResClassIII(t *testing.T) {
	t.Parallel()

	assertTrue(t, validCell.IsResClassIII())
	assertFalse(t, validCell.ImmediateParent().IsResClassIII())
}

func TestIsPentagon(t *testing.T) {
	t.Parallel()
	assertFalse(t, validCell.IsPentagon())
	assertTrue(t, pentagonCell.IsPentagon())
}

func TestIsNeighbor(t *testing.T) {
	t.Parallel()
	assertFalse(t, validCell.IsNeighbor(pentagonCell))
	assertTrue(t, validCell.DirectedEdges()[0].Destination().IsNeighbor(validCell))
}

func TestDirectedEdge(t *testing.T) {
	t.Parallel()

	origin := validDiskDist3_1[1][0]
	destination := origin.DirectedEdges()[0].Destination()
	edge := origin.DirectedEdge(destination)

	t.Run("is valid", func(t *testing.T) {
		t.Parallel()
		assertTrue(t, edge.IsValid())
		assertFalse(t, DirectedEdge(validCell).IsValid())
	})

	t.Run("get origin/destination from edge", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, origin, edge.Origin())
		assertEqual(t, destination, edge.Destination())

		// shadow origin/destination
		cells := edge.Cells()
		origin, destination := cells[0], cells[1]
		assertEqual(t, origin, edge.Origin())
		assertEqual(t, destination, edge.Destination())
	})

	t.Run("get edges from hexagon", func(t *testing.T) {
		t.Parallel()
		edges := validCell.DirectedEdges()
		assertEqual(t, 6, len(edges), "hexagon has 6 edges")
	})

	t.Run("get edges from pentagon", func(t *testing.T) {
		t.Parallel()
		edges := pentagonCell.DirectedEdges()
		assertEqual(t, 5, len(edges), "pentagon has 5 edges")
	})

	t.Run("get boundary from edge", func(t *testing.T) {
		t.Parallel()
		gb := edge.Boundary()
		assertEqual(t, 2, len(gb), "edge has 2 boundary cells")
	})
}

func TestStrings(t *testing.T) {
	t.Parallel()

	t.Run("bad string", func(t *testing.T) {
		t.Parallel()
		i := IndexFromString("oops")
		assertEqual(t, 0, i)
	})

	t.Run("good string round trip", func(t *testing.T) {
		t.Parallel()
		i := IndexFromString(validCell.String())

		//nolint:gosec // IndexFromString returns uint64 and fixing that to detect integer overflows will break package API. Let's skip it for now.
		assertEqual(t, validCell, Cell(i))
	})

	t.Run("no 0x prefix", func(t *testing.T) {
		t.Parallel()
		h3addr := validCell.String()
		assertEqual(t, "850dab63fffffff", h3addr)
	})

	t.Run("marshalling text", func(t *testing.T) {
		t.Parallel()
		c := Cell(0)
		text, err := validCell.MarshalText()
		assertNoErr(t, err)

		err = c.UnmarshalText([]byte("0x" + string(text)))
		assertNoErr(t, err)
		assertEqual(t, validCell, c)

		err = c.UnmarshalText([]byte(""))
		assertErr(t, err)
	})
}

func TestPolygonToCells(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		cells := PolygonToCells(GeoPolygon{}, 6)
		assertEqual(t, 0, len(cells))
	})

	t.Run("without holes", func(t *testing.T) {
		t.Parallel()
		cells := validGeoPolygonNoHoles.Cells(6)
		expectedIndexes := []Cell{
			0x860dab607ffffff,
			0x860dab60fffffff,
			0x860dab617ffffff,
			0x860dab61fffffff,
			0x860dab627ffffff,
			0x860dab62fffffff,
			0x860dab637ffffff,
		}
		assertEqualCells(t, expectedIndexes, cells)
	})

	t.Run("with hole", func(t *testing.T) {
		t.Parallel()
		cells := validGeoPolygonHoles.Cells(6)
		expectedIndexes := []Cell{
			0x860dab60fffffff,
			0x860dab617ffffff,
			0x860dab61fffffff,
			0x860dab627ffffff,
			0x860dab62fffffff,
			0x860dab637ffffff,
		}
		assertEqualCells(t, expectedIndexes, cells)
	})
}

func TestCellsToMultiPolygon(t *testing.T) {
	t.Parallel()

	// 7 cells in disk -> 1 polygon, 18-point loop, and no holes
	c, _ := LatLngToCell(NewLatLng(0, 0), 10)
	cells, _ := GridDisk(c, 1)
	res := CellsToMultiPolygon(cells)
	assertEqual(t, len(res), 1)
	assertEqual(t, len(res[0].GeoLoop), 18)
	assertEqual(t, len(res[0].Holes), 0)

	// 6 cells in ring -> 1 polygon, 18-point loop, and 1 6-point hole
	c, _ = LatLngToCell(NewLatLng(0, 0), 10)
	cells, _ = GridDisk(c, 1)
	res = CellsToMultiPolygon(cells[1:])
	assertEqual(t, len(res), 1)
	assertEqual(t, len(res[0].GeoLoop), 18)
	assertEqual(t, len(res[0].Holes), 1)
	assertEqual(t, len(res[0].Holes[0]), 6)

	// 2 hexagons connected -> 1 polygon, 10-point loop (2 shared points) and no holes
	c, _ = LatLngToCell(NewLatLng(0, 0), 10)
	cells, _ = GridDisk(c, 1)
	res = CellsToMultiPolygon(cells[:2])
	assertEqual(t, len(res), 1)
	assertEqual(t, len(res[0].GeoLoop), 10)
	assertEqual(t, len(res[0].Holes), 0)

	// 2 distinct disks -> 2 polygons, 2 18-point loops, and no holes
	c, _ = LatLngToCell(NewLatLng(0, 0), 10)
	cells1, _ := GridDisk(c, 1)

	c, _ = LatLngToCell(NewLatLng(10, 10), 10)
	cells2, _ := GridDisk(c, 1)
	cells = append(cells1, cells2...)
	res = CellsToMultiPolygon(cells)
	assertEqual(t, len(res), 2)
	assertEqual(t, len(res[0].GeoLoop), 18)
	assertEqual(t, len(res[0].Holes), 0)
	assertEqual(t, len(res[1].GeoLoop), 18)
	assertEqual(t, len(res[1].Holes), 0)

	// empty
	res = CellsToMultiPolygon([]Cell{})
	assertEqual(t, len(res), 0)
}

func TestGridPath(t *testing.T) {
	t.Parallel()
	path, err := lineStartCell.GridPath(lineEndCell)

	assertNoErr(t, err)
	assertEqual(t, lineStartCell, path[0])
	assertEqual(t, lineEndCell, path[len(path)-1])

	for i := 0; i < len(path)-1; i++ {
		assertTrue(t, path[i].IsNeighbor(path[i+1]))
	}

	path, err = GridPath(1, -1)
	assertErr(t, err)
	assertErrIs(t, err, ErrRsolutionMismatch)
	assertNil(t, path)

	c1, _ := NewLatLng(1, 1).Cell(5)
	c2, _ := NewLatLng(50.10320148224132, -143.47849001502516).Cell(5)
	path, err = GridPath(c1, c2)
	assertErr(t, err)
	assertErrIs(t, err, ErrFailed)
	assertNil(t, path)
}

func TestHexAreaKm2(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		assertEqualEps(t, float64(4357449.4161), HexagonAreaAvgKm2(0))
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		assertEqualEps(t, float64(0.0000009), HexagonAreaAvgKm2(15))
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		assertEqualEps(t, float64(0.7373276), HexagonAreaAvgKm2(8))
	})
}

func TestHexAreaM2(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		assertEqualEps(t, float64(4357449416078.3901), HexagonAreaAvgM2(0))
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		assertEqualEps(t, float64(0.8953), HexagonAreaAvgM2(15))
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		assertEqualEps(t, float64(737327.5976), HexagonAreaAvgM2(8))
	})
}

func TestPointDistRads(t *testing.T) {
	t.Parallel()
	distance := GreatCircleDistanceRads(validLatLng1, validLatLng2)
	assertEqualEps(t, float64(0.6796147656451452), distance)
}

func TestPointDistKm(t *testing.T) {
	t.Parallel()
	distance := GreatCircleDistanceKm(validLatLng1, validLatLng2)
	assertEqualEps(t, float64(4329.830552183446), distance)
}

func TestPointDistM(t *testing.T) {
	t.Parallel()
	distance := GreatCircleDistanceM(validLatLng1, validLatLng2)
	assertEqualEps(t, float64(4329830.5521834465), distance)
}

func TestCellAreaRads2(t *testing.T) {
	t.Parallel()
	assertEqualEps(t, float64(0.000006643967854567278), CellAreaRads2(validCell))
}

func TestCellAreaKm2(t *testing.T) {
	t.Parallel()
	assertEqualEps(t, float64(269.6768779509321), CellAreaKm2(validCell))
}

func TestCellAreaM2(t *testing.T) {
	t.Parallel()
	assertEqualEps(t, float64(269676877.95093215), CellAreaM2(validCell))
}

func TestHexagonEdgeLengthKm(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, float64(1107.712591), HexagonEdgeLengthAvgKm(0))
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, float64(0.000509713), HexagonEdgeLengthAvgKm(15))
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, float64(0.461354684), HexagonEdgeLengthAvgKm(8))
	})
}

func TestHexagonEdgeLengthM(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area := HexagonEdgeLengthAvgM(0)
		assertEqual(t, float64(1107712.591), area)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area := HexagonEdgeLengthAvgM(15)
		assertEqual(t, float64(0.509713273), area)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area := HexagonEdgeLengthAvgM(8)
		assertEqual(t, float64(461.3546837), area)
	})
}

func TestEdgeLengthRads(t *testing.T) {
	t.Parallel()
	assertEqualEps(t, float64(0.001569665746947077), EdgeLengthRads(validEdge))
}

func TestEdgeLengthKm(t *testing.T) {
	t.Parallel()

	distance := EdgeLengthKm(validEdge)
	assertEqualEps(t, float64(10.00035174544159), distance)
}

func TestEdgeLengthM(t *testing.T) {
	t.Parallel()

	distance := EdgeLengthM(validEdge)
	assertEqualEps(t, float64(10000.351745441589), distance)
}

func TestNumCells(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, 122, NumCells(0))
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, 569707381193162, NumCells(15))
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		assertEqual(t, 691776122, NumCells(8))
	})
}

func TestRes0Cells(t *testing.T) {
	t.Parallel()
	cells := Res0Cells()

	assertEqual(t, 122, len(cells))
	assertEqual(t, Cell(0x8001fffffffffff), cells[0])
	assertEqual(t, Cell(0x80f3fffffffffff), cells[121])
}

func TestGridDistance(t *testing.T) {
	t.Parallel()

	dist, err := lineStartCell.GridDistance(lineEndCell)
	assertEqual(t, 1823, dist)
	assertNoErr(t, err)

	_, err = GridDistance(-1, -2)
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestCenterChild(t *testing.T) {
	t.Parallel()

	child := validCell.CenterChild(15)
	assertEqual(t, Cell(0x8f0dab600000000), child)
}

func TestIcosahedronFaces(t *testing.T) {
	t.Parallel()

	faces, err := validDiskDist3_1[1][1].IcosahedronFaces()

	assertEqual(t, 1, len(faces))
	assertEqual(t, 1, faces[0])
	assertNoErr(t, err)

	c := Cell(-1)

	_, err = c.IcosahedronFaces()
	assertErr(t, err)
	assertErrIs(t, err, ErrCellInvalid)
}

func TestPentagons(t *testing.T) {
	t.Parallel()

	for _, res := range []int{0, 8, 15} {
		res := res
		t.Run(fmt.Sprintf("res=%d", res), func(t *testing.T) {
			t.Parallel()

			pentagons := Pentagons(res)
			assertEqual(t, 12, len(pentagons))

			for _, pentagon := range pentagons {
				assertTrue(t, pentagon.IsPentagon())
				assertEqual(t, res, pentagon.Resolution())
			}
		})
	}
}

func TestCellToVertex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cell           Cell
		expectedVertex Cell
		vertexNum      int
	}{
		{cell: validCell, expectedVertex: 0x2050dab63fffffff, vertexNum: 0},
		{cell: validCell, expectedVertex: 0, vertexNum: 6}, // vertex num should be between 0 and 5 for hexagonal cells.
	}

	for i, tc := range testCases {
		tc := tc

		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()

			vertex := CellToVertex(tc.cell, tc.vertexNum)
			assertEqual(t, tc.expectedVertex, vertex)
		})
	}
}

func TestCellToVertexes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cell        Cell
		numVertexes int
	}{
		{cell: validCell, numVertexes: 6},
		{cell: pentagonCell, numVertexes: 5},
		{cell: -1, numVertexes: 0}, // Invalid cel.
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprint(tc.numVertexes), func(t *testing.T) {
			t.Parallel()

			vertexes := CellToVertexes(tc.cell)
			assertEqual(t, tc.numVertexes, len(vertexes))
		})
	}
}

func TestVertexToLatLng(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		vertex         Cell
		expectedLatLng LatLng
	}{
		{vertex: CellToVertex(validCell, 0), expectedLatLng: LatLng{Lat: 67.22475, Lng: -168.52301}},
		{vertex: -1, expectedLatLng: LatLng{}}, // Invalid vertex.
	}

	for i, tc := range testCases {
		tc := tc

		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()

			latLng := VertexToLatLng(tc.vertex)
			assertEqualLatLng(t, tc.expectedLatLng, latLng)
		})
	}
}

func TestIsValidVertex(t *testing.T) {
	t.Parallel()

	assertFalse(t, IsValidVertex(0))
	assertTrue(t, IsValidVertex(2473183460575936511))
}

func equalEps(expected, actual float64) bool {
	return math.Abs(expected-actual) < eps
}

func assertErr(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func assertErrIs(t *testing.T, err, target error) {
	t.Helper()

	if errors.Is(err, target) {
		return
	}

	t.Errorf("errors don't match, err: %s, target err: %s", err, target)
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func assertEqual[T comparable](t *testing.T, expected, actual T, msgAndArgs ...interface{}) {
	t.Helper()

	if expected != actual {
		var (
			expStr, actStr string

			e, a interface{} = expected, actual
		)

		switch e.(type) {
		case Cell:
			expStr = fmt.Sprintf("%s (res=%d)", e.(Cell), e.(Cell).Resolution())
			actStr = fmt.Sprintf("%s (res=%d)", a.(Cell), a.(Cell).Resolution())
		default:
			expStr = fmt.Sprintf("%v", e)
			actStr = fmt.Sprintf("%v", a)
		}

		t.Errorf("%v != %v", expStr, actStr)
		logMsgAndArgs(t, msgAndArgs...)
	}
}

func assertEqualEps(t *testing.T, expected, actual float64, msgAndArgs ...interface{}) {
	t.Helper()

	if !equalEps(expected, actual) {
		t.Errorf("%0.4f > %v (%0.4f - %0.4f)", math.Abs(expected-actual), eps, expected, actual)
		logMsgAndArgs(t, msgAndArgs...)
	}
}

func assertEqualLatLng(t *testing.T, expected, actual LatLng) {
	t.Helper()
	assertEqualEps(t, expected.Lat, actual.Lat, "latitude mismatch")
	assertEqualEps(t, expected.Lng, actual.Lng, "longitude mismatch")
}

func assertEqualLatLngs(t *testing.T, expected, actual []LatLng, msgAndArgs ...interface{}) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("length mismatch: %v != %v", len(expected), len(actual))
		logMsgAndArgs(t, msgAndArgs...)

		return
	}

	count := 0

	for i, ll := range expected {
		equalLat := equalEps(ll.Lat, actual[i].Lat)
		equalLng := equalEps(ll.Lng, actual[i].Lng)

		if !equalLat || !equalLng {
			latStr := tern(equalLat, fmt.Sprintf("%v", ll.Lat), fmt.Sprintf("%v != %v", ll.Lat, actual[i].Lat))
			lngStr := tern(equalLng, fmt.Sprintf("%v", ll.Lng), fmt.Sprintf("%v != %v", ll.Lng, actual[i].Lng))

			t.Errorf("LatLngs[%d]: (%s, %s)", i, latStr, lngStr)
			logMsgAndArgs(t, msgAndArgs...)

			count++

			if count > 10 {
				t.Log("...and more")
				break
			}
		}
	}
}

func assertEqualCells(t *testing.T, expected, actual []Cell, msgAndArgs ...interface{}) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("length mismatch: %v != %v", len(expected), len(actual))
		logMsgAndArgs(t, msgAndArgs...)

		return
	}

	expected = sortCells(copyCells(expected))
	actual = sortCells(copyCells(actual))

	count := 0

	for i, c := range expected {
		if c != actual[i] {
			t.Errorf("Cells[%d]: %v != %v", i, c, actual[i])
			logMsgAndArgs(t, msgAndArgs...)

			count++

			if count > 10 {
				t.Log("...and more")
				break
			}
		}
	}
}

func assertEqualDiskDistances(t *testing.T, expected, actual [][]Cell) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("number of rings mismatch: %v != %v", len(expected), len(actual))
		return
	}

	expected = copyRings(expected)
	actual = copyRings(actual)

	for i := range expected {
		if len(expected[i]) != len(actual[i]) {
			t.Errorf("ring[%d] length mismatch: %v != %v", i, len(expected[i]), len(actual[i]))
			return
		}

		expected[i] = sortCells(expected[i])
		actual[i] = sortCells(actual[i])

		for j, cell := range expected[i] {
			if cell != actual[i][j] {
				t.Errorf("ring[%d][%d] mismatch: %v != %v", i, j, cell, actual[i][j])
				return
			}
		}
	}
}

func assertEqualDisks(t *testing.T, expected, actual []Cell) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("disk size mismatch: %v != %v", len(expected), len(actual))
		return
	}

	expected = sortCells(copyCells(expected))
	actual = sortCells(copyCells(actual))

	count := 0

	for i, cell := range expected {
		if cell != actual[i] {
			t.Errorf("cell[%d]: %v != %v", i, cell, actual[i])

			count++

			if count > 5 {
				t.Logf("... and more")
				break
			}
		}
	}
}

func assertCellIn(t *testing.T, needle Cell, haystack []Cell) {
	t.Helper()

	var found bool
	for _, h := range haystack {
		found = needle == h
		if found {
			break
		}
	}

	if !found {
		t.Errorf("%v not found in %+v", needle, haystack)
	}
}

func assertNoPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic: %v", r)
		}
	}()
	f()
}

func assertFalse(t *testing.T, b bool) {
	t.Helper()
	assertEqual(t, false, b)
}

func assertTrue(t *testing.T, b bool) {
	t.Helper()
	assertEqual(t, true, b)
}

func assertNil(t *testing.T, val any) {
	t.Helper()

	if val == nil {
		return
	}

	value := reflect.ValueOf(val)
	switch value.Kind() {
	case
		reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		if value.IsNil() {
			return
		}
	}

	t.Errorf("expected value to be nil, got: %v", val)
}

func sortCells(s []Cell) []Cell {
	sort.SliceStable(s, func(i, j int) bool {
		return s[i] < s[j]
	})

	return s
}

func logMsgAndArgs(t *testing.T, msgAndArgs ...interface{}) {
	t.Helper()

	if len(msgAndArgs) > 0 {
		t.Logf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
}

func flattenDisks(diskDist [][]Cell) []Cell {
	if len(diskDist) == 0 {
		return nil
	}

	flat := make([]Cell, 0, maxGridDiskSize(len(diskDist)-1))
	for _, disk := range diskDist {
		flat = append(flat, append([]Cell{}, disk...)...)
	}

	return flat
}

func tern(b bool, x, y string) string {
	if b {
		return x
	}

	return y
}

func copyRings(s [][]Cell) [][]Cell {
	c := make([][]Cell, len(s))
	copy(c, s)

	for i := range c {
		c[i] = append([]Cell{}, s[i]...)
	}

	return c
}

func copyCells(s []Cell) []Cell {
	c := make([]Cell, len(s))
	copy(c, s)

	return c
}
