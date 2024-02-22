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

	validEdge   = DirectedEdge(0x1250dab73fffffff)
	invalidEdge = DirectedEdge(0x175283773fffffff)
)

func TestLatLngToCell(t *testing.T) {
	t.Parallel()
	c, err := LatLngToCell(validLatLng1, 5)
	assertNoErr(t, err)
	assertEqual(t, validCell, c)
}
func TestLatLngToCellError(t *testing.T) {
	t.Parallel()
	_, err := LatLngToCell(validLatLng1, -1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))

	_, err = LatLngToCell(NewLatLng(math.Inf(1), math.Inf(1)), 5)
	assertTrue(t, errors.Is(err, ErrH3LatLngDomain))

	_, err = LatLngToCell(NewLatLng(math.Inf(-1), math.Inf(-1)), 5)
	assertTrue(t, errors.Is(err, ErrH3LatLngDomain))
}

func TestCellToLatLng(t *testing.T) {
	t.Parallel()
	g, err := CellToLatLng(validCell)
	assertNoErr(t, err)
	assertEqualLatLng(t, validLatLng1, g)
}
func TestCellToLatLngError(t *testing.T) {
	t.Parallel()
	_, err := CellToLatLng(Cell(0xfffffffffffffff))
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestToCellBoundary(t *testing.T) {
	t.Parallel()
	boundary, err := validCell.Boundary()
	assertNoErr(t, err)
	assertEqualLatLngs(t, validGeoLoop[:], boundary[:])
}
func TestToCellBoundaryError(t *testing.T) {
	t.Parallel()
	_, err := Cell(0xfffffffffffffff).Boundary()
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestGridDisk(t *testing.T) {
	t.Parallel()
	t.Run("no pentagon", func(t *testing.T) {
		t.Parallel()
		disks, err := validCell.GridDisk(len(validDiskDist3_1) - 1)
		assertNoErr(t, err)
		assertEqualDisks(t,
			flattenDisks(validDiskDist3_1),
			disks,
		)
	})
	t.Run("pentagon ok", func(t *testing.T) {
		t.Parallel()
		assertNoPanic(t, func() {
			disk, err := GridDisk(pentagonCell, 1)
			assertNoErr(t, err)
			assertEqual(t, 6, len(disk), "expected pentagon disk to have 6 cells")
		})
	})
}
func TestGridDiskError(t *testing.T) {
	t.Parallel()
	_, err := GridDisk(Cell(0xfffffffffffffff), 1)
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))

	_, err = GridDisk(validCell, -1)
	assertTrue(t, errors.Is(err, ErrH3Domain))
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
}
func TestGridDiskDistancesError(t *testing.T) {
	t.Parallel()
	_, err := GridDiskDistances(Cell(0xfffffffffffffff), 1)
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))

	_, err = GridDiskDistances(validCell, -1)
	assertTrue(t, errors.Is(err, ErrH3Domain))
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
		c, _ = expectedGeo.Cell(MaxResolution)
		latlng, _ := c.LatLng()
		assertEqualLatLng(t, expectedGeo, latlng)
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
		c, err := LatLngToCell(validLatLng1, i)
		assertNoErr(t, err)
		assertEqual(t, i, c.Resolution())
	}

	edges, err := validCell.DirectedEdges()
	assertNoErr(t, err)

	for _, e := range edges {
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
	parent, errParent := validCell.ImmediateParent()

	// get the children at the resolution of the original index
	children, errChildren := parent.ImmediateChildren()

	assertNoErr(t, errParent)
	assertNoErr(t, errChildren)
	assertCellIn(t, validCell, children)
}
func TestParentError(t *testing.T) {
	t.Parallel()
	_, err := validCell.Parent(6)
	assertTrue(t, errors.Is(err, ErrH3ResMismatch))

	_, err = validCell.Parent(-1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
}

func TestCompactCells(t *testing.T) {
	t.Parallel()

	in := flattenDisks(validDiskDist3_1[:2])
	t.Logf("in: %v", in)

	out, err := CompactCells(in)
	t.Logf("out: %v", out)
	assertNoErr(t, err)
	assertEqual(t, 1, len(out))

	parentCell, err := validDiskDist3_1[0][0].ImmediateParent()
	assertNoErr(t, err)
	assertEqual(t, parentCell, out[0])
}
func TestCompactCellsError(t *testing.T) {
	t.Parallel()

	cell := Cell(0x863e35407ffffff)
	in := []Cell{}

	for i := 0; i < 8; i++ {
		in = append(in, cell)
	}

	_, err := CompactCells(in)
	assertTrue(t, errors.Is(err, ErrH3DupInput))
}

func TestUncompactCells(t *testing.T) {
	t.Parallel()
	// get the index's parent by requesting that index's resolution+1
	parent, err := validCell.ImmediateParent()
	assertNoErr(t, err)
	out, err := UncompactCells([]Cell{parent}, parent.Resolution()+1)
	assertNoErr(t, err)
	assertCellIn(t, validCell, out)
}
func TestUncompactCellsError(t *testing.T) {
	t.Parallel()

	in := validDiskDist3_1[1]

	_, err := UncompactCells(in, 3)
	assertTrue(t, errors.Is(err, ErrH3ResMismatch))

	_, err = UncompactCells(in, -1)
	assertTrue(t, errors.Is(err, ErrH3ResMismatch))

	_, err = UncompactCells(in, 16)
	assertTrue(t, errors.Is(err, ErrH3ResMismatch))
}

func TestChildPosToCell(t *testing.T) {
	t.Parallel()

	children, err := validCell.Children(6)
	assertNoErr(t, err)

	child, err := validCell.ChildPosToCell(0, 6)
	assertNoErr(t, err)
	assertEqual(t, children[0], child)

	child, err = ChildPosToCell(0, validCell, 6)
	assertNoErr(t, err)
	assertEqual(t, children[0], child)
}

func TestChildPos(t *testing.T) {
	t.Parallel()

	children, err := validCell.Children(7)
	assertNoErr(t, err)

	child, err := children[32].ChildPos(validCell.Resolution())
	assertNoErr(t, err)
	assertEqual(t, 32, child)

	child, err = CellToChildPos(children[32], validCell.Resolution())
	assertNoErr(t, err)
	assertEqual(t, 32, child)
}

func TestIsResClassIII(t *testing.T) {
	t.Parallel()

	parentCell, err := validCell.ImmediateParent()
	assertNoErr(t, err)
	assertTrue(t, validCell.IsResClassIII())
	assertFalse(t, parentCell.IsResClassIII())
}

func TestIsPentagon(t *testing.T) {
	t.Parallel()
	assertFalse(t, validCell.IsPentagon())
	assertTrue(t, pentagonCell.IsPentagon())
}

func TestIsNeighbor(t *testing.T) {
	t.Parallel()

	res, err := validCell.IsNeighbor(validDiskDist3_1[2][0])
	assertNoErr(t, err)
	assertFalse(t, res)

	edges, _ := validCell.DirectedEdges()
	dest, _ := edges[0].Destination()

	res, err = dest.IsNeighbor(validCell)
	assertNoErr(t, err)
	assertTrue(t, res)
}

func TestDirectedEdge(t *testing.T) {
	t.Parallel()

	origin := validDiskDist3_1[1][0]
	edges, err := origin.DirectedEdges()
	assertNoErr(t, err)

	destination, err := edges[0].Destination()
	assertNoErr(t, err)

	edge, err := origin.DirectedEdge(destination)
	assertNoErr(t, err)

	t.Run("is valid", func(t *testing.T) {
		t.Parallel()
		assertTrue(t, edge.IsValid())
		assertFalse(t, DirectedEdge(validCell).IsValid())
	})

	t.Run("get origin/destination from edge", func(t *testing.T) {
		t.Parallel()
		originExpected, err := edge.Origin()
		assertNoErr(t, err)
		assertEqual(t, origin, originExpected)

		destinationExpected, err := edge.Destination()
		assertNoErr(t, err)
		assertEqual(t, destination, destinationExpected)

		// shadow origin/destination
		cells, err := edge.Cells()
		assertNoErr(t, err)

		origin, destination := cells[0], cells[1]
		originExpected, err = edge.Origin()
		assertNoErr(t, err)
		assertEqual(t, origin, originExpected)

		destinationExpected, err = edge.Destination()
		assertNoErr(t, err)
		assertEqual(t, destination, destinationExpected)
	})

	t.Run("get edges from hexagon", func(t *testing.T) {
		t.Parallel()
		edges, err := validCell.DirectedEdges()
		assertNoErr(t, err)
		assertEqual(t, 6, len(edges), "hexagon has 6 edges")
	})

	t.Run("get edges from pentagon", func(t *testing.T) {
		t.Parallel()
		edges, err := pentagonCell.DirectedEdges()
		assertNoErr(t, err)
		assertEqual(t, 5, len(edges), "pentagon has 5 edges")
	})

	t.Run("get boundary from edge", func(t *testing.T) {
		t.Parallel()
		gb, err := edge.Boundary()
		assertNoErr(t, err)
		assertEqual(t, 2, len(gb), "edge has 2 boundary cells")
	})
}
func TestDirectedEdgeError(t *testing.T) {
	t.Parallel()

	_, err := validCell.DirectedEdge(validDiskDist3_1[2][0])
	assertTrue(t, errors.Is(err, ErrH3Neighbors))

	_, err = DirectedEdge(0).Origin()
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))

	_, err = DirectedEdge(0).Destination()
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))
	_, err = invalidEdge.Destination()
	assertTrue(t, errors.Is(err, ErrH3Failed))

	_, err = DirectedEdge(0).Cells()
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))
	_, err = invalidEdge.Cells()
	assertTrue(t, errors.Is(err, ErrH3Failed))

	_, err = invalidEdge.Boundary()
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))
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
		cells, err := PolygonToCells(GeoPolygon{}, 6)
		assertNoErr(t, err)
		assertEqual(t, 0, len(cells))
	})

	t.Run("without holes", func(t *testing.T) {
		t.Parallel()
		cells, err := validGeoPolygonNoHoles.Cells(6)
		expectedIndexes := []Cell{
			0x860dab607ffffff,
			0x860dab60fffffff,
			0x860dab617ffffff,
			0x860dab61fffffff,
			0x860dab627ffffff,
			0x860dab62fffffff,
			0x860dab637ffffff,
		}
		assertNoErr(t, err)
		assertEqualCells(t, expectedIndexes, cells)
	})

	t.Run("with hole", func(t *testing.T) {
		t.Parallel()
		cells, err := validGeoPolygonHoles.Cells(6)
		expectedIndexes := []Cell{
			0x860dab60fffffff,
			0x860dab617ffffff,
			0x860dab61fffffff,
			0x860dab627ffffff,
			0x860dab62fffffff,
			0x860dab637ffffff,
		}
		assertNoErr(t, err)
		assertEqualCells(t, expectedIndexes, cells)
	})
}
func TestPolygonToCellsError(t *testing.T) {
	t.Parallel()
	_, err := PolygonToCells(validGeoPolygonHoles, -1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))

	invalidGeoPolygon := GeoPolygon{
		GeoLoop: GeoLoop{
			{Lat: math.Inf(1), Lng: math.Inf(1)},
			{Lat: math.Inf(-1), Lng: math.Inf(-1)},
		},
		Holes: []GeoLoop{},
	}
	_, err = PolygonToCells(invalidGeoPolygon, 5)
	assertTrue(t, errors.Is(err, ErrH3Failed))
}

func TestGridPath(t *testing.T) {
	t.Parallel()
	path, err := lineStartCell.GridPath(lineEndCell)
	assertNoErr(t, err)
	assertEqual(t, lineStartCell, path[0])
	assertEqual(t, lineEndCell, path[len(path)-1])

	for i := 0; i < len(path)-1; i++ {
		res, _ := path[i].IsNeighbor(path[i+1])
		assertTrue(t, res)
	}
}

func TestHexAreaKm2(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgKm2(0)
		assertNoErr(t, err)
		assertEqualEps(t, float64(4357449.4161), area)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgKm2(15)
		assertNoErr(t, err)
		assertEqualEps(t, float64(0.0000009), area)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgKm2(8)
		assertNoErr(t, err)
		assertEqualEps(t, float64(0.7373276), area)
	})
}
func TestHexAreaKm2Error(t *testing.T) {
	t.Parallel()
	_, err := HexagonAreaAvgKm2(-1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))

	_, err = HexagonAreaAvgKm2(16)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
}

func TestHexAreaM2(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgM2(0)
		assertNoErr(t, err)
		assertEqualEps(t, float64(4357449416078.3901), area)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgM2(15)
		assertNoErr(t, err)
		assertEqualEps(t, float64(0.8953), area)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonAreaAvgM2(8)
		assertNoErr(t, err)
		assertEqualEps(t, float64(737327.5976), area)
	})
}
func TestHexAreaM2Error(t *testing.T) {
	t.Parallel()
	_, err := HexagonAreaAvgM2(-1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))

	_, err = HexagonAreaAvgM2(16)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
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
	area, err := CellAreaRads2(validCell)
	assertNoErr(t, err)
	assertEqualEps(t, float64(0.000006643967854567278), area)
}
func TestCellAreaRads2Error(t *testing.T) {
	t.Parallel()
	_, err := CellAreaRads2(Cell(0xfffffffffffffff))
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestCellAreaKm2(t *testing.T) {
	t.Parallel()
	area, err := CellAreaKm2(validCell)
	assertNoErr(t, err)
	assertEqualEps(t, float64(269.6768779509321), area)
}
func TestCellAreaKm2Error(t *testing.T) {
	t.Parallel()
	_, err := CellAreaKm2(Cell(0xfffffffffffffff))
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestCellAreaM2(t *testing.T) {
	t.Parallel()
	area, err := CellAreaM2(validCell)
	assertNoErr(t, err)
	assertEqualEps(t, float64(269676877.95093215), area)
}
func TestCellAreaM2Error(t *testing.T) {
	t.Parallel()
	_, err := CellAreaM2(Cell(0xfffffffffffffff))
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestHexagonEdgeLengthKm(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgKm(0)
		assertNoErr(t, err)
		assertEqual(t, float64(1107.712591), area)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgKm(15)
		assertNoErr(t, err)
		assertEqual(t, float64(0.000509713), area)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgKm(8)
		assertNoErr(t, err)
		assertEqual(t, float64(0.461354684), area)
	})
}
func TestHexagonEdgeLengthKmError(t *testing.T) {
	t.Parallel()
	_, err := HexagonEdgeLengthAvgKm(-1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))

	_, err = HexagonEdgeLengthAvgKm(16)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
}

func TestHexagonEdgeLengthM(t *testing.T) {
	t.Parallel()
	t.Run("min resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgM(0)
		assertNoErr(t, err)
		assertEqual(t, float64(1107712.591), area)
	})
	t.Run("max resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgM(15)
		assertNoErr(t, err)
		assertEqual(t, float64(0.509713273), area)
	})
	t.Run("mid resolution", func(t *testing.T) {
		t.Parallel()
		area, err := HexagonEdgeLengthAvgM(8)
		assertNoErr(t, err)
		assertEqual(t, float64(461.3546837), area)
	})
}
func TestHexagonEdgeLengthMError(t *testing.T) {
	t.Parallel()
	_, err := HexagonEdgeLengthAvgM(-1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))

	_, err = HexagonEdgeLengthAvgM(16)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
}

func TestEdgeLengthRads(t *testing.T) {
	t.Parallel()

	distance, err := EdgeLengthRads(validEdge)
	assertNoErr(t, err)
	assertEqualEps(t, float64(0.001569665746947077), distance)
}
func TestEdgeLengthRadsError(t *testing.T) {
	t.Parallel()
	_, err := EdgeLengthRads(DirectedEdge(0))
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))
}

func TestEdgeLengthKm(t *testing.T) {
	t.Parallel()

	distance, err := EdgeLengthKm(validEdge)
	assertNoErr(t, err)
	assertEqualEps(t, float64(10.00035174544159), distance)
}
func TestEdgeLengthKmError(t *testing.T) {
	t.Parallel()
	_, err := EdgeLengthRads(DirectedEdge(0))
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))
}

func TestEdgeLengthM(t *testing.T) {
	t.Parallel()

	distance, err := EdgeLengthM(validEdge)
	assertNoErr(t, err)
	assertEqualEps(t, float64(10000.351745441589), distance)
}
func TestEdgeLengthMError(t *testing.T) {
	t.Parallel()
	_, err := EdgeLengthRads(DirectedEdge(0))
	assertTrue(t, errors.Is(err, ErrH3InvalidDirEdge))
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
	cells, err := Res0Cells()

	assertNoErr(t, err)
	assertEqual(t, 122, len(cells))
	assertEqual(t, Cell(0x8001fffffffffff), cells[0])
	assertEqual(t, Cell(0x80f3fffffffffff), cells[121])
}

func TestGridDistance(t *testing.T) {
	t.Parallel()
	dist, err := lineStartCell.GridDistance(lineEndCell)
	assertNoErr(t, err)
	assertEqual(t, 1823, dist)
}
func TestGridDistanceError(t *testing.T) {
	t.Parallel()
	_, err := GridDistance(Cell(0x821c37fffffffff), Cell(0x822837fffffffff))
	assertTrue(t, errors.Is(err, ErrH3Failed))

	_, err = GridDistance(Cell(0x81283ffffffffff), Cell(0x8029fffffffffff))
	assertTrue(t, errors.Is(err, ErrH3ResMismatch))

	_, err = GridDistance(Cell(0xfffffffffffffff), Cell(0xfffffffffffffff))
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestCenterChild(t *testing.T) {
	t.Parallel()

	child, err := validCell.CenterChild(15)
	assertNoErr(t, err)
	assertEqual(t, Cell(0x8f0dab600000000), child)
}
func TestCenterChildError(t *testing.T) {
	t.Parallel()
	_, err := validCell.CenterChild(4)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
}

func TestIcosahedronFaces(t *testing.T) {
	t.Parallel()

	faces, err := validDiskDist3_1[1][1].IcosahedronFaces()

	assertNoErr(t, err)
	assertEqual(t, 1, len(faces))
	assertEqual(t, 1, faces[0])
}
func TestIcosahedronFacesError(t *testing.T) {
	t.Parallel()
	_, err := Cell(0xfffffffffffffff).IcosahedronFaces()
	assertTrue(t, errors.Is(err, ErrH3InvalidIndex))
}

func TestPentagons(t *testing.T) {
	t.Parallel()

	for _, res := range []int{0, 8, 15} {
		t.Run(fmt.Sprintf("res=%d", res), func(t *testing.T) {
			t.Parallel()
			pentagons, err := Pentagons(res)
			assertNoErr(t, err)
			assertEqual(t, 12, len(pentagons))
			for _, pentagon := range pentagons {
				assertTrue(t, pentagon.IsPentagon())
				assertEqual(t, res, pentagon.Resolution())
			}
		})
	}
}
func TestPentagonsError(t *testing.T) {
	t.Parallel()
	_, err := Pentagons(-1)
	assertTrue(t, errors.Is(err, ErrH3ResDomain))
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
